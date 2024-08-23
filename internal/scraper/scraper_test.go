package scraper

import (
	"scraper/internal/database"
	"testing"

	"github.com/go-rod/rod"
)

func TestScraper(t *testing.T) {
	browser, l := InitialiseBrowser()
	defer l.Cleanup()
	defer browser.MustClose()

	pool := rod.NewPagePool(PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	CloseBrowserOnForceExit(browser)

	t.Run("Testing get fund link", func(t *testing.T) {
		fundName := "AB FCP I Global Equity Blend"
		fundLink := "https://secure.fundsupermart.com/fsmone/funds/factsheet/ACM019"

		tableName := "testfunds"
		db := database.ConnectDB()

		database.CreateTestFundTable(db, tableName)

		fund := database.Fund{Fundname: fundName, Link: fundLink}
		database.AddFund(db, tableName, fund)

		page, _ := pool.Get(func() (*rod.Page, error) { return browser.MustIncognito().MustPage(), nil })
		defer pool.Put(page)

		page.MustNavigate(fundLink)
		err := checkFundName(fundName, page)

		if err != nil {
			results, err := database.FundsByNames(db, tableName, []string{fundName})
			if err != nil {
				t.Fatal(err)
			}
			t.Fatalf("Fund name is %s, expected fund name to be AB FCP I GLOBAL EQUITY BLEND A SGD", results[0].Fundname)
		}
	})
}
