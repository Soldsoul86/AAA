// Package savings persists every repeated prompt again detects, with its
// real, measured token count — not an estimate of tokens "saved" (that
// would require guessing what would have happened otherwise), just an
// honest record of tokens actually spent re-sending something already
// said. Stored cumulatively across projects at ~/.again/savings.jsonl,
// since again isn't tied to a single git repo the way checkpoint is.
package savings

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Time            time.Time `json:"time"`
	EstimatedTokens int       `json:"estimated_tokens"` // real measured length of the repeated prompt itself, /4
	SimilarityPct   float64   `json:"similarity_pct"`
	SourceFile      string    `json:"source_file"`
}

func LogPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".again", "savings.jsonl"), nil
}

// Append records one detected repeat. Never fails loudly to the caller in
// a way that would interrupt watching — logging a repeat is a secondary
// concern to detecting it live.
func Append(path string, e Entry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

func ReadAll(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			continue // skip malformed lines rather than fail the whole read
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

type Summary struct {
	Count       int
	TotalTokens int
}

func Summarize(entries []Entry) Summary {
	s := Summary{}
	for _, e := range entries {
		s.Count++
		s.TotalTokens += e.EstimatedTokens
	}
	return s
}
