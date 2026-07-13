// Package detector implements loop detection over a stream of terminal
// output lines. It is deliberately conservative: a redrawn spinner (the
// same line repeated on consecutive frames) never counts as a loop, and
// once a stuck pattern has been reported it will not report the exact same
// episode again until that line has actually aged out of the window.
package detector

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b\][^\a]*\a|\r`)

// Normalize strips ANSI escape sequences and carriage returns, then trims
// surrounding whitespace, so visually-identical lines compare equal even if
// their raw bytes differ (color codes, cursor moves, trailing \r).
func Normalize(line string) string {
	return strings.TrimSpace(ansiRe.ReplaceAllString(line, ""))
}

// Detector tracks a sliding window of recently-seen normalized lines.
type Detector struct {
	Threshold int // how many occurrences within Window count as a loop
	Window    int // how many recent distinct lines to remember

	history  []string
	lastLine string
	alerted  map[string]bool
}

func New(threshold, window int) *Detector {
	if threshold < 1 {
		threshold = 1
	}
	if window < 1 {
		window = 1
	}
	return &Detector{Threshold: threshold, Window: window, alerted: map[string]bool{}}
}

// Feed processes one raw line of output. If this line has just reached the
// repetition threshold for the first time in its current run, it returns
// (normalizedLine, occurrenceCount, true).
func (d *Detector) Feed(rawLine string) (string, int, bool) {
	line := Normalize(rawLine)
	if line == "" {
		return "", 0, false
	}

	// Consecutive identical lines (a redrawn spinner, a progress bar) are a
	// single event, not repeated evidence of being stuck.
	if line == d.lastLine {
		return "", 0, false
	}
	d.lastLine = line

	d.history = append(d.history, line)
	if len(d.history) > d.Window {
		dropped := d.history[0]
		d.history = d.history[1:]
		if !contains(d.history, dropped) {
			delete(d.alerted, dropped)
		}
	}

	count := 0
	for _, h := range d.history {
		if h == line {
			count++
		}
	}

	if count >= d.Threshold && !d.alerted[line] {
		d.alerted[line] = true
		return line, count, true
	}
	return "", 0, false
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
