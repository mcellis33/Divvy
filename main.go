package main

import (
	"flag"
	"fmt"
	"os"
	"path"
)

type parameters struct {
	historyDir       string
	transactionsPath string
}

func main() {
	parameters, err := getParameters()
	if err != nil {
		fmt.Printf("error in parameters: %v\n", err)
		return
	}

	// Load transactions
	transactions, err := LoadTransactions(parameters.transactionsPath)
	if err != nil {
		fmt.Printf("failed to load transactions: %v\n", err)
		return
	}

	// Load divvy history
	history, err := LoadHistory(parameters.historyDir)
	if err != nil {
		fmt.Printf("failed to load history: %v\n", err)
		return
	}

	// TODO: create a config file and load this from it
	people := []string{"Anne", "Mark"}

	historyFile, err := NewHistoryFile(parameters.historyDir)
	if err != nil {
		fmt.Printf("failed to create new history file: %v\n", err)
		return
	}
	defer historyFile.Close()
	fmt.Printf("history file: %v\n", historyFile.Path())

	err = DivvyTransactions(people, transactions, history, historyFile)
	if err != nil {
		fmt.Printf("failed to divvy transactions: %v\n", err)
		return
	}

	divvies, err := LoadHistoryFile(historyFile.Path())
	if err != nil {
		fmt.Errorf("failed to open history file '%v' for reporting: %v", historyFile.Path(), err)
	}
	if len(divvies) == 0 {
		fmt.Printf("no new transactions found\n")
		os.Remove(historyFile.Path())
		return
	}

	fmt.Println()
	err = ReportDivvies(divvies)
	if err != nil {
		fmt.Printf("failed to report divvied transactions: %v\n", err)
	}
}

func getParameters() (*parameters, error) {
	// Get command line parameters
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get the working directory: %v", err)
	}

	var historyDir string
	var transactionsPath string
	flag.StringVar(
		&historyDir,
		"history",
		path.Join(workingDir, "divvy_history"),
		"The directory in which program results are stored.",
	)
	flag.StringVar(
		&transactionsPath,
		"transactions",
		path.Join(workingDir, "transactions.csv"),
		"The transactions to process.",
	)
	flag.Parse()

	// Validate that history directory exists
	historyDirStat, err := os.Stat(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			mkdErr := os.Mkdir(historyDir, 0777)
			if mkdErr != nil {
				return nil, fmt.Errorf("failed to create history directory '%v': %v", historyDir, err)
			}
		} else {
			return nil, fmt.Errorf("failed to query history directory '%v': %v", historyDir, err)
		}
	} else {
		if !historyDirStat.IsDir() {
			fmt.Printf("'%v' is not a directory", historyDir)
		}
	}

	// Validate that transactions file exists
	if _, err := os.Stat(transactionsPath); err != nil {
		return nil, fmt.Errorf("transactions file '%v' does not exist", transactionsPath)
	}

	return &parameters{historyDir, transactionsPath}, nil
}

func LoadTransactions(transactionsFilePath string) ([]*Transaction, error) {
	f, err := os.Open(transactionsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open transactions file '%v': %v", transactionsFilePath, err)
	}
	transactions, err := ParseTransactions(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transactions: %v", err)
	}
	return transactions, nil
}

func ReportDivvies(divvies []*Divvy) error {
	totals := make(map[string]float64)
	for _, d := range divvies {
		for person, amount := range d.Assignment {
			prevTotal, ok := totals[person]
			if ok {
				totals[person] = prevTotal + amount
			} else {
				totals[person] = amount
			}
		}
	}

	fmt.Println("Total responsibilities:")
	for person, total := range totals {
		fmt.Printf("  %v: $%v\n", person, total)
	}

	return nil
}
