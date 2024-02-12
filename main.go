package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type Config struct {
	PersistFilePath       string
	RequestTimeoutSeconds int64
	ServerPort            string
}

var (
	requestTimestamps []int64
	mutex             sync.Mutex
	logger            *log.Logger
)

var config = Config{
	PersistFilePath:       "requests.bin",
	RequestTimeoutSeconds: 60,
	ServerPort:            ":8080",
}

func appendRequestTimestamp(timestamp int64) {
	mutex.Lock()
	defer mutex.Unlock()

	cutoff := time.Now().Unix() - config.RequestTimeoutSeconds
	startIndex := 0
	for i, ts := range requestTimestamps {
		if ts >= cutoff {
			startIndex = i
			break
		}
	}
	requestTimestamps = append(requestTimestamps[startIndex:], timestamp)

	file, err := os.OpenFile(config.PersistFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Println("Error opening persistence file:", err)
		return
	}
	defer file.Close()

	if err := binary.Write(file, binary.BigEndian, timestamp); err != nil {
		logger.Println("Error writing to persistence file:", err)
	}
}

func countRecentRequests() int {
	mutex.Lock()
	defer mutex.Unlock()

	cutoff := time.Now().Unix() - config.RequestTimeoutSeconds
	count := 0
	for _, ts := range requestTimestamps {
		if ts >= cutoff {
			count++
		}
	}
	return count
}

func recoverState() {
	file, err := os.Open(config.PersistFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Println("Error opening persistence file for recovery:", err)
		}
		return
	}
	defer file.Close()

	var timestamp int64
	for {
		if err := binary.Read(file, binary.BigEndian, &timestamp); err != nil {
			break
		}
		requestTimestamps = append(requestTimestamps, timestamp)
	}
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	timestamp := time.Now().Unix()
	appendRequestTimestamp(timestamp)

	count := countRecentRequests()
	response := fmt.Sprintf("Requests in the last %d seconds: %d", config.RequestTimeoutSeconds, count)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func main() {
	logger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	recoverState()

	http.HandleFunc("/", requestHandler)
	logger.Println("Starting server on port", config.ServerPort)
	if err := http.ListenAndServe(config.ServerPort, nil); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
