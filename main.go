package main

import (
	"database/sql"
	"log"
	"scraper/internal/database"
	"scraper/internal/local"
	"scraper/internal/scraper"
	"sync"

	"github.com/go-rod/rod"
)

const DOWNLOAD_ONLY_FROM_PLANNING_EXCEL = true

// shared settings
const (
	fullhist = false //false if only want 3 months of data, true if want full data on fsm website
)

// settings to download all funds data when download_only_from_planning_excel = true
const (
	planningRelativeFilepath = "Planning.xlsx"
)

// settings to download all funds data when download_only_from_planning_excel = false
const (
	tableName          = "funds"
	batchsize          = 1000 //290 seems to be the max limit to download in 1 session, decreases over time
	downloadWithinDays = 3
)

func main() {
	if DOWNLOAD_ONLY_FROM_PLANNING_EXCEL == true {
		main_local()
	} else {
		main_db()
	}
}

func main_db() {
	fundNames := local.GetAllFunds("export(1722502686274).xlsx")
	// Set up scraping tools
	browser, l := scraper.InitialiseBrowser()
	defer l.Cleanup()

	concBrowser := &scraper.ConcBrowser{Browser: browser}
	browser = concBrowser.Browser
	defer browser.MustClose()

	pool := rod.NewPagePool(scraper.PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	scraper.CloseBrowserOnForceExit(browser)

	db := database.ConnectDB()

	// Get fund links to directly scrape from fund page
	getFundLinksDB(db, fundNames, tableName)
	log.Print("Fund links successfully obtained")

	fundsNotDownloaded, err := database.FundsNotDownloadedWithinDays(db, tableName, downloadWithinDays)
	if err != nil {
		log.Fatal(err)
	}

	funds := fundsNotDownloaded[:min(batchsize, len(fundsNotDownloaded))]

	pageCookies, browserCookies, sessionStorage, localStorage := scraper.LoginSteps(&pool, browser)

	var wg sync.WaitGroup
	wg.Add(len(funds))

	for _, fund := range funds {
		go func() {
			defer wg.Done()
			//Close browser if any panic warnings are thrown
			defer func() {
				if r := recover(); r != nil {
					log.Fatalf("Panic: %v\n. ScrapeFSM function failed, exiting program", r)
				}
			}()

			scraper.ScrapeFSM(fund, browser, &pool, pageCookies, browserCookies, sessionStorage, localStorage, concBrowser, fullhist, "data/downloaded")
			database.UpdateLastDownloaded(db, tableName, fund.Fundname)

			concBrowser.Counter++
			log.Printf("%d/%d funds successfully downloaded, %d/%d total funds", concBrowser.Counter, batchsize, len(fundNames)-len(fundsNotDownloaded)+concBrowser.Counter, len(fundNames))
		}()
	}

	wg.Wait()
}

func main_local() {
	fundNames := local.GetFundsOwned("Planning.xlsx")
	// Set up scraping tools
	browser, l := scraper.InitialiseBrowser()
	defer l.Cleanup()

	concBrowser := &scraper.ConcBrowser{Browser: browser}
	browser = concBrowser.Browser
	defer browser.MustClose()

	pool := rod.NewPagePool(scraper.PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	scraper.CloseBrowserOnForceExit(browser)

	//db := database.ConnectDB()

	// Get fund links to directly scrape from fund page
	//funds := getFundLinks(db, fundNames, tableName)

	local.ClearFolder("data/planning")

	funds := getFundLinksLocal(planningRelativeFilepath, fundNames)
	log.Print("Fund links successfully obtained")

	pageCookies, browserCookies, sessionStorage, localStorage := scraper.LoginSteps(&pool, browser)

	var wg sync.WaitGroup
	wg.Add(len(fundNames))

	for _, fund := range funds {
		go func() {
			defer wg.Done()
			//Close browser if any panic warnings are thrown
			defer func() {
				if r := recover(); r != nil {
					log.Fatalf("Panic: %v\n. ScrapeFSM function failed, exiting program", r)
				}
			}()

			scraper.ScrapeFSM(fund, browser, &pool, pageCookies, browserCookies, sessionStorage, localStorage, concBrowser, fullhist, "data/planning")
			concBrowser.Counter++
			log.Printf("%d/%d funds successfully downloaded", concBrowser.Counter, len(fundNames))
		}()
	}

	wg.Wait()
}

func getFundLinksLocal(planningRelativeFilepath string, fundNames []string) []database.Fund {
	// Set up scraping tools
	browser, l := scraper.InitialiseBrowser()
	defer l.Cleanup()

	concBrowser := &scraper.ConcBrowser{Browser: browser}
	browser = concBrowser.Browser
	defer browser.MustClose()

	pool := rod.NewPagePool(scraper.PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	scraper.CloseBrowserOnForceExit(browser)

	fundsNotIn, err := local.FundsNotInNames(planningRelativeFilepath, "Link", fundNames)
	if err != nil {
		log.Fatal(err)
	}

	if len(fundsNotIn) == 0 {
		funds, err := local.FundsByNames(planningRelativeFilepath, "Link", fundNames)
		if err != nil {
			log.Fatal(err)
		}
		return funds
	}

	log.Printf("%s not in DB, starting scrape to get fund links", fundsNotIn)

	page := browser.MustPage()
	page.MustNavigate(scraper.FSMfundSelectorSite).MustWaitLoad()

	var fundsToAdd []database.Fund
	for _, fundName := range fundsNotIn {
		if err != nil {
			log.Fatalf("Error creating new page from pool")
		} else {
			log.Printf("Getting link for %s", fundName)
		}

		fundLink := scraper.FindFundLink(fundName, page)

		fundsToAdd = append(fundsToAdd, database.Fund{Fundname: fundName, Link: fundLink})
	}

	if len(fundsToAdd) != 0 {
		local.AddFunds(fundsToAdd, planningRelativeFilepath, "Link")
	}

	funds, err := local.FundsByNames(planningRelativeFilepath, "Link", fundNames)
	if err != nil {
		log.Fatal(err)
	}

	return funds
}

func getFundLinksDB(db *sql.DB, fundNames []string, tableName string) []database.Fund {
	// Set up scraping tools
	browser, l := scraper.InitialiseBrowser()
	defer l.Cleanup()

	concBrowser := &scraper.ConcBrowser{Browser: browser}
	browser = concBrowser.Browser
	defer browser.MustClose()

	pool := rod.NewPagePool(scraper.PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	scraper.CloseBrowserOnForceExit(browser)

	fundsNotIn, err := database.FundsNotInNames(db, tableName, fundNames)
	if err != nil {
		log.Fatal(err)
	}

	if len(fundsNotIn) == 0 {
		funds, err := database.FundsByNames(db, tableName, fundNames)
		if err != nil {
			log.Fatal(err)
		}
		return funds
	}

	createPages(browser, &pool)

	log.Printf("%s not in DB, starting scrape to get fund links", fundsNotIn)

	var wg sync.WaitGroup
	wg.Add(len(fundsNotIn))

	for _, fundName := range fundsNotIn {
		go func() {
			defer wg.Done()
			page, err := pool.Get(func() (*rod.Page, error) { return browser.MustIncognito().MustPage(), nil }) //Create a new page in page pool, must use .MustIncognito for concurrency
			defer pool.Put(page)

			if err != nil {
				log.Fatalf("Error creating new page from pool")
			} else {
				log.Printf("Getting link for %s", fundName)
			}

			fundLink := scraper.FindFundLink(fundName, page)

			database.AddFund(db, tableName, database.Fund{Fundname: fundName, Link: fundLink})
			concBrowser.Counter++
			log.Printf("%d/%d links successfully extracted", concBrowser.Counter, len(fundsNotIn))
		}()
	}

	wg.Wait()

	funds, err := database.FundsByNames(db, tableName, fundNames)
	if err != nil {
		log.Fatal(err)
	}

	return funds
}

func createPages(browser *rod.Browser, pool *rod.Pool[rod.Page]) {
	for i := 0; i < scraper.PoolLimit; i++ {
		page, err := pool.Get(func() (*rod.Page, error) { return browser.MustIncognito().MustPage(), nil }) //Create a new page in page pool, must use .MustIncognito for concurrency
		if err != nil {
			log.Fatal(err)
		}
		page.MustNavigate(scraper.FSMfundSelectorSite).MustWaitLoad()
		defer pool.Put(page)
	}
}
