package main

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestWriteThenRead(t *testing.T) {
	// Create a test history file
	historyFile, err := NewHistoryFile(".")
	if err != nil {
		t.Fatal(err)
	}
	defer historyFile.Close()
	historyFilePath := historyFile.Path()
	t.Log("New history file path: ", historyFilePath)

	// Create some test divvies
	divvies := []*Divvy{
		{
			&Transaction{
				time.Now(),
				"desc 0",
				"orig desc 0",
				-42.1,
				"cat 0",
				"acct 0",
				"label 0",
				"note 0",
			},
			Assignment{
				"Mark": float64(12.1),
				"Anne": float64(30.0),
			},
		},
		{
			&Transaction{
				time.Now().Add(-5 * time.Hour),
				"desc 1",
				"orig desc 1",
				50,
				"cat 1",
				"acct 1",
				"label 1",
				"note 1",
			},
			Assignment{
				"Mark": float64(24),
				"Anne": float64(26),
			},
		},
	}

	// Write the Divvy to the history file and close the history file
	for _, d := range divvies {
		historyFile.Write(d)
	}
	historyFile.Close()

	// Load it back out
	divviesReloaded, err := LoadHistoryFile(historyFilePath)
	if err != nil {
		t.Fatal(err)
	}
	for i, dr := range divviesReloaded {
		if !reflect.DeepEqual(dr, divvies[i]) {
			t.Fatalf("read divvy '%v' was different from written divvy '%v'", dr, divvies[i])
		}
	}

	os.Remove(historyFilePath)
}
