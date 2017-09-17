package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

type parameters struct {
	historyDir       string
	transactionsPath string
	continueLastFile bool
	settlementPeriod time.Duration
	checkHistory     bool
	sumFile          string
}

func main() {
	parameters, err := getParameters()
	if err != nil {
		fmt.Printf("error in parameters: %v\n", err)
		return
	}

	if parameters.sumFile != "" {
		divvies, err := LoadHistoryFile(parameters.sumFile)
		if err != nil {
			fmt.Printf("failed to load history file for sum: %v\n", err)
			return
		}
		err = ReportDivvies(divvies)
		if err != nil {
			fmt.Printf("failed to report divvies for sum: %v\n", err)
		}
		return
	}

	// Load transactions
	transactions, err := LoadTransactions(parameters.transactionsPath, parameters.settlementPeriod)
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

	if parameters.checkHistory {
		extraTransactions, extraHistory, err := CheckHistory(transactions, history)
		if err != nil {
			fmt.Printf("failed to check history: %v\n", err)
			return
		}
		fmt.Printf("Extra transactions:\n")
		for _, t := range extraTransactions {
			fmt.Printf("%v\n", t)
		}
		fmt.Printf("Extra divvies in history:\n")
		for _, h := range extraHistory {
			if h.Transaction.Description != "Payroll Contribution" &&
				h.Transaction.Description != "Deposit Shared Branch" &&
				h.Transaction.AccountName != "Microsoft 401k" &&
				h.Transaction.Category != "Dividend & Cap Gains" &&
				len(h.Assignment) != 0 {
				fmt.Printf("%v\n", h)
			}
		}
		return
	}

	// TODO: create a config file and load this from it
	people := []string{"Anne", "Mark"}

	var historyFile *HistoryFile
	if parameters.continueLastFile {
		lastHistoryFileInfo, err := GetLastModifiedFile(parameters.historyDir)
		if err != nil {
			fmt.Printf("failed to get latest history file: %v\n", err)
			return
		}
		lastHistoryFilePath := path.Join(parameters.historyDir, lastHistoryFileInfo.Name())
		historyFile, err = OpenHistoryFile(lastHistoryFilePath)
		if err != nil {
			fmt.Printf("failed to open latest history file: %v\n", err)
		}
	} else {
		historyFile, err = NewHistoryFile(parameters.historyDir)
		if err != nil {
			fmt.Printf("failed to create new history file: %v\n", err)
			return
		}
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

	p := &parameters{}
	flag.StringVar(
		&p.historyDir,
		"history",
		path.Join(workingDir, "divvy_history"),
		"The directory in which program results are stored.",
	)
	flag.StringVar(
		&p.transactionsPath,
		"transactions",
		path.Join(workingDir, "transactions.csv"),
		"The transactions to process.",
	)
	flag.BoolVar(
		&p.continueLastFile,
		"continue",
		false,
		"Append to the latest history file instead of creating a new one.",
	)
	flag.DurationVar(
		&p.settlementPeriod,
		"settlement-period",
		168*time.Hour,
		"When transactions settle, their dates and descriptions sometimes change "+
			"such that they look like new transactions to divvy. This causes duplicate "+
			"divvies. Thus, we only load transactions that are older than the "+
			"settlement period.",
	)
	flag.BoolVar(
		&p.checkHistory,
		"check-history",
		false,
		"Map each entry in the history to an entry in the transactions file, "+
			"then print all unmapped entries in the history. "+
			"This is intended to find transactions that Mint modified.",
	)
	flag.StringVar(
		&p.sumFile,
		"sum",
		"",
		"Show the sum from a history file.")
	flag.Parse()

	if p.sumFile != "" {
		return p, nil
	}

	// Validate that history directory exists
	historyDirStat, err := os.Stat(p.historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			mkdErr := os.Mkdir(p.historyDir, 0777)
			if mkdErr != nil {
				return nil, fmt.Errorf("failed to create history directory '%v': %v", p.historyDir, err)
			}
		} else {
			return nil, fmt.Errorf("failed to query history directory '%v': %v", p.historyDir, err)
		}
	} else {
		if !historyDirStat.IsDir() {
			return nil, fmt.Errorf("'%v' is not a directory", p.historyDir)
		}
	}

	// Validate that transactions file exists
	if _, err := os.Stat(p.transactionsPath); err != nil {
		return nil, fmt.Errorf("transactions file '%v' does not exist", p.transactionsPath)
	}

	// Validate settlement period
	if p.settlementPeriod < 0 {
		return nil, fmt.Errorf("settlement period cannot be negative")
	}

	return p, nil
}

func LoadTransactions(transactionsFilePath string, settlementPeriod time.Duration) ([]*Transaction, error) {
	f, err := os.Open(transactionsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open transactions file '%v': %v", transactionsFilePath, err)
	}
	transactions, err := ParseTransactions(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transactions: %v", err)
	}
	// Only load transactions that are older than the settlement period.
	settled := make([]*Transaction, 0)
	pending := make([]*Transaction, 0)
	settledDate := time.Now().Add(-settlementPeriod)
	for _, t := range transactions {
		if t.Time.Before(settledDate) {
			settled = append(settled, t)
		} else {
			pending = append(pending, t)
		}
	}
	// Notify the user which transactions are pending and thus ignored.
	fmt.Printf("Pending transactions ignored:\n")
	for _, t := range pending {
		fmt.Printf("    %v\n", t.Id())
	}
	return settled, nil
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

func GetLastModifiedFile(dir string) (os.FileInfo, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in directory '%v': %v", dir, err)
	}
	var lastTime time.Time = time.Time{}
	var lastFile os.FileInfo = nil
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			if !file.IsDir() {
				t, err := ParseHistoryFileCreationTimeFromName(file.Name())
				if err != nil {
					fmt.Printf("failed to get creation time: %v\n", err)
				} else {
					if t.After(lastTime) {
						lastTime = t
						lastFile = file
					}
				}
			} else {
				fmt.Printf("'%v' is not a file, skipping\n", file.Name())
			}
		} else {
			fmt.Printf("'%v' is hidden, skipping\n", file.Name())
		}
	}
	return lastFile, nil
}
