package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"

	"github.com/docopt/docopt-go"
	ortfomk "github.com/ortfo/mk"
)

const CLIUsage = `
Usage:
	ortfomk (build|develop) <templates> with <database> to <destination> [--load=<filepath>]... [options]

Commands:
	build            Build the website
	develop          Watch for changes and re-build automatically

Arguments:
	<database>       Path to the database JSON file
	<templates>      Path to the directory containing .pug or .html template files
	<destination>    Path to the output directory, where the site will be built.

Options:
	--write-progress=<filepath>   Write current build progress to <filepath>
	--silent                      Don't output progress status to console
	--clean					      Clean the output directory before building
	--load=<filepath>             Path to a JSON or YAML file containing additional data which
							      will be available to templates as objects (or arrays) whose names will
								  be the files', but without the extension, and turned into camelCase
								  (e.g. "my-data.json"'s data is available as "myData").

Build Progress:
  For integration purposes, the current build progress can be written to a file.
  The progress information is written as JSON, and has the following structure:

	total: the total number of pages to process (multiple pages of the same template count as different pages).
	processed: the number of pages processed so far.
	percent: The current overall progress percentage of the build. Equal to processed/total * 100.
	current: {
		id: The id of the work being built.
		step: The current step. One of: "thumbnail", "color extraction", "description", "media analysis"
		resolution: The resolution of the thumbnail being generated. 0 when step is not "thumbnails"
		file: The file being processed (
			original media when making thumbnails or during media analysis,
			media the colors are being extracted from, or
			the description.md file when parsing description
		)
		output: The output file being generated.
		language: The current language in which the page is being built.
	}
`

func main() {
	defer func() {
		if r := recover(); r != nil {
			showCursor()
			ortfomk.LogFatal("ortfo/mk crashed… Here's why: %s", r)
		}
	}()
	usage := CLIUsage
	args, _ := docopt.ParseDoc(usage)
	isSilent, _ := args.Bool("--silent")
	clean, _ := args.Bool("--clean")
	progressFilePath, _ := args.String("--write-progress")
	outputDirectory, _ := args.String("<destination>")
	templatesDirectory, _ := args.String("<templates>")
	templatesDirectory, _ = filepath.Abs(templatesDirectory)
	flags := ortfomk.Flags{
		Silent:       isSilent,
		ProgressFile: progressFilePath,
	}
	additionalDataFiles, _ := args["--load"].([]string)
	additionalData, err := ortfomk.LoadAdditionalData(additionalDataFiles)
	if err != nil {
		ortfomk.LogFatal("couldn't load data files %v: %s", additionalDataFiles, err)
		return
	}

	ortfomk.WarmUp(&ortfomk.GlobalData{
		Flags:              flags,
		OutputDirectory:    outputDirectory,
		TemplatesDirectory: templatesDirectory,
		HTTPLinks:          make(map[string][]string),
		AdditionalData:     additionalData,
	})

	if os.Getenv("DEBUG") == "1" {
		cpuProfileFile, err := os.Create("ortfomk_cpu.pprof")
		if err != nil {
			panic(err)
		}
		defer cpuProfileFile.Close()
		pprof.StartCPUProfile(cpuProfileFile)
		defer pprof.StopCPUProfile()
	}
	//
	// Preparing dist directory
	//
	if _, err := os.Stat(outputDirectory); err == nil && clean {
		os.RemoveAll(outputDirectory)
	}
	//
	// Loading files
	//
	db, err := ortfomk.LoadDatabase("database")
	if err != nil {
		ortfomk.LogError("Could not load the database: %s", err)
		return
	}
	translations, err := ortfomk.LoadTranslations()
	if err != nil {
		ortfomk.LogError("Couldn't load the translation files: %s", err)
		return
	}
	ortfomk.SetTranslationsOnGlobalData(translations)
	ortfomk.SetDatabaseOnGlobalData(db)
	ortfomk.ComputeTotalToBuildCount()
	var httpLinks map[string][]string
	//
	// Watch mode
	//
	if val, _ := args.Bool("develop"); val {
		_, httpLinks, err = ortfomk.BuildAll(templatesDirectory, 0)
		if err != nil {
			ortfomk.LogError("During initial build: %s", err)
		}

		ortfomk.StartWatcher(db)
	} else {
		_, httpLinks, err = ortfomk.BuildAll(templatesDirectory, 0)

		if err != nil {
			ortfomk.LogError("While building: %s", err)
		}

		for _, lang := range []string{"fr", "en"} {
			// Save the updated .po file
			translations[lang].SavePO()
			// Save list of unused messages
			err = translations[lang].WriteUnusedMessages()
			if err != nil {
				ortfomk.LogError("While writing unused message file: %s", err)
			}
		}

		// Check for dead links
		if os.Getenv("DEADLINKS_CHECK") != "0" {
			ortfomk.Status(ortfomk.StepDeadLinks, ortfomk.ProgressDetails{})
			noneDead := true
			channel := make(chan string)
			var wg sync.WaitGroup
			workersCount := len(httpLinks)
			wg.Add(workersCount)

			for i := 0; i < workersCount; i++ {
				go func(c chan string) {
					for {
						link, more := <-c
						if !more {
							wg.Done()
							return
						}

						dead, err := ortfomk.IsLinkDead(link)
						if err != nil {
							ortfomk.LogError("could not check for dead link %q: %s", link, err)
						}
						if dead {
							noneDead = false
							ortfomk.LogInfo("- %s (from %s)", link, strings.Join(httpLinks[link], ", "))
						}
					}
				}(channel)
			}

			for link, _ := range httpLinks {
				channel <- link
			}

			close(channel)
			wg.Wait()

			if noneDead {
				ortfomk.LogInfo("No dead links found.")
			} else {
				ortfomk.LogInfo("are dead links.")
			}
		}

		ortfomk.CoolDown()
	}

	if os.Getenv("DEBUG") == "1" {
		heapProfileFile, err := os.Create("ortfomk_heap.pprof")
		if err != nil {
			panic(err)
		}
		defer heapProfileFile.Close()
		runtime.GC()
		if err := pprof.WriteHeapProfile(heapProfileFile); err != nil {
			ortfomk.LogFatal("couldn't write heap profile: %s", err)
		}
	}
}

// showCursor is used to make the cursor again without relying on the spinner working, in cases where the program crashes.
func showCursor() {
	fmt.Printf("\033[?25h")
}
