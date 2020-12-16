package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/joho/godotenv"
)

func IsVerbose() bool {
	// return len(os.Args) > 1 && os.Args[1] == "--verbose"
	return false // TODO
}

func VerboseLog(s string, fmtArgs ...interface{}) {
	if IsVerbose() {
		fmt.Printf(s+"\n", fmtArgs...)
	}
}

func main() {
	db, err := LoadDatabase("../database/database.json")
	if err != nil {
		panic(err)
	}
	files, err := ioutil.ReadDir("../src")
	if err != nil {
		panic(err)
	}
	VerboseLog("------")
	if BuildingForProduction() {
		VerboseLog("Hydrating for production")
	} else {
		VerboseLog("Hydrating for developement")
	}
	VerboseLog("------")
	MakeDirs([]string{"fr", "en"})
	for _, lang := range []string{"fr", "en"} {
		VerboseLog("[hydrator]     language: " + lang)
		for _, file := range files {
			if file.IsDir() && file.Name() == "using" {
				files, err := ioutil.ReadDir("../src/using")
				if err != nil {
					panic(err)
				}
				for _, file := range files {
					if file.Name() == "_technology.pug" {
						templateContent, err := ReadFile("../src/using/_technology.pug")
						templateName := "using/_technology.pug"
						if err != nil {
							panic(err)
						}
						for _, tech := range KnownTechnologies {
							LogHydrating(templateName, tech.URLName)
							HydrateDynamicFileWithLang(db, lang, templateName, templateContent, CurrentlyHydrated{tech: tech})
						}
					} else {
						templateContent, err := ReadFile("../src/using/" + file.Name())
						templateName := "using/" + file.Name()
						LogHydrating(templateName, "")
						replaced, err := ExecuteTemplate(db, lang, file.Name(), templateContent, CurrentlyHydrated{})
						if err != nil {
							panic(err)
						}
						WriteHydratedContent(lang, templateName, replaced)
					}
				}
			} else if strings.HasSuffix(file.Name(), ".pug") {
				templateContent, err := ReadFile("../src/" + file.Name())
				if err != nil {
					panic(err)
				}
				templateName := file.Name()
				if templateName == "_work.pug" {
					for _, work := range GetOneLang(lang, db.Works...) {
						LogHydrating(templateName, work.ID)
						HydrateDynamicFileWithLang(db, lang, templateName, templateContent, CurrentlyHydrated{work: work})
					}
				} else if templateName == "_tag.pug" {
					for _, tag := range KnownTags {
						LogHydrating(templateName, tag.URLName())
						HydrateDynamicFileWithLang(db, lang, templateName, templateContent, CurrentlyHydrated{tag: tag})
					}
				} else if templateName == ".gallery.pug" {
					continue
				} else {
					LogHydrating(templateName, "")
					replaced, err := ExecuteTemplate(db, lang, file.Name(), templateContent, CurrentlyHydrated{})
					if err != nil {
						panic(err)
					}
					WriteHydratedContent(lang, templateName, replaced)
				}

			}
		}
	}
}

type CurrentlyHydrated struct {
	tag  Tag
	tech Technology
	work WorkOneLang
}

func HydrateDynamicFileWithLang(db Database, language string, templateName string, templateContent []byte, currentlyHydrated CurrentlyHydrated) {
	// Execute template
	replaced, err := ExecuteTemplate(db, language, templateName, templateContent, currentlyHydrated)
	if err != nil {
		panic(err)
	}

	// determine where the destination file(path) name
	var pathIdentifier string
	if currentlyHydrated.work.ID != "" {
		pathIdentifier = currentlyHydrated.work.ID
	} else if currentlyHydrated.tag.URLName() != "" {
		pathIdentifier = currentlyHydrated.tag.URLName()
	} else {
		pathIdentifier = "using/" + currentlyHydrated.tech.URLName
	}

	WriteHydratedContent(language, pathIdentifier, replaced)
}

func BuildingForProduction() bool {
	err := godotenv.Load("./.env")
	if err != nil {
		panic("Could not load the .env file")
	}
	return os.Getenv("ENVIRONMENT") != "dev"
}

func ExecuteTemplate(db Database, language string, templateName string, templateContent []byte, currentlyHydrated CurrentlyHydrated) (string, error) {
	var currentlyHydratedStuff string
	if currentlyHydrated.tag.Singular != "" {
		currentlyHydratedStuff = "<" + currentlyHydrated.tag.Singular + ">"
	} else if currentlyHydrated.work.ID != "" {
		currentlyHydratedStuff = "<" + currentlyHydrated.work.ID + ">"
	} else if currentlyHydrated.tech.URLName != "" {
		currentlyHydratedStuff = "<" + currentlyHydrated.tech.URLName + ">"
	} else {
		currentlyHydratedStuff = ""
	}
	tmpl := template.Must(
		template.New(templateName + currentlyHydratedStuff).Funcs(GetTemplateFuncMap()).Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap{
			"tindent":  IndentWithTabs,
			"tnindent": IndentWithTabsNewline,
		}).Parse(string(templateContent)))
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, TemplateData{
		KnownTags:         KnownTags,
		KnownTechnologies: KnownTechnologies,
		Works:             GetOneLang(language, db.Works...),
		Age:               GetAge(),
		CurrentTag:        currentlyHydrated.tag,
		CurrentTech:       currentlyHydrated.tech,
		CurrentWork:       currentlyHydrated.work,
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func WriteHydratedContent(language string, templateName string, replacedString string) {
	file, err := os.Create("../artifacts/phase_1/" + language + "/" + strings.TrimSuffix(templateName, ".pug") + ".pug")
	if err != nil {
		panic(err)
	}
	_, err = file.WriteString(replacedString)
	if err != nil {
		panic(err)
	}
}

func MakeDirs(languages []string) {
	for _, lang := range languages {
		os.MkdirAll("../artifacts/phase_1/"+lang+"/using", 0777) // TODO: 0777 is evil
	}
}

func LogHydrating(filename string, identifier string) {
	if identifier != "" {
		VerboseLog("[hydrator]     hydrating: '%s' @ %s\n", filename, identifier)
	} else {
		VerboseLog("[hydrator]     hydrating: '%s'\n", filename)
	}
}
