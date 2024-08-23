package local

import (
	"log"
	"reflect"
	"scraper/internal/database"
	"testing"
)

func TestIntialise(t *testing.T) {

	t.Run("Testing", func(t *testing.T) {
		filepath := "TestPlanning.xlsx"
		fundNames := GetFundsOwned(filepath)
		for _, fundName := range fundNames {
			print(fundName)
		}
		print(len(fundNames))
		if fundNames[0] == "Fund Name" {
			t.Errorf("Do not include column name")
		}

		fund1 := database.Fund{Fundname: "fund1", Link: "link1"}
		fundsToAdd := []database.Fund{fund1, fund1}
		AddFunds(fundsToAdd, filepath, "Link")
		AddFunds(fundsToAdd, filepath, "Planning")
		err := UpdateFundName("fund1", "newfund1", filepath, "Link")
		if err != nil {
			t.Error(err)
		}

		funds, err := FundsByNames(filepath, "Link", []string{"newfund1", "fund2"})
		if err != nil {
			t.Error(err)
		}
		log.Printf("%+v", funds)

		fundsNotIn, err := FundsNotInNames(filepath, "Link", []string{"newfund1", "fund2"})
		if err != nil {
			t.Error(err)
		}

		if !reflect.DeepEqual(fundsNotIn, []string{"fund2"}) {
			t.Errorf("FundsNotInNames not workign properly, got %+v", fundsNotIn)
		}
	})
}
