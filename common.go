/*
Package liner implements a simple command line editor, inspired by linenoise
(https://github.com/antirez/linenoise/). This package supports WIN32 in
addition to the xterm codes supported by everything else.
*/
package liner

import (
	"bufio"
	"container/ring"
	"errors"
	"fmt"
)

type commonState struct {
	terminalSupported bool
	outputRedirected  bool
	inputRedirected   bool
	history           History
	completer         WordCompleter
	columns           int
	killRing          *ring.Ring
	ctrlCAborts       bool
	r                 *bufio.Reader
	tabStyle          TabStyle
	multiLineMode     bool
	cursorRows        int
	maxRows           int
	shouldRestart     ShouldRestart
	noBeep            bool
	needRefresh       bool
}

// TabStyle is used to select how tab completions are displayed.
type TabStyle int

// Two tab styles are currently available:
//
// TabCircular cycles through each completion item and displays it directly on
// the prompt
//
// TabPrints prints the list of completion items to the screen after a second
// tab key is pressed. This behaves similar to GNU readline and BASH (which
// uses readline)
const (
	TabCircular TabStyle = iota
	TabPrints
)

// ErrPromptAborted is returned from Prompt or PasswordPrompt when the user presses Ctrl-C
// if SetCtrlCAborts(true) has been called on the State
var ErrPromptAborted = errors.New("prompt aborted")

// ErrNotTerminalOutput is returned from Prompt or PasswordPrompt if the
// platform is normally supported, but stdout has been redirected
var ErrNotTerminalOutput = errors.New("standard output is not a terminal")

// ErrInvalidPrompt is returned from Prompt or PasswordPrompt if the
// prompt contains any unprintable runes (including substrings that could
// be colour codes on some platforms).
var ErrInvalidPrompt = errors.New("invalid prompt")

// ErrInternal is returned when liner experiences an error that it cannot
// handle. For example, if the number of colums becomes zero during an
// active call to Prompt
var ErrInternal = errors.New("liner: internal error")

// KillRingMax is the max number of elements to save on the killring.
const KillRingMax = 60

// HistoryLimit is the maximum number of entries saved in the scrollback history.
const HistoryLimit = 1000

func (s *State) getHistoryByPrefix(prefix string) []string {
	return s.history.FindByPrefix(prefix)
}

// Returns the history lines matching the intelligent search
func (s *State) getHistoryByPattern(pattern string) (ph []string, pos []int) {
	return s.history.FindByPattern(pattern)
}

// Completer takes the currently edited line content at the left of the cursor
// and returns a list of completion candidates.
// If the line is "Hello, wo!!!" and the cursor is before the first '!', "Hello, wo" is passed
// to the completer which may return {"Hello, world", "Hello, Word"} to have "Hello, world!!!".
type Completer func(line string) []string

// WordCompleter takes the currently edited line with the cursor position and
// returns the completion candidates for the partial word to be completed.
// If the line is "Hello, wo!!!" and the cursor is before the first '!', ("Hello, wo!!!", 9) is passed
// to the completer which may returns ("Hello, ", {"world", "Word"}, "!!!") to have "Hello, world!!!".
type WordCompleter func(line string, pos int) (head string, completions []string, tail string)

// SetCompleter sets the completion function that Liner will call to
// fetch completion candidates when the user presses tab.
func (s *State) SetCompleter(f Completer) {
	if f == nil {
		s.completer = nil
		return
	}
	s.completer = func(line string, pos int) (string, []string, string) {
		return "", f(string([]rune(line)[:pos])), string([]rune(line)[pos:])
	}
}

// SetWordCompleter sets the completion function that Liner will call to
// fetch completion candidates when the user presses tab.
func (s *State) SetWordCompleter(f WordCompleter) {
	s.completer = f
}

// SetTabCompletionStyle sets the behvavior when the Tab key is pressed
// for auto-completion.  TabCircular is the default behavior and cycles
// through the list of candidates at the prompt.  TabPrints will print
// the available completion candidates to the screen similar to BASH
// and GNU Readline
func (s *State) SetTabCompletionStyle(tabStyle TabStyle) {
	s.tabStyle = tabStyle
}

// ModeApplier is the interface that wraps a representation of the terminal
// mode. ApplyMode sets the terminal to this mode.
type ModeApplier interface {
	ApplyMode() error
}

// SetCtrlCAborts sets whether Prompt on a supported terminal will return an
// ErrPromptAborted when Ctrl-C is pressed. The default is false (will not
// return when Ctrl-C is pressed). Unsupported terminals typically raise SIGINT
// (and Prompt does not return) regardless of the value passed to SetCtrlCAborts.
func (s *State) SetCtrlCAborts(aborts bool) {
	s.ctrlCAborts = aborts
}

// SetMultiLineMode sets whether line is auto-wrapped. The default is false (single line).
func (s *State) SetMultiLineMode(mlmode bool) {
	s.multiLineMode = mlmode
}

// ShouldRestart is passed the error generated by readNext and returns true if
// the the read should be restarted or false if the error should be returned.
type ShouldRestart func(err error) bool

// SetShouldRestart sets the restart function that Liner will call to determine
// whether to retry the call to, or return the error returned by, readNext.
func (s *State) SetShouldRestart(f ShouldRestart) {
	s.shouldRestart = f
}

// SetBeep sets whether liner should beep the terminal at various times (output
// ASCII BEL, 0x07). Default is true (will beep).
func (s *State) SetBeep(beep bool) {
	s.noBeep = !beep
}

func (s *State) promptUnsupported(p string) (string, error) {
	if !s.inputRedirected || !s.terminalSupported {
		fmt.Print(p)
	}
	linebuf, _, err := s.r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(linebuf), nil
}
