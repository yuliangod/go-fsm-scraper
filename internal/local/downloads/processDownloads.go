package local

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func processDownloads(fundNames []string) error {
	downloadsDir := getDownloadsDir()
	dataDir := getDataDir()

	err := shiftFSMfiles(fundNames, downloadsDir, dataDir)
	if err != nil {
		return fmt.Errorf("error shifting FSM files: %s", err)
	}

	cmd := exec.Command("python", "compile_data.py")
	_, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("error running python script to compile fsm data: %s", err)
	}
	return nil
}

func shiftFSMfiles(fundNames []string, targetDir, destDir string) error {

	// Read the contents of the Downloads directory
	files, err := os.ReadDir(targetDir)
	if err != nil {
		log.Fatalf("Error reading Downloads directory: %s", err)
	}

	clearDataFolder(destDir)

	// Get the current date
	now := time.Now()
	today := now.YearDay()

	fundMap := sliceToMap(fundNames)

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".xlsx" {
			fileInfo, err := file.Info()
			if err != nil {
				fmt.Println("Error getting file info:", err)
				continue
			}

			// Check if the file was modified today
			if fileInfo.ModTime().YearDay() == today && fileInfo.ModTime().Year() == now.Year() {
				FSMfilename := file.Name()

				// Check if file name is in list of funds
				for _, fundName := range fundNames {
					if strings.Contains(FSMfilename, fundName) {
						// Check if file is already shifted over to the new folder
						if !fundMap[fundName] {
							FSMfilepath := filepath.Join(targetDir, FSMfilename)
							renameColumn(FSMfilepath, fundName)
							relocateExcel(FSMfilename, FSMfilepath, destDir)
							fundMap[fundName] = true
						}
					}
				}
			}
		}
	}

	var failedSlice []string
	for key, value := range fundMap {
		if !value {
			failedSlice = append(failedSlice, key)
		}
	}
	if len(failedSlice) != 0 {
		return fmt.Errorf("%s failed to download", failedSlice)
	}

	return nil
}

func relocateExcel(FSMfilename string, FSMfilepath string, destDir string) {
	err := os.MkdirAll(destDir, 0777)
	if err != nil {
		log.Fatalf("Error creating directory to store FSM data: %s", err)
	}

	// Relocate excel file to new directory
	destPath := filepath.Join(destDir, FSMfilename)
	err = os.Rename(FSMfilepath, destPath)
	if err != nil {
		log.Fatalf("Error relocating file: %s", err)
	}
	log.Printf("Relocating %s from %s to %s", FSMfilename, FSMfilepath, destPath)
}

func sliceToMap(slice []string) map[string]bool {
	// Create a map to hold the elements
	m := make(map[string]bool)

	// Iterate over the slice and add each element to the map
	for _, item := range slice {
		m[item] = false
	}

	return m
}

func getDownloadsDir() string {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Error getting home directory: %s", err)
	}

	// Construct the path to the Downloads directory
	downloadsDir := filepath.Join(homeDir, "Downloads")
	return downloadsDir
}

func getDataDir() string {
	curDir, err := os.Getwd()

	if err != nil {
		log.Fatalf("Error getting current working directory: %s", err)
	}

	// Create new directory to store data if it does not exist
	dataDir := filepath.Join(curDir, "data")
	return dataDir
}

func renameColumn(excelFilePath, fundName string) error {
	// Open the existing Excel file
	xlFile, err := excelize.OpenFile(excelFilePath)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}

	defer func() {
		// Close the spreadsheet.
		if err := xlFile.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sheetName := xlFile.GetSheetName(0)
	xlFile.SetCellValue(sheetName, "B1", fundName)

	// Save spreadsheet by the given path.
	if err := xlFile.SaveAs(excelFilePath); err != nil {
		return fmt.Errorf("unable to save spreadsheet: %s", err)
	}

	return nil
}

func clearDataFolder(destDir string) error {
	// Delete all old files in data directory
	files, err := os.ReadDir(destDir)
	if err != nil {
		log.Fatalf("Error reading data directory: %s", err)
	}

	// Iterate over the list of files and delete each one
	for _, file := range files {
		filePath := filepath.Join(destDir, file.Name())
		if !file.IsDir() {
			err = os.Remove(filePath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
