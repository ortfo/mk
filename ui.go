package ortfomk

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/mitchellh/colorstring"
	"github.com/theckman/yacspin"
)

// A yacspin spinner or a dummy spinner that does nothing.
// Used to avoid having to check for nil pointers everywhere when --silent is set.
type Spinner interface {
	Start() error
	Stop() error
	Message(string)
	Pause() error
	Unpause() error
}

type DummySpinner struct {
}

func (d DummySpinner) Start() error   { return nil }
func (d DummySpinner) Stop() error    { return nil }
func (d DummySpinner) Message(string) {}
func (d DummySpinner) Pause() error   { return nil }
func (d DummySpinner) Unpause() error { return nil }

func CreateSpinner() Spinner {
	writer := os.Stdout

	// Don't clog stdout if we're not in a tty
	if !isatty.IsTerminal(os.Stdout.Fd()) {
		writer = os.Stderr
	}

	spinner, err := yacspin.New(yacspin.Config{
		Writer:            writer,
		Frequency:         100 * time.Millisecond,
		Suffix:            " ",
		Message:           "  0% Warming up",
		CharSet:           yacspin.CharSets[14],
		Colors:            []string{"fgCyan"},
		StopCharacter:     "✓",
		StopColors:        []string{"fgGreen"},
		StopMessage:       colorstring.Color(fmt.Sprintf("Website built to [bold]./%s[reset]", g.OutputDirectory)),
		StopFailCharacter: "✗",
		StopFailColors:    []string{"fgRed"},
	})

	if err != nil {
		LogError("Couldn't start spinner: %s", err)
		return DummySpinner{}
	}
	if g.Flags.Silent {
		return DummySpinner{}
	}

	return spinner
}

// absorb absorbs errors and returns the inner value, silently ignoring errors.
// It's kind of like Rust's .unwrap_or_default().
// Use wisely and as sparsely as possible.
func absorb[T any](value T, err error) (val T) {
	if err == nil {
		val = value
	}
	return
}

func UpdateSpinner() {
	var message string
	cwdRel := func(p string) string {
		if pretty, err := filepath.Rel(absorb(os.Getwd()), p); err == nil {
			return pretty
		} else {
			return p
		}
	}
	switch g.Progress.Step {
	case StepBuildPage:
		message = fmt.Sprintf("Building page [magenta]%s[reset] as [magenta]%s[reset]", cwdRel(g.Progress.File), cwdRel(g.CurrentOutputFile))
	case StepDeadLinks:
		message = "Checking for dead links (this might take a while, disable it with DEADLINKS_CHECK=0)"
	case StepGeneratePDF:
		message = fmt.Sprintf("Generating PDF for [magenta]%s[reset] as [magenta]%s[reset]", cwdRel(g.Progress.File), cwdRel(g.CurrentOutputFile))
	case StepLoadExternalSites:
		message = fmt.Sprintf("Loading external sites from [magenta]%s[reset]", cwdRel(g.Progress.File))
	case StepLoadTags:
		message = fmt.Sprintf("Loading tags from [magenta]%s[reset]", cwdRel(g.Progress.File))
	case StepLoadTechnologies:
		message = fmt.Sprintf("Loading technologies from [magenta]%s[reset]", cwdRel(g.Progress.File))
	case StepLoadTranslations:
		message = fmt.Sprintf("Loading translations from [magenta]%s[reset]", cwdRel(g.Progress.File))
	case StepLoadWorks:
		message = fmt.Sprintf("Loading works from database [magenta]%s[reset]", cwdRel(g.Progress.File))
	case StepLoadCollections:
		message = fmt.Sprintf("Loading work collections from [magenta]%s[reset]", cwdRel(g.Progress.File))
	default:
		message = string(g.Progress.Step)
	}
	var currentObjectType = ""
	if strings.Contains(g.Progress.File, ":work") {
		currentObjectType = "work"
	} else if strings.Contains(g.Progress.File, ":tag") {
		currentObjectType = "tag"
	} else if strings.Contains(g.Progress.File, ":tech") {
		currentObjectType = "tech"
	} else if strings.Contains(g.Progress.File, ":site") {
		currentObjectType = "site"
	} else if strings.Contains(g.Progress.File, ":collection") {
		currentObjectType = "collection"
	} else if g.Progress.Step == StepBuildPage || g.Progress.Step == StepGeneratePDF {
		currentObjectType = "page"
	}

	fullMessage := colorstring.Color(fmt.Sprintf(
		"[light_blue]%3d%%[reset] [bold][green]%s[reset] [bold]%s [yellow]%s[reset][bold][dim]:[reset] %s…",
		g.ProgressPercent(),
		currentObjectType,
		g.CurrentObjectID,
		g.CurrentLanguage,
		message,
	))
	g.Spinner.Message(fullMessage)
}

// LogError logs non-fatal errors.
func LogError(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[red]error[reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

// LogFatal logs fatal errors.
func LogFatal(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[invert][bold][red]crash[reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

// LogInfo logs infos.
func LogInfo(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[blue]info [reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

var lastDebugTimestamp time.Time = time.Now()

// LogDebug logs debug messages.
func LogDebug(message string, fmtArgs ...interface{}) {
	if os.Getenv("DEBUG") != "1" {
		return
	}
	spinner.Pause()
	duration := time.Since(lastDebugTimestamp)
	colorstring.Fprintf(os.Stderr, "\033[2K\r[magenta]debug[reset] [bold][dim](%s) %s[reset] %s\n", currentWorkID, duration.String(), fmt.Sprintf(message, fmtArgs...))
	lastDebugTimestamp = time.Now()
	spinner.Unpause()
}

// LogWarning logs warnings.
func LogWarning(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[yellow]warn [reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

// codeSpinnetAround shows a 10-line code exerpt around the given line.
// Line numbers are displayed on the left.
func codeSpinnetAround(file string, lineNumber uint64, columnNumber uint64) string {
	output := ""
	for i, line := range strings.Split(file, "\n") {
		if uint64(i) >= lineNumber-1-5 && uint64(i) <= lineNumber-1+5 {
			line := fmt.Sprintf("%3d | %s\n", i+1, truncateLineAround(line, int(columnNumber)-1, 200))
			if uint64(i) == lineNumber-1 {
				line = "> " + line
			} else {
				line = "  " + line
			}
			output += line
		}
	}
	return output
}

// truncateLineAround truncates a line to a maximum length, centered around the character the given index.
// when the line exceeds the maximum length, it is truncated at the given index.
func truncateLineAround(line string, columnIndex int, maxLength int) string {
	if len(line) <= maxLength {
		return line
	}
	var ellipsisLeft, ellipsisRight bool
	start := columnIndex - maxLength/2
	if start < 0 {
		start = 0
	} else {
		ellipsisLeft = true
	}
	end := start + maxLength
	if end > len(line) {
		end = len(line)
	} else {
		ellipsisRight = true
	}
	output := line[start:end]
	if ellipsisLeft {
		output = "…" + output
	}
	if ellipsisRight {
		output = output + "…"
	}
	return output
}
