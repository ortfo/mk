package main

import (
	"os"

	"github.com/docopt/docopt-go"
	ortfomk "github.com/ortfo/mk"
)

const CLIUsage = `
Usage:
	ortfomk <templates> (build|develop) with <database> to <destination> [options]

Commands:
	build            Build the website
	develop          Watch for changes and re-build automatically

Arguments:
	<database>       Path to the database JSON file
	<templates>      Path to .pug or .html template files
	<destination>    Path to the output directory, where the site will be built.

Options:
	--write-progress=<filepath>   Write current build progress to <filepath>
	--silent                      Don't output progress status to console

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
	usage := CLIUsage
	args, _ := docopt.ParseDoc(usage)
	isSilent, _ := args.Bool("--silent")
	progressFilePath, _ := args.String("--write-progress")
	outputDirectory, _ := args.String("<destination>")
	templatesDirectory, _ := args.String("<templates>")
	flags := ortfomk.Flags{
		Silent:       isSilent,
		ProgressFile: progressFilePath,
	}
	ortfomk.WarmUp(ortfomk.GlobalData{
		Flags:              flags,
		OutputDirectory:    outputDirectory,
		TemplatesDirectory: templatesDirectory,
		HTTPLinks:          make(map[string][]string),
	})

	//
	// Preparing dist directory
	//
	err := os.MkdirAll("dist/fr/using", 0777)
	if err != nil {
		ortfomk.LogError("Couldn't create directories for writing: %s", err)
		return
	}
	os.MkdirAll("dist/en/using", 0777)
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
	//
	// Watch mode
	//
	if val, _ := args.Bool("develop"); val {
		ortfomk.StartWatcher(db)
	} else {
		_, err := ortfomk.BuildAll(templatesDirectory)

		if err != nil {
			ortfomk.LogError("While building: %s", err)
		}

		// Save the updated .po file
		translations.SavePO("i18n/fr.po")
		// Save list of unused msgids
		err = translations.WriteUnusedMsgIds("i18n/unused-msgids.yaml")
		if err != nil {
			ortfomk.LogError("While writing unused msgids file: %s", err)
		}

		// Check for dead links
		if os.Getenv("DEADLINKS_CHECK") != "0" {
			ortfomk.Status(ortfomk.StepDeadLinks, ortfomk.ProgressDetails{})
			noneDead := true
			// FIXME
			// for link, sites := range g.HTTPLinks {
			// 	dead, err := ortfomk.IsLinkDead(link)
			// 	if err != nil {
			// 		ortfomk.LogError("could not check for dead links: %s", err)
			// 	}
			// 	if !dead {
			// 		continue
			// 	}

			// 	noneDead = false
			// 	ortfomk.LogInfo("- %s (from %s),", link, strings.Join(sites, ", "))

			// }

			if noneDead {
				ortfomk.LogInfo("No dead links found.")
			} else {
				ortfomk.LogInfo("are dead links.")
			}
		}

		ortfomk.CoolDown()
	}
}
