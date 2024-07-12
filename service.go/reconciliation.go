package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reinhardhotma/amartha-test/model"
	"github.com/reinhardhotma/amartha-test/util"
)

type Reconciliation struct {
	TransactionFilePath string
	StatementFilePaths  []string
	StartDate           time.Time
	EndDate             time.Time

	wgStatementProcessing *sync.WaitGroup
	wgDataProcessing      *sync.WaitGroup

	Transactions         map[string]*model.Transaction
	ProcessedTransaction int64

	Result *model.ReconciliationResult

	jobs chan *model.BankStatement
}

func NewReconciliation(transactionFilePath string, statementFilePaths []string, startDate, endDate time.Time) (*Reconciliation, error) {
	// convert startDate and endDate to localTime
	localStartDate := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, time.Local)
	localEndDate := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, time.Local).Add(24 * time.Hour)

	reconciliation := &Reconciliation{
		TransactionFilePath:   transactionFilePath,
		StatementFilePaths:    statementFilePaths,
		StartDate:             localStartDate,
		EndDate:               localEndDate,
		wgStatementProcessing: &sync.WaitGroup{},
		wgDataProcessing:      &sync.WaitGroup{},
		Transactions:          make(map[string]*model.Transaction),
		ProcessedTransaction:  0,
		Result: &model.ReconciliationResult{
			UnmatchedTransactions: &model.UnmatchedTransactions{
				MissingBankStatements: make([]*model.Transaction, 0),
				MissingTransactions:   make(map[string][]*model.BankStatement),
				MuMap:                 make(map[string]*sync.Mutex),
			},
		},
		jobs: make(chan *model.BankStatement, 10),
	}
	err := reconciliation.validateFileName()
	if err != nil {
		return nil, err
	}
	return reconciliation, nil
}

// handle concurrent process
func (r *Reconciliation) incrementProcessedTransaction() {
	atomic.AddInt64(&r.ProcessedTransaction, 1)
}

func (r *Reconciliation) validateFileName() error {
	if !util.IsCSV(r.TransactionFilePath) {
		return fmt.Errorf("%s is not a csv", r.TransactionFilePath)
	}

	if !util.IsFileExist(r.TransactionFilePath) {
		return fmt.Errorf("%s is not exist", r.TransactionFilePath)
	}

	for _, path := range r.StatementFilePaths {
		if !util.IsCSV(path) {
			return fmt.Errorf("%s is not a csv", path)
		}

		if !util.IsFileExist(path) {
			return fmt.Errorf("%s is not exist", path)
		}
	}

	return nil
}

func (r *Reconciliation) Process() error {
	err := r.populateTransactions()
	if err != nil {
		return err
	}

	r.reconcile()
	r.collectUnmatchedTransactions()
	r.printResult()

	return nil
}

func (r *Reconciliation) printResult() {
	// this value sum the total number of rows of transactions csv and bank statements csv
	fmt.Println("Total number of transactions processed :", r.ProcessedTransaction)

	fmt.Println("Total number of matched transactions   :", r.Result.MatchedTransactions*2)

	fmt.Println("Total number of unmatched transactions :", r.Result.UnmatchedTransactions.Total)

	fmt.Println("    Details of unmatched transactions  :")

	fmt.Println("        System transaction details (missing bank transactions):")
	for _, transaction := range r.Result.UnmatchedTransactions.MissingBankStatements {
		fmt.Printf("          - TrxID : %s\n", transaction.TrxID)
		fmt.Printf("            Amount: %.2f\n", transaction.Amount)
		fmt.Printf("            Type  : %s\n", transaction.Type)
		fmt.Printf("            Time  : %v\n", transaction.Time.Format(time.DateTime))
	}

	fmt.Println("        Bank statement details (missing system transactions):")
	for bankName, statements := range r.Result.UnmatchedTransactions.MissingTransactions {
		fmt.Printf("            BANK %s:\n", bankName)
		for _, statement := range statements {
			fmt.Printf("              - ID    : %s\n", statement.ID)
			fmt.Printf("              	Amount: %.2f\n", statement.Amount)
			fmt.Printf("                Date  : %s\n", statement.Date.Format(time.DateOnly))
		}
	}
	fmt.Printf("Total discrepancies: %.2f\n", r.Result.TotalDiscrepancies)
}

func (r *Reconciliation) reconcile() {
	// initiate worker pool, currently being set to 5 worker
	for i := 0; i < 5; i++ {
		r.wgDataProcessing.Add(1)
		go r.worker(r.jobs)
	}

	// process each bank statement asynchronously
	for _, fileName := range r.StatementFilePaths {
		r.wgStatementProcessing.Add(1)
		go r.processBankStatements(fileName)
	}

	r.wgStatementProcessing.Wait()
	close(r.jobs)

	r.wgDataProcessing.Wait()
}

func (r *Reconciliation) collectUnmatchedTransactions() {
	for _, transaction := range r.Transactions {
		if transaction.Time.After(r.StartDate) && transaction.Time.Before(r.EndDate) && !transaction.Matched {
			r.Result.UnmatchedTransactions.MissingBankStatements = append(r.Result.UnmatchedTransactions.MissingBankStatements, transaction)
		}
	}
	r.Result.UnmatchedTransactions.Total += int64(len(r.Result.UnmatchedTransactions.MissingBankStatements))
}

func (r *Reconciliation) populateTransactions() error {
	transactionFile, err := os.Open(r.TransactionFilePath)
	if err != nil {
		return err
	}
	defer transactionFile.Close()

	csvReader := csv.NewReader(transactionFile)
	// skip header
	csvReader.Read()

	// we will skip error row
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println(err)
			continue
		}

		date, err := time.Parse(time.RFC3339, rec[3])
		if err != nil {
			log.Println(err)
			continue
		}

		// only process data which match the provided datetime
		if date.Before(r.StartDate) || date.After(r.EndDate) {
			continue
		}

		trxID := rec[0]

		// use decimal datatype
		amount, err := strconv.ParseFloat(rec[1], 64)
		if err != nil {
			log.Println(err)
			continue
		}

		trxType := rec[2]

		r.Transactions[trxID] = &model.Transaction{
			TrxID:   trxID,
			Amount:  amount,
			Type:    trxType,
			Time:    date,
			Matched: false,
		}
		r.incrementProcessedTransaction()
	}

	return nil
}

func (r *Reconciliation) processBankStatements(fileName string) {
	bankStatementFile, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer bankStatementFile.Close()

	// assumption: bank name will be taken from file name (expected format: {bank_name}-{anything}.csv)
	// if format is incorrect, default bank name will be "General"
	splittedDir := strings.Split(fileName, "/")
	splittedFileName := strings.Split(splittedDir[len(splittedDir)-1], ".")

	bankName := "General"
	if splittedFileName[0] != "" {
		bankName = splittedFileName[0]
	}

	r.Result.UnmatchedTransactions.MuMap[bankName] = &sync.Mutex{}

	csvReader := csv.NewReader(bankStatementFile)
	// skip header
	csvReader.Read()
	for {
		rec, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
			continue
		}
		amount, err := strconv.ParseFloat(rec[1], 64)
		if err != nil {
			log.Fatal(err)
		}

		date, err := time.Parse("2006-01-02", rec[2])
		if err != nil {
			log.Fatal(err)
		}

		statement := &model.BankStatement{
			ID:       rec[0],
			Amount:   amount,
			Date:     date.In(time.Local),
			BankName: bankName,
		}

		r.jobs <- statement
	}

	r.wgStatementProcessing.Done()
}

func (r *Reconciliation) worker(statements <-chan *model.BankStatement) {
	for statement := range statements {
		// only process data which match the provided datetime
		if statement.Date.After(r.StartDate) && statement.Date.Before(r.EndDate) {
			r.incrementProcessedTransaction()
			trx, exist := r.Transactions[statement.ID]
			if exist {
				amount := trx.Amount
				if trx.Type == string(util.CREDIT) {
					amount *= -1
				}
				r.Transactions[statement.ID].Matched = true
				r.Result.IncrementMatchedTransactions()
				r.Result.AddTotalDiscrepancies(abs(amount - statement.Amount))
			} else {
				r.Result.UnmatchedTransactions.AddMissingTransaction(statement)
				r.Result.UnmatchedTransactions.IncrementTotal()
			}
		}
	}
	r.wgDataProcessing.Done()
}

func abs(a float64) float64 {
	if a < 0 {
		a *= -1
	}
	return a
}
