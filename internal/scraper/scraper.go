package scraper

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"scraper/internal/database"
	"scraper/internal/scraper/persiststate"
	"strings"
	"sync"
	"syscall"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/rod/lib/utils"
	"github.com/micmonay/keybd_event"
)

const (
	FSMfundSelectorSite = "https://secure.fundsupermart.com/fsmone/tools/fund-selector"
	PoolLimit           = 5 //Number of pages that can be loaded concurrently
	pref                = `{
		"download": {
		  "default_directory": "/Users/yad/Downloads/ttt"
		}
	  }`
)

type ConcBrowser struct {
	Browser *rod.Browser
	Counter int
	MU      sync.Mutex
}

func ScrapeFSM(fund database.Fund, browser *rod.Browser, pool *rod.Pool[rod.Page], pageCookies, browserCookies []*proto.NetworkCookie, sessionStorage, localStorage persiststate.StorageData, c *ConcBrowser, fullhist bool, downloadFolderPath string) {
	page, err := pool.Get(func() (*rod.Page, error) { return browser.MustPage(), nil }) //Create a new page in page pool, must use .MustIncognito for concurrency
	defer pool.Put(page)

	if err != nil {
		log.Fatalf("Error creating new page from pool")
	} else {
		log.Println("Starting scrape for", fund.Fundname)
	}

	//page.MustNavigate(FSMfundSelectorSite).MustWaitLoad()
	//setSessionData(page, pageCookies, sessionStorage, localStorage)
	//page.MustReload().MustWaitLoad()

	//Switch to new fund page button
	//fundLink := findFundLink(fundName, page)
	page.MustNavigate(fund.Link)
	//if err != nil {
	//	log.Fatalf("Couldn't open fund page for: %s", fundName)
	//} else {
	//	log.Println("Opened fund page for:", fundName)
	//}

	//defer fundPage.Close()

	downloadFromFundPage(fund.Fundname, page, c, fullhist, downloadFolderPath)
}

func InitialiseBrowser() (*rod.Browser, *launcher.Launcher) {
	// Settings to launch browser non-headless
	l := launcher.New().
		Preferences(pref).
		Headless(false).
		Devtools(false)
		//Set("download.default_directory", "C:/Users/Acer/Downloads").
		//Set("download.prompt_for_download", "false").
		//Set("profile.default_content_settings.popups", "0").
		//Set("disable-popup-blocking", "true")

	url := l.MustLaunch()

	// Launch a browser with non-headless settings
	browser := rod.New().
		ControlURL(url).
		Trace(false). //disable verbose tracking of actions
		//SlowMotion(2 * time.Second).
		MustConnect()

	return browser, l
}

func LoginSteps(pool *rod.Pool[rod.Page], browser *rod.Browser) (pageCookies, browserCookies []*proto.NetworkCookie, sessionStorage, localStorage persiststate.StorageData) {
	//Refresh page once to get rid of annoying popups
	page, _ := pool.Get(func() (*rod.Page, error) { return browser.MustPage(), nil })
	defer pool.Put(page)

	page.MustNavigate(FSMfundSelectorSite).MustWaitLoad()
	page.MustElementX("//span[@aria-hidden='true']").MustClick()
	page.MustNavigate("https://secure.fundsupermart.com/fsm/account/login").MustWaitStable()

	//Wait for user to login to FSM account before hitting enter into the terminal
	var i string
	fmt.Print("Input any random characters and hit enter after logging in, REMEMBER TO FULLY ZOOM OUT OF BROWSER WINDOW: ")
	fmt.Scan(&i)

	//Save cookies so subsequent pages do not need to relogin and clear annoying popup
	pageCookies, _ = page.Cookies(make([]string, 0))
	browserCookies, _ = browser.GetCookies()
	log.Printf("Number of page cookes: %d", len(pageCookies))
	for i, cookie := range pageCookies {
		log.Printf("%d. Cookie Name:%s, Value: %s", i, cookie.Name, cookie.Value)
	}
	log.Printf("Number of browser cookes: %d", len(browserCookies))
	for i, cookie := range browserCookies {
		log.Printf("%d. Cookie Name:%s, Value: %s", i, cookie.Name, cookie.Value)
	}

	sessionStorage = persiststate.ExtractStorageData(page, "sessionStorage")
	localStorage = persiststate.ExtractStorageData(page, "localStorage")

	log.Print("Setting local storage to login state")
	for key, value := range localStorage {
		log.Printf("Local storage Key: %s, Value: %s", key, value)
	}

	log.Print("Setting session storage to login state")
	for key, value := range sessionStorage {
		log.Printf("Session storage Key: %s, Value: %s", key, value)
	}

	return pageCookies, browserCookies, sessionStorage, localStorage
}

func CloseBrowserOnForceExit(browser *rod.Browser) {
	// Setup signal handling to close the browser on SIGINT and SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs

		browser.MustClose()
		log.Fatalf("Received signal: %s, closing browser", sig)
	}()
}

func FindFundLink(fundName string, page *rod.Page) string {
	//Enter fund name into search bar
	searchBar := page.MustElement(`input[placeholder="Search"]`)
	searchBar.SelectAllText()
	searchBar.MustInput(fundName)

	//Find fund page button
	log.Println("Searching for fund page button:", fundName)
	fundPageButtonX := fmt.Sprintf("//span[contains(text(), '%s')]", fundName)
	fundPageButton := page.MustElementX(fundPageButtonX).MustParent()

	log.Println("Element found for fund page button:", fundName)
	//openFundPageFunc := page.WaitOpen()
	//page.Mouse.Scroll(0, 100, 1)
	//fundPageButton.MustClick()
	link, err := fundPageButton.Attribute("href")
	fundLink := *link

	fundLink = fmt.Sprintf("https://secure.fundsupermart.com%s", fundLink)
	log.Printf("Fund link for %s grabbed: %s", fundName, fundLink)
	if err != nil {
		log.Fatalf("Failed to get fund link: %s", fundLink)
	}

	return fundLink
}

func downloadFromFundPage(fundName string, fundPage *rod.Page, c *ConcBrowser, fullhist bool, downloadFolderPath string) {
	err := checkFundName(fundName, fundPage)
	if err != nil {
		log.Fatal(err)
	}

	c.MU.Lock()
	fundPage.Activate()

	//Export CSV
	priceButton := fundPage.MustElementX("//span[normalize-space(text())='Price']")
	priceButton.MustClick()

	// Click on 10Y if fullhist is true
	if fullhist {
		fundPage.MustElementX("//span[normalize-space(text())='Price']") //wait for price button to appear again

		moreButton := fundPage.MustElementX("//span[contains(text(), 'More')]").MustParent().MustParent().MustParent()
		moreButton.MustClick()

		tenButton := fundPage.MustElementX("//div[contains(text(), '10Y')]")
		tenButton.MustClick()
	}

	wait := c.Browser.MustWaitDownload()

	//Input keyboard enter into system to trigger download from popup window
	exportButton := fundPage.MustElementX("//span[normalize-space(text())='Export']")
	exportButton.MustClick()

	//time.Sleep(2 * time.Second)
	//pressEnterKey()

	err = utils.OutputFile(fmt.Sprintf("%s/%s.csv", downloadFolderPath, strings.ReplaceAll(fundName, "/", "")), wait())
	if err != nil {
		log.Fatal(err)
	}
	c.MU.Unlock()

	log.Println(fundName, "successfully downloaded")
}

func checkFundName(fundName string, fundPage *rod.Page) error {
	fundPageName := fundPage.MustElementX("//div[@class='flex flex-col items-start']/div/div").MustText()
	if strings.EqualFold(strings.ReplaceAll(fundPageName, " ", ""), strings.ReplaceAll(fundName, " ", "")) {
		log.Printf("Correct fund page opened for: %s", fundName)
	} else {
		//database.UpdateFundName(db, tableName, fundName, fundPageName)
		return fmt.Errorf("fund name has been updated from %s to %s", fundName, fundPageName)
	}

	return nil
}

func pressEnterKey() {
	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		log.Fatalf("Keyboard library not working properly")
	}
	kb.SetKeys(keybd_event.VK_ENTER)
	err = kb.Launching()
	if err != nil {
		log.Fatalf("Error pressing enter")
	}
}
