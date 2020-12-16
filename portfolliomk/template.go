package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/davecgh/go-spew/spew"
	"github.com/jaytaylor/html2text"
)

// GetTemplateFuncMap returns the funcmap used to hydrate files
func GetTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		// "into" transforms to HTML structures
		"intoGallery":   intoGallery,
		"intoLayout":    intoLayout,
		"intoColorsCSS": intoColorsCSS,
		// "get" gets a Go value (string, map[string]string, etc.)
		"getColorsMap":       getColorsMap,
		"getSummary":         getSummary,
		"getThumbnailSource": getThumbnailSource,
		"getYears":           getYears,
		// "with" filters a []WorkOneLang
		"withTag":       withTag,
		"withTech":      withTech,
		"withWIPStatus": withWIPStatus,
		"excluding":     excluding,
		// Nice, cosy aliases for filters
		"withCreatedYear": withCreatedYear,
		"tagged":          withTag,
		"madeWith":        withTech,
		"createdIn":       withCreatedYear,
		"finished": func(ws []WorkOneLang) []WorkOneLang {
			return withWIPStatus(false, ws)
		},
		"unfinished": func(ws []WorkOneLang) []WorkOneLang {
			return withWIPStatus(true, ws)
		},
		// reduces a []WorkOneLang down to a single WorkOneLang
		"latest": latest,
		// functions acting on paths
		"asset": asset,
		"media": media,
		// lookups for tags & technologies
		"lookupTag":  lookupTag,
		"lookupTech": lookupTech,
		// debugging
		"log": log,
		// various
		"makeWS":   makeWorkSlice,
		"appendWS": appendWorkSlice,
	}
}

type TemplateData struct {
	Age               uint8
	KnownTags         [len(KnownTags)]Tag
	KnownTechnologies [len(KnownTechnologies)]Technology
	Works             []WorkOneLang
	MusicTag          Tag
	// Template data for _-prefixed .pug files: relevant struct instance of what's being hydrated
	CurrentTag  Tag
	CurrentTech Technology
	CurrentWork WorkOneLang
}

func GetAge() uint8 {
	// TODO Do it dynamically
	return 17
}

func intoGallery(customClasses string, ws []WorkOneLang) string {
	templateContent, err := ReadFile("../src/.gallery.pug")

	// Hydrate .gallery.pug with ws
	tmpl := template.Must(
		template.New(".gallery.pug").Funcs(GetTemplateFuncMap()).Funcs(sprig.TxtFuncMap()).Funcs(template.FuncMap{
			"tindent":  IndentWithTabs,
			"tnindent": IndentWithTabsNewline,
		}).Parse(string(templateContent)))
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, struct {
		GivenWorks    []WorkOneLang
		KnownTags     [len(KnownTags)]Tag
		CustomClasses string
	}{
		GivenWorks:    ws,
		KnownTags:     KnownTags,
		CustomClasses: customClasses,
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func intoLayout(w WorkOneLang) string {
	return w.BuildLayout()
}

func intoColorsCSS(w WorkOneLang) string {
	var cssDeclaration string
	for key, value := range getColorsMap(w) {
		cssDeclaration += fmt.Sprintf("--%s:%s;", key, value)
	}
	return cssDeclaration
}

// getColorsMap returns a mapping of "primary", "secondary", etc to the color values,
// with an added "#" prefix if needed
func getColorsMap(w WorkOneLang) map[string]string {
	colorsMap := make(map[string]string, 3)
	if w.Metadata.Colors.Primary != "" {
		colorsMap["primary"] = AddOctothorpeIfNeeded(w.Metadata.Colors.Primary)
	}
	if w.Metadata.Colors.Secondary != "" {
		colorsMap["secondary"] = AddOctothorpeIfNeeded(w.Metadata.Colors.Secondary)
	}
	if w.Metadata.Colors.Tertiary != "" {
		colorsMap["tertiary"] = AddOctothorpeIfNeeded(w.Metadata.Colors.Tertiary)
	}
	return colorsMap
}

// getSummary summarizes the given work's first description paragraph
func getSummary(w WorkOneLang) string {
	if len(w.Paragraphs) == 0 {
		return ""
	}
	plainText, err := html2text.FromString(w.Paragraphs[0].Content, html2text.Options{OmitLinks: true})
	if err != nil {
		panic(err)
	}
	return SummarizeString(plainText, 150)
}

// getThumbnailSource gets the source URL of the work's first media
func getThumbnailSource(w WorkOneLang) string {
	if len(w.Media) == 0 {
		return ""
	}
	return media(w.Media[0].Source)
}

func getYears(ws []WorkOneLang) []int {
	years := make([]int, 0)
	for _, work := range ws {
		var isDuplicate bool
		for _, year := range years {
			if work.Created().Year() == year {
				isDuplicate = true
				continue
			}
		}
		if !isDuplicate {
			years = append(years, work.Created().Year())
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years
}

// withTag returns an array of works that have tag in their tags
func withTag(tag Tag, ws []WorkOneLang) []WorkOneLang {
	filtered := make([]WorkOneLang, 0)
	for _, work := range ws {
		for _, tagName := range work.Metadata.Tags {
			if tag.ReferredToBy(tagName) {
				filtered = append(filtered, work)
			}
		}
	}
	if len(filtered) == 0 && IsVerbose() {
		wsIDs := make([]string, len(ws))
		for _, work := range ws {
			wsIDs = append(wsIDs, work.ID)
		}
		fmt.Printf("WARNING: No works from %v have the %s tag\n", wsIDs, tag.URLName())
	}
	return filtered
}

// withTag returns an array of works that have tech in their "made with" technologies list
func withTech(tech Technology, ws []WorkOneLang) []WorkOneLang {
	filtered := make([]WorkOneLang, 0)
	for _, work := range ws {
		for _, techName := range work.Metadata.MadeWith {
			if tech.ReferredToBy(techName) {
				filtered = append(filtered, work)
			}
		}
	}
	return filtered
}

func withWIPStatus(wipStatus bool, ws []WorkOneLang) []WorkOneLang {
	filtered := make([]WorkOneLang, 0)
	for _, work := range ws {
		if work.IsWIP() == wipStatus {
			filtered = append(filtered, work)
		}
	}
	return filtered
}

func withCreatedYear(createdYear int, ws []WorkOneLang) []WorkOneLang {
	filtered := make([]WorkOneLang, 0)
	for _, work := range ws {
		if work.Created().Year() == createdYear {
			filtered = append(filtered, work)
		}
	}
	return filtered
}

// excluding returns the given works excluding those given in excludelist, comparing IDs.
func excluding(excludelist []WorkOneLang, ws []WorkOneLang) []WorkOneLang {
	filtered := make([]WorkOneLang, len(ws))
	for _, work := range ws {
		excluded := false
		for _, excludedWork := range excludelist {
			if work.ID == excludedWork.ID {
				excluded = true
			}
		}
		if !excluded {
			filtered = append(filtered, work)
		}
	}
	return filtered
}

func latest(ws []WorkOneLang) WorkOneLang {
	if len(ws) == 0 {
		panic("cannot get the latest element of an empty array")
	}
	latest := ws[0]
	for _, work := range ws {
		if work.Created().Year() == 65535 {
			continue
		}
		if work.Created().After(latest.Created()) {
			latest = work
		}
	}
	return latest
}

func log(o interface{}) string {
	spew.Dump(o)
	if !BuildingForProduction() {
		return fmt.Sprintf("logged %v to stdout", o)
	}
	return ""
}

// asset returns the full URL for a given asset (ie a website's static asset like an icon)
func asset(assetPath string) string {
	assetPath = strings.ReplaceAll(assetPath, "#", "sharp")
	var urlScheme string
	if !BuildingForProduction() {
		urlScheme = "file://" + os.Getenv("LOCAL_PROJECTS_DIR") + "portfolio/assets/%s"
	} else {
		urlScheme = "https://assets.ewen.works/%s"
	}
	return fmt.Sprintf(urlScheme, assetPath)
}

// media returns the full URL for a given media (ie a work's media URL)
func media(mediaPath string) string {
	mediaPath = strings.ReplaceAll(mediaPath, "#", "sharp")
	var urlScheme string
	if !BuildingForProduction() {
		urlScheme = "file://" + os.Getenv("LOCAL_PROJECTS_DIR") + "/portfolio/database/%s"
	} else {
		urlScheme = "https://media.ewen.works/%s"
	}
	urlScheme = strings.ReplaceAll(urlScheme, "//", "/")
	urlScheme = strings.Replace(urlScheme, "https:/", "https://", 1)
	urlScheme = strings.Replace(urlScheme, "file:/", "file://", 1)
	return fmt.Sprintf(urlScheme, mediaPath)
}

// lookupTag returns the tag referred to by name
func lookupTag(name string) Tag {
	for _, tag := range KnownTags {
		if tag.ReferredToBy(name) {
			return tag
		}
	}
	panic("cannot find tag with display name " + name + ", look at /home/ewen/projects/portfolio/hydrator/tags.go")
}

// lookupTech returns the tech referred to by name
func lookupTech(name string) Technology {
	for _, tech := range KnownTechnologies {
		if tech.ReferredToBy(name) {
			return tech
		}
	}
	panic("cannot find tech with display name " + name + ", look at /home/ewen/projects/portfolio/hydrator/technologies.go")
}

func makeWorkSlice() []WorkOneLang {
	s := make([]WorkOneLang, 0)
	return s
}

func appendWorkSlice(toAppend WorkOneLang, ws []WorkOneLang) []WorkOneLang {
	new := append(ws, toAppend)
	return new
}
