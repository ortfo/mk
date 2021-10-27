package main

import (
	"os"
	"strings"

	"github.com/ortfo/mk"
)

func main() {
	//
	// Preparing dist directory
	//
	err := os.MkdirAll("dist/fr/using", 0777)
	if err != nil {
		printerr("Couldn't create directories for writing", err)
		return
	}
	os.MkdirAll("dist/en/using", 0777)
	//
	// Loading files
	//
	db, err := ortfomk.LoadDatabase("database")
	if err != nil {
		printerr("Could not load the database", err)
		return
	}
	translations, err := ortfomk.LoadTranslations()
	if err != nil {
		printerr("Couldn't load the translation files", err)
	}
	data := ortfomk.GlobalData{
		Translations: translations,
		Database:     db,
		HTTPLinks:    make(map[string][]string),
	}
	//
	// Watch mode
	//
	if len(os.Args) >= 2 && os.Args[1] == "watch" {
		ortfomk.StartWatcher(db)
	} else {
		_, err := data.BuildAll("/home/ewen/projects/portfolio/src")

		if err != nil {
			printerr("While building", err)
		}

		// Save the updated .po file
		data.SavePO("i18n/fr.po")
		// Save list of unused msgids
		err = data.WriteUnusedMsgIds("i18n/unused-msgids.yaml")
		if err != nil {
			printerr("While writing unused msgids file", err)
		}

		// Check for dead links
		if os.Getenv("DEADLINKS_CHECK") != "0" {
			printfln("Checking for dead links… (this might take a while, disable it with DEADLINKS_CHECK=0)")
			noneDead := true
			for link, sites := range data.HTTPLinks {
				dead, err := ortfomk.IsLinkDead(link)
				if err != nil {
					printerr("could not check for dead links", err)
				}
				if !dead {
					continue
				}

				noneDead = false
				printfln("- %s (from %s),", link, strings.Join(sites, ", "))

			}

			if noneDead {
				printfln("No dead links found.")
			} else {
				printfln("are dead links.")
			}
		}

		// Final newline
		println("")
	}
}
