package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Assignment map[string]float64

type Divvy struct {
	*Transaction
	Assignment
}

var DIVVY_CHOICE_RUNES = [...]rune{'1', '2', '3', '4', '5', '6', '7', '8', '9'}

func DivvyTransactions(
	people []string,
	transactions []*Transaction,
	history []*Divvy,
	historyFile *HistoryFile,
) error {
	// Format the prompt and set up the map from rune (input by the user) to choice
	choices := append(people, "Split", "Skip")
	runeToChoice := make(map[rune]string)
	var promptElements []string
	for i, choice := range choices {
		choiceRune := DIVVY_CHOICE_RUNES[i]
		promptElements = append(promptElements, fmt.Sprintf("[%v] %v", string(choiceRune), choice))
		runeToChoice[choiceRune] = choice
	}
	prompt := strings.Join(promptElements, "  ")

	// Build a set of IDs of transactions that have already been divvied
	// so that we can quickly determine whether to divvy a transaction
	divviedIds := make(map[TransactionId]struct{})
	for _, d := range history {
		divviedIds[d.Transaction.Id()] = struct{}{}
	}

	bio := bufio.NewReader(os.Stdin)
	for _, transaction := range transactions {
		// If the transaction has not already been divvied
		if _, ok := divviedIds[transaction.Id()]; !ok {
			// Get the user's choice
			fmt.Println()
			fmt.Println(transaction.String())
			var choice string
			for {
				fmt.Println()
				fmt.Println(prompt)
				s, err := bio.ReadString('\n')
				if err != nil {
					return fmt.Errorf("failed to read string: %v", err)
				}
				var ok bool
				choiceRune := []rune(s)[0]
				choice, ok = runeToChoice[choiceRune]
				if ok {
					break
				} else {
					fmt.Printf("choice '%v' not found\n", string(choiceRune))
				}
			}

			assignment := make(Assignment)
			if choice == "Split" {
				for _, p := range people {
					assignment[p] = transaction.Amount / float64(len(people))
				}
			} else if choice == "Skip" {
				// No assignments
			} else {
				assignment[choice] = transaction.Amount
			}

			err := historyFile.Write(&Divvy{
				transaction,
				assignment,
			})
			if err != nil {
				return fmt.Errorf("failed to write history: %v", err)
			}
		}
	}

	return nil
}
