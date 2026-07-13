package savings

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndReadAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "savings.jsonl")

	e1 := Entry{Time: time.Now(), EstimatedTokens: 42, SimilarityPct: 62.5, SourceFile: "a.jsonl"}
	e2 := Entry{Time: time.Now(), EstimatedTokens: 17, SimilarityPct: 80.0, SourceFile: "b.jsonl"}

	if err := Append(path, e1); err != nil {
		t.Fatal(err)
	}
	if err := Append(path, e2); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAll(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].EstimatedTokens != 42 || entries[1].EstimatedTokens != 17 {
		t.Fatalf("got %+v", entries)
	}
}

func TestReadAllMissingFileIsNotAnError(t *testing.T) {
	entries, err := ReadAll(filepath.Join(t.TempDir(), "does-not-exist.jsonl"))
	if err != nil {
		t.Fatalf("a missing log file should not be an error, got %v", err)
	}
	if entries != nil {
		t.Fatalf("expected nil entries, got %+v", entries)
	}
}

func TestReadAllSkipsCorruptedLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "savings.jsonl")
	if err := Append(path, Entry{EstimatedTokens: 10}); err != nil {
		t.Fatal(err)
	}
	// append a corrupted line by hand
	appendRaw(t, path, "{not valid json\n")
	if err := Append(path, Entry{EstimatedTokens: 20}); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAll(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected the 2 good entries to survive a corrupted line between them, got %d", len(entries))
	}
}

func TestSummarize(t *testing.T) {
	entries := []Entry{
		{EstimatedTokens: 10},
		{EstimatedTokens: 25},
		{EstimatedTokens: 5},
	}
	s := Summarize(entries)
	if s.Count != 3 {
		t.Fatalf("Count = %d, want 3", s.Count)
	}
	if s.TotalTokens != 40 {
		t.Fatalf("TotalTokens = %d, want 40", s.TotalTokens)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := Summarize(nil)
	if s.Count != 0 || s.TotalTokens != 0 {
		t.Fatalf("expected zero summary for no entries, got %+v", s)
	}
}

func appendRaw(t *testing.T, path, raw string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.WriteString(raw); err != nil {
		t.Fatal(err)
	}
}
