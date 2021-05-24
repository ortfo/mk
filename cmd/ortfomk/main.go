package main

import (
	"os"
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
	data := ortfomk.GlobalData{translations, db}
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

		// Final newline
		println("")
	}
}
