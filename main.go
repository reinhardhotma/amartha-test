package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/reinhardhotma/amartha-test/service.go"
)

func main() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	time.Local = loc

	fmt.Println("Enter system transaction csv file path:")
	var transactionPath string
	fmt.Scan(&transactionPath)

	fmt.Println("Enter bank statement csv file paths: (for multiple entries, separate by comma. ex: bca.csv,mandiri.csv)")
	var bankStatementPath string
	fmt.Scan(&bankStatementPath)

	fmt.Println("Enter reconciliation start date: (use ISO8601 date format; YYYY-MM-DD)")
	var startDateStr string
	fmt.Scan(&startDateStr)

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		panic(err)
	}

	fmt.Println("Enter reconciliation end date: (use ISO8601 date format; YYYY-MM-DD)")
	var endDateStr string
	fmt.Scan(&endDateStr)

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		panic(err)
	}

	bankStatementPaths := strings.Split(bankStatementPath, ",")

	reconciliation, err := service.NewReconciliation(transactionPath, bankStatementPaths, startDate, endDate)
	if err != nil {
		panic(err)
	}
	err = reconciliation.Process()
	if err != nil {
		panic(err)
	}
}
