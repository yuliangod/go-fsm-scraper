package local

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/tealeg/xlsx"
)

func TestProcessDownloads(t *testing.T) {

	t.Run("Testing if only correct files are moved to data directory", func(t *testing.T) {
		tmpDownloadsDir, tmpDataDir := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDownloadsDir)
		defer os.RemoveAll(tmpDataDir)

		testFundNames := []string{"fund1", "fund2", "fund3"}
		createFundFiles := []string{"fund1-copy", "fund2", "fund3", "fund4", "fund1"}

		for _, testFundName := range createFundFiles {
			createTempXlsx(t, testFundName, tmpDownloadsDir)
		}

		createTempXlsx(t, "fund5", tmpDataDir)

		shiftFSMfiles(testFundNames, tmpDownloadsDir, tmpDataDir)
		tmpDatafiles, err := os.ReadDir(tmpDataDir)
		if err != nil {
			log.Fatal(err)
		}

		var shiftedFiles []string
		for _, file := range tmpDatafiles {
			shiftedFiles = append(shiftedFiles, file.Name())
		}

		want := []string{"fund1-copy.xlsx", "fund2.xlsx", "fund3.xlsx"}

		if !reflect.DeepEqual(want, shiftedFiles) {
			log.Fatalf("Wanted %v, got %v", want, shiftedFiles)
		}
	})
	t.Run("Testing if old files are not moved to data file", func(t *testing.T) {
		tmpDownloadsDir, tmpDataDir := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDownloadsDir)
		defer os.RemoveAll(tmpDataDir)

		testFundNames := []string{"fund1", "fund2", "fund3"}
		createFundFiles := []string{"fund1-copy", "fund2", "fund3", "fund4", "fund1"}

		for _, testFundName := range createFundFiles {
			createTempXlsx(t, testFundName, tmpDownloadsDir)
		}

		newModTime := time.Now().Add(-24 * time.Hour) // 24 hours ago
		err := os.Chtimes(filepath.Join(tmpDownloadsDir, "fund1-copy.xlsx"), newModTime, newModTime)
		if err != nil {
			fmt.Printf("Error changing file times: %v\n", err)
			return
		}

		shiftFSMfiles(testFundNames, tmpDownloadsDir, tmpDataDir)
		tmpDatafiles, err := os.ReadDir(tmpDataDir)
		if err != nil {
			log.Fatal(err)
		}

		var shiftedFiles []string
		for _, file := range tmpDatafiles {
			shiftedFiles = append(shiftedFiles, file.Name())
		}

		want := []string{"fund1.xlsx", "fund2.xlsx", "fund3.xlsx"}

		if !reflect.DeepEqual(want, shiftedFiles) {
			log.Fatalf("Wanted %v, got %v", want, shiftedFiles)
		}
	})

	t.Run("Test if downloads do not contain all funds in fund name", func(t *testing.T) {
		tmpDownloadsDir, tmpDataDir := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDownloadsDir)
		defer os.RemoveAll(tmpDataDir)

		testFundNames := []string{"fund1", "fund2", "fund3"}
		createFundFiles := []string{"fund1-copy", "fund2"}

		for _, testFundName := range createFundFiles {
			createTempXlsx(t, testFundName, tmpDownloadsDir)
		}

		err := shiftFSMfiles(testFundNames, tmpDownloadsDir, tmpDataDir)

		if err == nil {
			log.Fatalf("Expected error, none given")
		}
	})

	t.Run("Test if price column has been renamed to fund name", func(t *testing.T) {
		tmpDownloadsDir, tmpDataDir := setupTestEnvironment(t)
		defer os.RemoveAll(tmpDownloadsDir)
		defer os.RemoveAll(tmpDataDir)

		createTempXlsx(t, "fund1", tmpDownloadsDir)

		shiftFSMfiles([]string{"fund1"}, tmpDownloadsDir, tmpDataDir)

		file, err := xlsx.OpenFile(filepath.Join(tmpDataDir, "fund1.xlsx"))
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}

		sheet := file.Sheets[0]
		firstRow := sheet.Rows[0]

		if firstRow.Cells[1].Value != "fund1" {
			log.Fatalf("Column heading was not renamed from Price to fund name")
		}

	})
}

func setupTestEnvironment(t testing.TB) (tmpDownloadsDir string, tmpDataDir string) {
	t.Helper()

	// Create a temporary downloads directory
	tmpDownloadsDir, err := os.MkdirTemp("", "testdir")
	if err != nil {
		t.Fatal(err)
	}

	// Create a temporary data directory
	tmpDataDir, err = os.MkdirTemp("", "data")
	if err != nil {
		t.Fatal(err)
	}

	return tmpDownloadsDir, tmpDataDir
}

func createTempXlsx(t testing.TB, testFundName, tmpDownloadsDir string) string {
	t.Helper()

	fileName := fmt.Sprintf("%s.xlsx", testFundName)
	file := xlsx.NewFile()
	sheet, err := file.AddSheet("Sheet1")
	if err != nil {
		log.Fatal(err)
	}

	row := sheet.AddRow()
	cell1 := row.AddCell()
	cell1.Value = "Date"

	cell2 := row.AddCell()
	cell2.Value = "Price"

	filePath := filepath.Join(tmpDownloadsDir, fileName)
	err = file.Save(filePath)
	if err != nil {
		log.Fatalf("Error creating test excel file: %s", err)
	}

	return filePath
}
