package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReconciliationInitialization(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name                string
		transactionFilePath string
		statementFilePaths  []string
		startDate           time.Time
		endDate             time.Time
		expectedErr         error
	}{
		{
			name:                "valid input",
			transactionFilePath: "../file/transaction.csv",
			statementFilePaths:  []string{"../file/bca.csv"},
			startDate:           now,
			endDate:             now,
			expectedErr:         nil,
		},
		{
			name:                "file not csv",
			transactionFilePath: "test",
			expectedErr:         errors.New("test is not a csv"),
		},
		{
			name:                "file not exist",
			transactionFilePath: "test.csv",
			expectedErr:         errors.New("test.csv is not exist"),
		},
	}

	for _, tt := range testCases {
		_, err := NewReconciliation(tt.transactionFilePath, tt.statementFilePaths, tt.startDate, tt.endDate)
		assert.Equal(t, err, tt.expectedErr)
	}
}
