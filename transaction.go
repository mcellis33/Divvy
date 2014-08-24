package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type TransactionType int

const (
	Debit TransactionType = iota
	Credit
	TransactionTypeUnknown
)

func (t TransactionType) String() string {
	switch t {
	case Debit:
		return "Debit"
	case Credit:
		return "Credit"
	case TransactionTypeUnknown:
		return "Unknown"
	default:
		return "Invalid"
	}
}

func ParseTransactionType(s string) (TransactionType, error) {
	switch strings.ToLower(s) {
	case "credit":
		return Credit, nil
	case "debit":
		return Debit, nil
	default:
		return TransactionTypeUnknown, fmt.Errorf("failed to parse transaction type '%v'", s)
	}
}

type Transaction struct {
	Time                time.Time
	Description         string
	OriginalDescription string
	Amount              float64
	Type                TransactionType
	Category            string
	AccountName         string
	Labels              string
	Notes               string
}

func (t *Transaction) String() string {
	b := new(bytes.Buffer)
	b.WriteString("Transaction:\n")
	b.WriteString(fmt.Sprintf("  Time:                  %v\n", t.Time))
	b.WriteString(fmt.Sprintf("  Description:           %v\n", t.Description))
	b.WriteString(fmt.Sprintf("  Original Description:  %v\n", t.OriginalDescription))
	b.WriteString(fmt.Sprintf("  Amount:                %v\n", t.Amount))
	b.WriteString(fmt.Sprintf("  Type:                  %v\n", t.Type))
	b.WriteString(fmt.Sprintf("  Category:              %v\n", t.Category))
	b.WriteString(fmt.Sprintf("  Account Name:          %v\n", t.AccountName))
	b.WriteString(fmt.Sprintf("  Labels:                %v\n", t.Labels))
	b.WriteString(fmt.Sprintf("  Notes:                 %v", t.Notes))
	return b.String()
}

func (t *Transaction) AbbrString() string {
	return fmt.Sprintf("%v $%v '%v'", t.Time, t.Amount, t.Description)
}

type TransactionId string

// Get a unique identifier for the transaction
func (t *Transaction) Id() TransactionId {
	return TransactionId(t.AbbrString())
}

func ParseTransactions(reader io.Reader) (transactions []*Transaction, err error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = 9
	var records [][]string
	records, err = csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error parsing transactions as csv: %v", err)
	}
	dateFormat := "1/2/2006"
	for _, record := range records {
		time, err := time.Parse(dateFormat, record[0])
		if err != nil {
			ReportError(fmt.Errorf("failed to parse transaction date '%v': %v", record[0], err))
		}
		description := record[1]
		originalDescription := record[2]
		amount, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			ReportError(fmt.Errorf("failed to parse transaction amount '%v': %v", record[3], err))
		}
		transactionType, err := ParseTransactionType(record[4])
		if err != nil {
			ReportError(fmt.Errorf("failed to parse transaction type '%v': %v", record[4], err))
		}
		category := record[5]
		accountName := record[6]
		labels := record[7]
		notes := record[8]

		newTransaction := &Transaction{
			time,
			description,
			originalDescription,
			amount,
			transactionType,
			category,
			accountName,
			labels,
			notes,
		}
		transactions = append(transactions, newTransaction)
	}
	return transactions, nil
}

func ReportError(e error) {
	fmt.Printf("ERROR: %v\n", e.Error())
}
