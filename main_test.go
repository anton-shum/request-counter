package main

import (
	"os"
	"sync"
	"testing"
	"time"
)

const testFilePath = "temp_requests_test.bin"

func setupTestEnv() {
	config.PersistFilePath = testFilePath
	requestTimestamps = []int64{}
}

func teardownTestEnv() {
	os.Remove(testFilePath)
}

func resetTimestamps() {
	mutex.Lock()
	defer mutex.Unlock()
	requestTimestamps = []int64{}
}

func TestAppendingAndCountingTimestamps(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	currentTime := time.Now().Unix()
	appendRequestTimestamp(currentTime)

	if len(requestTimestamps) != 1 {
		t.Errorf("Expected 1 timestamp, got %d", len(requestTimestamps))
	}

	count := countRecentRequests()
	if count != 1 {
		t.Errorf("Expected count of 1, got %d", count)
	}

	oldTimestamp := currentTime - config.RequestTimeoutSeconds - 1
	appendRequestTimestamp(oldTimestamp)

	count = countRecentRequests()
	if count != 1 {
		t.Errorf("Expected count of 1 after adding old timestamp, got %d", count)
	}
}

func TestConcurrentAppending(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	var wg sync.WaitGroup
	concurrentRequests := 100

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			appendRequestTimestamp(time.Now().Unix())
		}()
	}

	wg.Wait()

	if len(requestTimestamps) != concurrentRequests {
		t.Errorf("Expected %d timestamps, got %d", concurrentRequests, len(requestTimestamps))
	}
}

func TestStateRecovery(t *testing.T) {
	setupTestEnv()
	defer teardownTestEnv()

	appendRequestTimestamp(time.Now().Unix())

	resetTimestamps()
	recoverState()

	if len(requestTimestamps) == 0 {
		t.Fatal("Failed to recover state, no timestamps found")
	}
}
