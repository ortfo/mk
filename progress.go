package ortfomk

import (
	"encoding/json"
	"io/ioutil"
	"math"
)

var spinner Spinner = DummySpinner{}
var currentWorkID string = ""

type BuildStep string

const (
	StepDeadLinks         BuildStep = "dead links"
	StepBuildPage         BuildStep = "build page"
	StepGeneratePDF       BuildStep = "pdf generation"
	StepLoadWorks         BuildStep = "load works"
	StepLoadTechnologies  BuildStep = "load technologies"
	StepLoadExternalSites BuildStep = "load external sites"
	StepLoadTags          BuildStep = "load tags"
	StepLoadTranslations  BuildStep = "load translations"
)

// ProgressFile holds the data that gets written to the progress file as JSON.
type ProgressFile struct {
	Total     int `json:"total"`
	Processed int `json:"processed"`
	Percent   int `json:"percent"`
	Current   struct {
		ID   string    `json:"id"`
		Step BuildStep `json:"step"`
		// The resolution of the thumbnail being generated. 0 when step is not "thumbnails"
		Resolution int `json:"resolution"`
		// The file being processed:
		//
		// - original media when making thumbnails or during media analysis,
		//
		// - media the colors are being extracted from, or
		//
		// - the description.md file when parsing description
		File     string `json:"file"`
		Language string `json:"language"`
		Output   string `json:"output"`
	} `json:"current"`
}

type ProgressDetails struct {
	Resolution int
	File       string
	Language   string
	OutFile    string
}

// Status updates the current progress and writes the progress to a file if --write-progress is set.
func Status(step BuildStep, details ProgressDetails) {
	g.Progress.Step = step
	g.Progress.Resolution = details.Resolution
	g.Progress.File = details.File
	if details.Language != "" {
		g.CurrentLanguage = details.Language
	}
	g.CurrentOutputFile = details.OutFile

	UpdateSpinner()
	err := WriteProgressFile()
	if err != nil {
		LogError("Couldn't write to progress file: %s", err)
	}
}

// IncrementProgress increments the number of processed works and writes the progress to a file if --write-progress is set.
func IncrementProgress() error {
	g.Progress.Current++

	UpdateSpinner()
	return WriteProgressFile()
}

// WriteProgressFile writes the progress to a file if --write-progress is set.
func WriteProgressFile() error {
	if g.Flags.ProgressFile == "" {
		return nil
	}

	progressDataJSON, err := json.Marshal(g.ProgressFileData())
	if err != nil {
		return err
	}

	return ioutil.WriteFile(g.Flags.ProgressFile, progressDataJSON, 0644)
}

// ProgressFileData returns a ProgressData struct ready to be marshalled to JSON for --write-progress.
func (g *GlobalData) ProgressFileData() ProgressFile {
	return ProgressFile{
		Total:     g.Progress.Total,
		Processed: g.Progress.Current,
		Percent:   int(math.Floor(float64(g.Progress.Current) / float64(g.Progress.Total) * 100)),
		Current: struct {
			ID         string    `json:"id"`
			Step       BuildStep `json:"step"`
			Resolution int       `json:"resolution"`
			File       string    `json:"file"`
			Language   string    `json:"language"`
			Output     string    `json:"output"`
		}{
			ID:         g.CurrentObjectID,
			Step:       g.Progress.Step,
			Resolution: g.Progress.Resolution,
			File:       g.Progress.File,
			Language:   g.CurrentLanguage,
			Output:     g.CurrentOutputFile,
		},
	}
}

// SetCurrentObjectID sets the current object ID and updates the spinner.
func SetCurrentObjectID(objectID string) {
	g.CurrentObjectID = objectID
	UpdateSpinner()
}
