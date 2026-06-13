package main

import (
	"crypto/md5"
	"testing"
)

func testEncryptionKey() []byte {
	sum := md5.Sum([]byte("test-api-key"))
	return sum[:]
}

func TestJobStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := acquireJobStore(dir, testEncryptionKey())
	if err != nil {
		t.Fatalf("failed to open the store: %v", err)
	}
	defer releaseJobStore(store)

	jobs, err := store.list()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(jobs) != 0 {
		t.Fatalf("expected an empty store, got %d jobs", len(jobs))
	}

	job := &jobRecord{
		PredictionID: "pred-1",
		Model:        "kling-v2.0",
		Prompt:       "Ocean waves at sunset",
		Status:       statusProcessing,
		CreatedAt:    "2026-06-10T10:00:00Z",
	}
	if err := store.put(job); err != nil {
		t.Fatalf("failed to put: %v", err)
	}

	jobs, err = store.list()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(jobs) != 1 || jobs[0].PredictionID != "pred-1" || jobs[0].Prompt != "Ocean waves at sunset" {
		t.Fatalf("unexpected jobs: %+v", jobs)
	}

	// Update the job to a terminal state
	job.Status = statusCompleted
	job.Outputs = []string{"https://video.mp4"}
	if err := store.put(job); err != nil {
		t.Fatalf("failed to update: %v", err)
	}
	jobs, _ = store.list()
	if len(jobs) != 1 || jobs[0].Status != statusCompleted || jobs[0].Outputs[0] != "https://video.mp4" {
		t.Fatalf("update was not persisted: %+v", jobs)
	}

	if err := store.delete("pred-1"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}
	jobs, _ = store.list()
	if len(jobs) != 0 {
		t.Fatalf("expected an empty store after delete, got %d jobs", len(jobs))
	}

	// A job without a prediction ID must be rejected
	if err := store.put(&jobRecord{}); err == nil {
		t.Fatalf("expected an error for a job without a prediction ID")
	}
}

func TestJobStoreOrderingAndPrune(t *testing.T) {
	dir := t.TempDir()
	store, err := acquireJobStore(dir, testEncryptionKey())
	if err != nil {
		t.Fatalf("failed to open the store: %v", err)
	}
	defer releaseJobStore(store)

	timestamps := []string{
		"2026-06-10T10:00:00Z",
		"2026-06-10T12:00:00Z",
		"2026-06-10T11:00:00Z",
	}
	for i, createdAt := range timestamps {
		job := &jobRecord{
			PredictionID: string(rune('a' + i)),
			Model:        "m",
			Prompt:       "p",
			Status:       statusProcessing,
			CreatedAt:    createdAt,
		}
		if err := store.put(job); err != nil {
			t.Fatalf("failed to put: %v", err)
		}
	}

	// Newest first
	jobs, err := store.list()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(jobs) != 3 || jobs[0].PredictionID != "b" || jobs[1].PredictionID != "c" || jobs[2].PredictionID != "a" {
		t.Fatalf("unexpected ordering: %+v", jobs)
	}

	// Prune keeps the most recent jobs
	if err := store.prune(2); err != nil {
		t.Fatalf("failed to prune: %v", err)
	}
	jobs, _ = store.list()
	if len(jobs) != 2 || jobs[0].PredictionID != "b" || jobs[1].PredictionID != "c" {
		t.Fatalf("prune kept the wrong jobs: %+v", jobs)
	}
}

func TestJobStorePersistence(t *testing.T) {
	dir := t.TempDir()
	store, err := acquireJobStore(dir, testEncryptionKey())
	if err != nil {
		t.Fatalf("failed to open the store: %v", err)
	}
	job := &jobRecord{
		PredictionID: "persisted",
		Model:        "m",
		Prompt:       "p",
		Status:       statusProcessing,
		CreatedAt:    "2026-06-10T10:00:00Z",
	}
	if err := store.put(job); err != nil {
		t.Fatalf("failed to put: %v", err)
	}
	if err := releaseJobStore(store); err != nil {
		t.Fatalf("failed to close: %v", err)
	}

	// Jobs survive across sessions
	store, err = acquireJobStore(dir, testEncryptionKey())
	if err != nil {
		t.Fatalf("failed to reopen the store: %v", err)
	}
	defer releaseJobStore(store)
	jobs, err := store.list()
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if len(jobs) != 1 || jobs[0].PredictionID != "persisted" {
		t.Fatalf("the job did not survive a reopen: %+v", jobs)
	}
}

func TestJobStoreSharedHandle(t *testing.T) {
	dir := t.TempDir()
	first, err := acquireJobStore(dir, testEncryptionKey())
	if err != nil {
		t.Fatalf("failed to open the store: %v", err)
	}
	// A second acquire from the same process must share the handle instead
	// of failing on Badger's directory lock
	second, err := acquireJobStore(dir, testEncryptionKey())
	if err != nil {
		t.Fatalf("failed to acquire the store a second time: %v", err)
	}
	if first != second {
		t.Fatalf("expected the same store handle")
	}

	if err := releaseJobStore(first); err != nil {
		t.Fatalf("failed to release: %v", err)
	}
	// Still usable through the second reference
	if err := second.put(&jobRecord{PredictionID: "x", CreatedAt: "2026-06-10T10:00:00Z"}); err != nil {
		t.Fatalf("the store should still be open: %v", err)
	}
	if err := releaseJobStore(second); err != nil {
		t.Fatalf("failed to release the last reference: %v", err)
	}
}
