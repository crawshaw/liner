package liner

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"
)

type History interface {
	// AppendHistory appends an entry to the scrollback history.
	// AppendHistory should be called iff Prompt returns a valid command.
	AppendHistory(item string)

	// FindByPrefix returns the history lines starting with prefix.
	FindByPrefix(prefix string) []string

	// FindByPrefix returns the history lines matching pattern.
	FindByPattern(pattern string) (res []string, pos []int)
}

type sliceHistory struct {
	mu      sync.RWMutex
	history []string
}

// ReadHistory reads scrollback history from r. Returns the number of lines
// read, and any read error (except io.EOF).
func (h *sliceHistory) ReadHistory(r io.Reader) (num int, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	in := bufio.NewReader(r)
	num = 0
	for {
		line, part, err := in.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return num, err
		}
		if part {
			return num, fmt.Errorf("line %d is too long", num+1)
		}
		if !utf8.Valid(line) {
			return num, fmt.Errorf("invalid string at line %d", num+1)
		}
		num++
		h.history = append(h.history, string(line))
		if len(h.history) > HistoryLimit {
			h.history = h.history[1:]
		}
	}
	return num, nil
}

// WriteHistory writes scrollback history to w. Returns the number of lines
// successfully written, and any write error.
//
// Unlike the rest of liner's API, WriteHistory is safe to call
// from another goroutine while Prompt is in progress.
// This exception is to facilitate the saving of the history buffer
// during an unexpected exit (for example, due to Ctrl-C being invoked)
func (h *sliceHistory) WriteHistory(w io.Writer) (num int, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, item := range h.history {
		_, err := fmt.Fprintln(w, item)
		if err != nil {
			return num, err
		}
		num++
	}
	return num, nil
}

// AppendHistory implements History.AppendHistory.
func (h *sliceHistory) AppendHistory(item string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.history) > 0 {
		if item == h.history[len(h.history)-1] {
			return
		}
	}
	h.history = append(h.history, item)
	if len(h.history) > HistoryLimit {
		h.history = h.history[1:]
	}
}

// ClearHistory clears the scrollback history.
func (h *sliceHistory) ClearHistory() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.history = nil
}

// FindByPrefix implements History.FindByPrefix.
func (h *sliceHistory) FindByPrefix(prefix string) (ph []string) {
	for _, h := range h.history {
		if strings.HasPrefix(h, prefix) {
			ph = append(ph, h)
		}
	}
	return ph
}

// FindByPattern implements History.FindByPattern.
func (h *sliceHistory) FindByPattern(pattern string) (ph []string, pos []int) {
	if pattern == "" {
		return
	}
	for _, h := range h.history {
		if i := strings.Index(h, pattern); i >= 0 {
			ph = append(ph, h)
			pos = append(pos, i)
		}
	}
	return ph, pos
}
