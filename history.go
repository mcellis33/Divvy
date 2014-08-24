package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"
)

const HISTORY_FILE_NAME_LAYOUT = "2006-01-02-15-04-05"

type HistoryFile struct {
	file *os.File
}

func NewHistoryFile(historyDir string) (*HistoryFile, error) {
	fileName := time.Now().Format(HISTORY_FILE_NAME_LAYOUT)
	filePath := path.Join(historyDir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open '%v': %v", filePath, err)
	}
	return &HistoryFile{f}, nil
}

func (h *HistoryFile) Write(d *Divvy) error {
	jsonBytes, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal divvy %v to JSON: %v", *d, err)
	}
	_, err = h.file.Write(jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to write JSON bytes for divvy %v: %v", *d, err)
	}
	return nil
}

func (h *HistoryFile) Close() {
	h.file.Close()
}

func (h *HistoryFile) Path() string {
	return h.file.Name()
}

func LoadHistory(historyDir string) ([]*Divvy, error) {
	var history []*Divvy
	historyFiles, err := ioutil.ReadDir(historyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in history directory '%v': %v", historyDir, err)
	}
	for _, historyFile := range historyFiles {
		historyFilePath := path.Join(historyDir, historyFile.Name())
		if !historyFile.IsDir() {
			moreHistory, err := LoadHistoryFile(historyFilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to load history file '%v': %v", historyFilePath, err)
			}
			history = append(history, moreHistory...)
		} else {
			fmt.Printf("'%v' is not a file, skipping", historyFilePath)
		}
	}
	return history, nil
}

func LoadHistoryFile(historyFilePath string) ([]*Divvy, error) {
	f, err := os.Open(historyFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open '%v': %v", historyFilePath, err)
	}
	decoder := json.NewDecoder(f)
	result := make([]*Divvy, 0)
	for {
		d := new(Divvy)
		err = decoder.Decode(&d)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, fmt.Errorf("failed to decode divvy from history file '%v': %v", historyFilePath, err)
			}
		}
		result = append(result, d)
	}
	return result, nil
}
