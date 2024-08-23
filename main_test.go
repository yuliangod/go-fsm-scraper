package main

import (
	"reflect"
	"scraper/internal/database"
	"scraper/internal/local"
	"scraper/internal/scraper"
	"sort"
	"testing"

	"github.com/go-rod/rod"
)

func TestMainDB(t *testing.T) {
	tableName := "testfunds"
	db := database.ConnectDB()

	database.CreateTestFundTable(db, tableName)
	funds := []database.Fund{{Fundname: "fund1", Link: "link1"}, {Fundname: "fund2", Link: "link2"}, {Fundname: "fund3", Link: "link3"}}

	for _, fund := range funds {
		database.AddFund(db, tableName, fund)
	}

	browser, l := scraper.InitialiseBrowser()
	defer l.Cleanup()
	defer browser.MustClose()

	pool := rod.NewPagePool(scraper.PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	t.Run("Testing getting links", func(t *testing.T) {
		actualFunds := []database.Fund{{Fundname: "AB FCP I Global Equity Blend A SGD", Link: "https://secure.fundsupermart.com/fsmone/funds/factsheet/ACM019"}, {Fundname: "abrdn SICAV I - Asian Credit Sustainable Bond A Gross MIncA SGD-H", Link: "https://secure.fundsupermart.com/fsmone/funds/factsheet/ABD035"}}
		fundNames := []string{"fund1", "fund2", "fund3"}
		expected := funds

		// Add new actual funds to expected funds
		for _, fund := range actualFunds {
			fundNames = append(fundNames, fund.Fundname)
			expected = append(expected, fund)
		}

		gotFunds := getFundLinksDB(db, fundNames, tableName)
		var got []database.Fund
		for _, fund := range gotFunds {
			fund.ID = 0
			got = append(got, fund)
		}

		got = sortFunds(t, got)
		expected = sortFunds(t, expected)

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("Get fund links failed. Expected: %+v, Got: %+v", expected, got)
		}
	})
}

func TestMainLocal(t *testing.T) {
	filepath := "internal/local/TestPlanning.xlsx"

	funds := []database.Fund{{Fundname: "fund1", Link: "link1"}, {Fundname: "fund2", Link: "link2"}, {Fundname: "fund3", Link: "link3"}}

	local.AddFunds(funds, filepath, "Link")

	browser, l := scraper.InitialiseBrowser()
	defer l.Cleanup()
	defer browser.MustClose()

	pool := rod.NewPagePool(scraper.PoolLimit)
	defer pool.Cleanup(func(p *rod.Page) { p.MustClose() })

	t.Run("Testing getting links", func(t *testing.T) {
		actualFunds := []database.Fund{{Fundname: "AB FCP I Global Equity Blend A SGD", Link: "https://secure.fundsupermart.com/fsmone/funds/factsheet/ACM019"}, {Fundname: "abrdn SICAV I - Asian Credit Sustainable Bond A Gross MIncA SGD-H", Link: "https://secure.fundsupermart.com/fsmone/funds/factsheet/ABD035"}}
		fundNames := []string{"fund1", "fund2", "fund3"}
		expected := funds

		// Add new actual funds to expected funds
		for _, fund := range actualFunds {
			fundNames = append(fundNames, fund.Fundname)
			expected = append(expected, fund)
		}

		gotFunds := getFundLinksLocal(filepath, fundNames)
		var got []database.Fund
		for _, fund := range gotFunds {
			fund.ID = 0
			got = append(got, fund)
		}

		got = sortFunds(t, got)
		expected = sortFunds(t, expected)

		if !reflect.DeepEqual(got, expected) {
			t.Fatalf("Get fund links failed. Expected: %+v, Got: %+v", expected, got)
		}
	})
}

func sortFunds(t testing.TB, funds []database.Fund) []database.Fund {
	t.Helper()
	sort.Slice(funds, func(i, j int) bool {
		return funds[i].Fundname > funds[j].Fundname
	})
	return funds
}
