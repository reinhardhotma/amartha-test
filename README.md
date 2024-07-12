# amartha-test
Repository for Amartha's Take Home Test
**Creator: Reinhard Hotma Daniel**

## How To
To run the program, run `go mod install` first then run the program using `go run main.go`
The program will ask for input, please provide the input using the correct format (instructions provided)

## Explanation

 - The program will use worker pool to process the data concurrently
 - The program will utilize channel and sync.Mutex to enable thread communication while also maintaining consistency

## Assumptions
There will be some assumptions on the program, listed below:

 - Total number of transactions processed is the sum of processed rows of all csvs, if there's 1 matched transaction, it means the total number will be 2
 - Transactions are considered match if `trxID` from `transaction.csv` match the `unique_identifier` from `{bank}.csv`. Other than that, the transactions will be counted as unmatched transactions.
 - `transactionTime` on the `transaction.csv` will use ISO8601 format, full timestamp with offset being set to +07:00 (`2024-07-12T09:09:35+00:00`)
 - `date`on the `{bank}.csv`will also use ISO8601 format, date only `2024-07-11`