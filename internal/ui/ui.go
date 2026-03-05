package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

var (
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()
	faint   = color.New(color.Faint).SprintFunc()
	Verbose bool
)

func Info(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", cyan("ℹ"), fmt.Sprintf(format, a...))
}

func Success(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", green("✓"), fmt.Sprintf(format, a...))
}

func Error(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", red("✗"), fmt.Sprintf(format, a...))
}

func Warn(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s %s\n", yellow("⚠"), fmt.Sprintf(format, a...))
}

func Step(step int, total int, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintf(os.Stderr, "%s %s\n", bold(fmt.Sprintf("[%d/%d]", step, total)), msg)
}

func Debug(format string, a ...interface{}) {
	if Verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", faint("DEBUG"), fmt.Sprintf(format, a...))
	}
}

func Fatal(format string, a ...interface{}) {
	Error(format, a...)
	os.Exit(1)
}

type Spinner struct {
	s *spinner.Spinner
}

func NewSpinner(suffix string) *Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + suffix
	s.Writer = os.Stderr
	return &Spinner{s: s}
}

func (sp *Spinner) Start() {
	sp.s.Start()
}

func (sp *Spinner) Stop() {
	sp.s.Stop()
}

func (sp *Spinner) StopWithSuccess(msg string) {
	sp.s.Stop()
	Success("%s", msg)
}

func (sp *Spinner) StopWithError(msg string) {
	sp.s.Stop()
	Error("%s", msg)
}

func Header(title string) {
	fmt.Fprintf(os.Stderr, "\n%s\n", bold(title))
	fmt.Fprintf(os.Stderr, "%s\n", faint("─────────────────────────────────────────"))
}

func KeyValue(key, value string) {
	fmt.Fprintf(os.Stderr, "  %-20s %s\n", faint(key+":"), value)
}

func StatusLine(label, status string, ok bool) {
	indicator := green("●")
	if !ok {
		indicator = red("●")
	}
	fmt.Fprintf(os.Stderr, "  %s %-30s %s\n", indicator, label, faint(status))
}
