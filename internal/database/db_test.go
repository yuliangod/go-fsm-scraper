package database

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDB(t *testing.T) {
	t.Run("Testing DB", func(t *testing.T) {
		tableName := "testfunds"
		db := ConnectDB()

		CreateTestFundTable(db, tableName)
		funds := []Fund{{Fundname: "fund1", Link: "link1"}, {Fundname: "fund2", Link: "link2"}, {Fundname: "fund3", Link: "link3"}}

		var addedFunds []Fund
		for _, fund := range funds {
			id := AddFund(db, tableName, fund)
			fund.ID = id
			addedFunds = append(addedFunds, fund)
		}

		results, err := FundsByNames(db, tableName, []string{"fund1", "fund2"})
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(addedFunds[:2], results) {
			t.Fatalf("Queried results from fundsByNames are wrong, expected %+v, got %+v", addedFunds[:2], results)
		}

		fundsNotIn, err := FundsNotInNames(db, tableName, []string{"fund1", "fund2", "fund3", "fund4"})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Print(fundsNotIn)

		if !reflect.DeepEqual([]string{"fund4"}, fundsNotIn) {
			t.Fatalf("Queried results from fundsNotInNames are wrong, expected %+v, got %+v", []string{"fund4"}, fundsNotIn)
		}

		queriedFunds, err := FundsNotDownloadedWithinDays(db, tableName, 5)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(addedFunds, queriedFunds) {
			t.Fatalf("funds not downloaded within days funciton is wrong, expected %+v, got %+v", funds, queriedFunds)
		}

		fund5 := Fund{Fundname: "fund5", Link: "link5"}
		AddFund(db, tableName, fund5)
		UpdateFundName(db, tableName, "fund5", "newfund5")
		queriedFunds, err = FundsByNames(db, tableName, []string{"newfund5"})
		if err != nil {
			t.Fatal(err)
		}

		if len(queriedFunds) == 0 {
			t.Fatal("Expected results when querying for updated fund 5, got empty funds list")
		} else if queriedFunds[0].Fundname != "newfund5" {
			t.Fatalf("Fund name not updated correctly, expected newfund5, got %s", queriedFunds[0].Fundname)
		}

		//Fund successfully downloaded
		UpdateLastDownloaded(db, tableName, "newfund5")
		queriedFunds, err = FundsNotDownloadedWithinDays(db, tableName, 5)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(addedFunds, queriedFunds) {
			t.Fatalf("Failed to update last downloaded field, expected %+v, got %+v", funds, queriedFunds)
		}
	})

}
