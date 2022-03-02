package ortfomk

import (
	"fmt"
	"os"
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

func UpdateSpinner() {
	var message string
	switch g.Progress.Step {
	case StepBuildPage:
		message = fmt.Sprintf("Building page [magenta]%s[reset] as [magenta]%s[reset]", g.Progress.File, g.CurrentOutputFile)
	case StepDeadLinks:
		message = "Checking for dead links (this might take a while, disable it with DEADLINKS_CHECK=0)"
	case StepGeneratePDF:
		message = fmt.Sprintf("Generating PDF for [magenta]%s[reset] as [magenta]%s[reset]", g.Progress.File, g.CurrentOutputFile)
	case StepLoadExternalSites:
		message = fmt.Sprintf("Loading external sites from [magenta]%s[reset]", g.Progress.File)
	case StepLoadTags:
		message = fmt.Sprintf("Loading tags from [magenta]%s[reset]", g.Progress.File)
	case StepLoadTechnologies:
		message = fmt.Sprintf("Loading technologies from [magenta]%s[reset]", g.Progress.File)
	case StepLoadTranslations:
		message = fmt.Sprintf("Loading translations from [magenta]%s[reset]", g.Progress.File)
	case StepLoadWorks:
		message = fmt.Sprintf("Loading works from database [magenta]%s[reset]", g.Progress.File)
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
	} else if g.Progress.Step == StepBuildPage || g.Progress.Step == StepGeneratePDF {
		currentObjectType = "page"
	}

	fullMessage := colorstring.Color(fmt.Sprintf("[light_blue]%3d%%[reset] [bold][green]%s[reset] [bold]%s [yellow]%s[reset][bold][dim]:[reset] %s…", g.ProgressFileData().Percent, currentObjectType, g.CurrentObjectID, g.CurrentLanguage, message))
	g.Spinner.Message(fullMessage)
}

// LogError logs non-fatal errors.
func LogError(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[red]error[reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

// LogInfo logs infos.
func LogInfo(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[blue]info [reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

// LogDebug logs debug messages.
func LogDebug(message string, fmtArgs ...interface{}) {
	if os.Getenv("DEBUG") != "1" {
		return
	}
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[magenta]debug[reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}

// LogWarning logs warnings.
func LogWarning(message string, fmtArgs ...interface{}) {
	spinner.Pause()
	colorstring.Fprintf(os.Stderr, "\033[2K\r[yellow]warn [reset] [bold][dim](%s)[reset] %s\n", currentWorkID, fmt.Sprintf(message, fmtArgs...))
	spinner.Unpause()
}
