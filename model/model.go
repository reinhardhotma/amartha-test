package model

import (
	"sync"
	"sync/atomic"
	"time"
)

type Transaction struct {
	TrxID   string    `json:"trxID"`
	Amount  float64   `json:"amount"`
	Type    string    `json:"type"`
	Time    time.Time `json:"transactionTime"`
	Matched bool
}

type BankStatement struct {
	ID       string    `json:"unique_identifier"`
	Amount   float64   `json:"amount"`
	Date     time.Time `json:"date"`
	BankName string
}

type ReconciliationResult struct {
	ProcessedTransactions int64
	MatchedTransactions   int64
	UnmatchedTransactions *UnmatchedTransactions
	TotalDiscrepancies    float64
	mu                    sync.Mutex
}

// handle concurrent process
func (rr *ReconciliationResult) IncrementMatchedTransactions() {
	atomic.AddInt64(&rr.MatchedTransactions, 1)
}

// handle concurrent process
func (rr *ReconciliationResult) AddTotalDiscrepancies(val float64) {
	rr.mu.Lock()
	rr.TotalDiscrepancies += val
	rr.mu.Unlock()
}

type UnmatchedTransactions struct {
	Total                 int64
	MissingBankStatements []*Transaction
	MissingTransactions   map[string][]*BankStatement
	MuMap                 map[string]*sync.Mutex
}

// handle concurrent process
func (u *UnmatchedTransactions) IncrementTotal() {
	atomic.AddInt64(&u.Total, 1)
}

// handle concurrent process
func (u *UnmatchedTransactions) AddMissingTransaction(statement *BankStatement) {
	u.MuMap[statement.BankName].Lock()
	u.MissingTransactions[statement.BankName] = append(u.MissingTransactions[statement.BankName], statement)
	u.MuMap[statement.BankName].Unlock()
}
