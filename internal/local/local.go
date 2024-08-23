package local

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"scraper/internal/database"
	"strings"

	"github.com/xuri/excelize/v2"
)

// const user string = "Liang"

func GetAllFunds(fundsinfopath string) []string {
	curDir, err := os.Getwd()

	if err != nil {
		log.Fatalf("Error getting current working directory: %s", err)
	}

	// Open the Excel file
	f, err := excelize.OpenFile(filepath.Join(curDir, fundsinfopath))
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}

	// Specify the sheet name
	sheetName := strings.ReplaceAll(fundsinfopath, ".xlsx", "")

	// Get all the rows in the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatalf("Error getting rows: %v", err)
	}

	var fundNames []string
	// Iterate through the rows and print the first column
	for _, row := range rows {
		if len(row) > 0 { //&& row[1] == user
			fundNames = append(fundNames, row[0])
		}
	}

	return fundNames[1:]
}

func GetFundsOwned(planningRelativeFilepath string) []string {
	sheetName := "Planning"
	f := openSheet(planningRelativeFilepath)

	// Get all the rows in the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatalf("Error getting rows: %v", err)
	}

	var fundNames []string
	// Iterate through the rows and print the first column
	for _, row := range rows {
		if len(row) > 0 { //&& row[1] == user
			fundNames = append(fundNames, row[0])
		}
	}

	return fundNames[1:]
}

func AddFunds(funds []database.Fund, planningRelativeFilepath, sheetName string) {
	f := openSheet(planningRelativeFilepath)

	for _, fund := range funds {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Fatalf("Error getting rows: %v", err)
		}

		newRow := []interface{}{fund.Fundname, fund.Link}

		// Insert the new row at the end of the sheet
		rowIndex := len(rows) + 1
		cellRange := fmt.Sprintf("A%d", rowIndex)
		if err := f.SetSheetRow(sheetName, cellRange, &newRow); err != nil {
			log.Fatalf("error setting sheet row: %v", err)
		}
	}

	// Save the file with the updated row
	if err := f.SaveAs(planningRelativeFilepath); err != nil {
		log.Fatalf("error saving file: %v", err)
	}

	fmt.Println("New row added successfully!")
}

func FundsByNames(planningRelativeFilepath, sheetName string, names []string) ([]database.Fund, error) {
	f := openSheet(planningRelativeFilepath)
	var funds []database.Fund

	for _, name := range names {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return nil, err
		}

		//fundFound := false
		for _, row := range rows {
			if row[0] == name {
				funds = append(funds, database.Fund{Fundname: name, Link: row[1]})
				//fundFound = true
				break
			}
		}

		/*if fundFound {
			continue
		} else {
			return nil, fmt.Errorf("%s not found in excel sheet %s", name, sheetName)
		}*/
	}

	return funds, nil
}

func FundsNotInNames(planningRelativeFilepath, sheetName string, names []string) ([]string, error) {
	funds, err := FundsByNames(planningRelativeFilepath, sheetName, names)
	if err != nil {
		return nil, err
	}

	var queriedFundNames []string
	for _, fund := range funds {
		queriedFundNames = append(queriedFundNames, fund.Fundname)
	}

	fundsNotIn := database.Difference(names, queriedFundNames)

	return fundsNotIn, nil
}

func UpdateFundName(oldfundName, newfundName, planningRelativeFilepath, sheetName string) error {
	f := openSheet(planningRelativeFilepath)

	cellAddressList, err := findCellCords(oldfundName, planningRelativeFilepath, sheetName)
	if err != nil {
		return err
	}

	for _, cellAddress := range cellAddressList {
		// Set the new value for the specified cell
		if err := f.SetCellValue(sheetName, cellAddress, newfundName); err != nil {
			log.Fatalf("error setting cell value: %v", err)
		}
	}

	// Save the file with the updated value
	if err := f.SaveAs(planningRelativeFilepath); err != nil {
		log.Fatalf("error saving file: %v", err)
	}

	fmt.Println("Cell updated successfully!")
	return nil
}

func findCellCords(cellvalue, planningRelativeFilepath, sheetName string) ([]string, error) {
	f := openSheet(planningRelativeFilepath)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Fatalf("Error getting rows: %v", err)
	}

	var cellAddressList []string
	// Iterate through rows and cells to find the value
	for rowIndex, row := range rows {
		for colIndex, cell := range row {
			if cell == cellvalue {
				// Convert row and column indices to cell address (e.g., A1, B2)
				cellAddress, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
				if err != nil {
					return []string{""}, err
				}
				cellAddressList = append(cellAddressList, cellAddress)
			}
		}
	}

	if len(cellAddressList) == 0 {
		return []string{""}, fmt.Errorf("value '%s' not found", cellvalue)
	}
	return cellAddressList, nil
}

func openSheet(planningRelativeFilepath string) *excelize.File {
	// Open the Excel file
	f, err := excelize.OpenFile(planningRelativeFilepath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}

	return f
}

func ClearFolder(relDirPath string) error {
	curDir, err := os.Getwd()

	if err != nil {
		log.Fatalf("Error getting current working directory: %s", err)
	}

	// Open the directory
	dir, err := os.Open(filepath.Join(curDir, relDirPath))
	if err != nil {
		return fmt.Errorf("could not open directory: %w", err)
	}
	defer dir.Close()

	// Read all directory entries
	entries, err := dir.Readdir(-1)
	if err != nil {
		return fmt.Errorf("could not read directory entries: %w", err)
	}

	// Iterate over each entry
	for _, entry := range entries {
		// Skip directories, only delete files
		if entry.IsDir() {
			continue
		}

		// Build the full path to the file
		filePath := filepath.Join(relDirPath, entry.Name())

		// Delete the file
		err := os.Remove(filePath)
		if err != nil {
			return fmt.Errorf("could not remove file %s: %w", filePath, err)
		}

		fmt.Printf("Deleted file: %s\n", filePath)
	}

	return nil
}
