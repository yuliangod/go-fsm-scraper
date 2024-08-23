package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var db *sql.DB

type Fund struct {
	ID             int64
	Fundname       string
	Link           string
	Lastdownloaded []uint8
}

func ConnectDB() *sql.DB {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	// Capture connection properties.
	cfg := mysql.Config{
		User:                 os.Getenv("DBUSER"),
		Passwd:               os.Getenv("DBPASS"),
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "recordings",
		AllowNativePasswords: true,
	}
	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	return db
}

func AddFund(db *sql.DB, tableName string, fund Fund) int64 {
	result, err := db.Exec(fmt.Sprintf("INSERT INTO %s (fundname, link) VALUES (?, ?)", tableName), fund.Fundname, fund.Link)
	if err != nil {
		log.Fatalf("addFund: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		log.Fatalf("addFund: %v", err)
	}
	log.Printf("Added %s to %s with index %v", fund.Fundname, tableName, id)
	return id
}

func FundsByNames(db *sql.DB, tableName string, names []string) ([]Fund, error) {
	template := fmt.Sprintf("SELECT * FROM %s WHERE fundname IN (", tableName)
	for _, name := range names {
		template = template + fmt.Sprintf("'%s',", name)
	}
	template = template[:len(template)-1]
	template += ");"

	funds, err := queryFunds(db, template)
	return funds, err
}

func queryFunds(db *sql.DB, template string) ([]Fund, error) {
	var funds []Fund

	rows, err := db.Query(template)
	if err != nil {
		log.Fatalf("get funds by template %s: %v", template, err)
	}
	defer rows.Close()
	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var fund Fund
		if err := rows.Scan(&fund.ID, &fund.Fundname, &fund.Link, &fund.Lastdownloaded); err != nil {
			return nil, fmt.Errorf("error obtaining values from row: %v", err)
		}
		funds = append(funds, fund)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error with row: %v", err)
	}

	//log.Printf("Querying %s with template: %s, got %+v", tableName, template, funds)

	return funds, nil
}

func FundsNotInNames(db *sql.DB, tableName string, names []string) ([]string, error) {
	funds, err := FundsByNames(db, tableName, names)
	if err != nil {
		return nil, err
	}

	var queriedFundNames []string
	for _, fund := range funds {
		queriedFundNames = append(queriedFundNames, fund.Fundname)
	}

	fundsNotIn := Difference(names, queriedFundNames)
	log.Print(fundsNotIn)

	return fundsNotIn, nil
}

// Create fund table if it does not exist
func CreateFundTable(db *sql.DB, tableName string) {
	_, err := db.Exec(fmt.Sprintf(
		`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			fundname VARCHAR(128) NOT NULL,
			link VARCHAR(255) NOT NULL,
			lastdownloaded DATETIME
		);
		`, tableName))
	if err != nil {
		log.Fatalf("Error creating fund table: %s", err)
	}
}

// difference returns the elements in `a` that aren't in `b`.
func Difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func CreateTestFundTable(db *sql.DB, testTableName string) {
	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", testTableName))
	if err != nil {
		log.Fatalf("Error deleting fund table: %s", err)
	}

	CreateFundTable(db, testTableName)
}

func UpdateFundName(db *sql.DB, tableName, oldfundName, newfundName string) {
	result, err := db.Exec(fmt.Sprintf("UPDATE %s SET fundname = '%s' WHERE fundname = '%s'", tableName, newfundName, oldfundName))
	if err != nil {
		log.Fatalf("Error deleting fund table: %s", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if rowsAffected == 0 {
		log.Fatalf("No rows affected when updating name for %s", oldfundName)
	}
}

func FundsNotDownloadedWithinDays(db *sql.DB, tableName string, days int) ([]Fund, error) {
	template := fmt.Sprintf("SELECT * FROM %s WHERE lastdownloaded NOT BETWEEN CURDATE() - INTERVAL %v DAY AND CURDATE() OR lastdownloaded IS NULL;", tableName, days)
	funds, err := queryFunds(db, template)
	return funds, err
}

func UpdateLastDownloaded(db *sql.DB, tableName, fundName string) {
	result, err := db.Exec(fmt.Sprintf("UPDATE %s SET lastdownloaded = CURDATE() WHERE fundname = '%s'", tableName, fundName))
	if err != nil {
		log.Fatalf("Error deleting fund table: %s", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}
	if rowsAffected == 0 {
		log.Fatalf("No rows affected when updating last downloaded for %s", fundName)
	}
}
