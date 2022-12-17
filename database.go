package ortfomk

import (
	"errors"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	db "github.com/ortfo/db"
	ortfodb "github.com/ortfo/db"
)

// WorkOneLang represents a work in a single language: language-dependent items
// have been replaced with their corresponding values in a language, there is no "language" map anymore.
type WorkOneLang struct {
	ID       string
	Metadata db.WorkMetadata
	Language string
	db.LocalizedWorkContent
}

// Database holds works & other metadata
type Database struct {
	Works        []Work
	Technologies []Technology
	Tags         []Tag
	Sites        []ExternalSite
	Collections  []Collection
}

type Work db.AnalyzedWork

// String returns a string representation of the work.
// This is used to construct output paths (and therefore future URLs).
func (work Work) String() string {
	return work.ID
}

// LoadWorks reads the database file at filename into a []Work
func LoadWorks(filename string) (works []Work, err error) {
	Status(StepLoadWorks, ProgressDetails{
		File: filename,
	})
	json := jsoniter.ConfigFastest
	SetJSONNamingStrategy(json, LowerCaseWithUnderscores)
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, &works)
	// Resolve shortcut "created" for finished + started.
	for i, work := range works {
		if created, ok := work.Metadata.AdditionalMetadata["created"].(string); ok && work.Metadata.Finished == "" && work.Metadata.Started == "" && created != "" {
			works[i].Metadata.Finished = created
			works[i].Metadata.Started = created
		}
		for _, lang := range keys(work.Content) {
			// Set arrays and objects to empty instead of leaving them nil.
			if work.Content[lang].Layout == nil {
				works[i].Metadata.AdditionalMetadata["layout"] = make([]interface{}, 0)
			}
		}
		if work.Metadata.Tags == nil {
			works[i].Metadata.Tags = []string{}
		}
		if work.Metadata.MadeWith == nil {
			works[i].Metadata.MadeWith = []string{}
		}
	}
	return
}

// LoadDatabase loads works, technologies and tags into a Database
// Standard filepaths relative to databaseDir are assumed:
// - database.json for the works
// - tags.yaml for the tags
// - technologies.yaml for the technologies
func LoadDatabase(databaseDir string) (Database, error) {
	works, err := LoadWorks(path.Join(databaseDir, "database.json"))
	if err != nil {
		return Database{}, errors.New("While loading database.json: " + err.Error())
	}
	tags, err := LoadTags(path.Join(databaseDir, "tags.yaml"))
	if err != nil {
		return Database{}, errors.New("While loading tags.yaml: " + err.Error())
	}
	techs, err := LoadTechnologies(path.Join(databaseDir, "technologies.yaml"))
	if err != nil {
		return Database{}, errors.New("While loading technologies.yaml: " + err.Error())
	}
	sites, err := LoadExternalSites(path.Join(databaseDir, "sites.yaml"))
	if err != nil {
		return Database{}, errors.New("While loading sites.yaml: " + err.Error())
	}
	collections, err := LoadCollections(path.Join(databaseDir, "collections.yaml"), works, tags, techs)
	if err != nil {
		return Database{}, errors.New("While loading collections.yaml: " + err.Error())
	}
	return Database{
		Works:        works,
		Tags:         tags,
		Technologies: techs,
		Sites:        sites,
		Collections:  collections,
	}, nil
}

// Created returns the creation date of a work
func (work *WorkOneLang) Created() time.Time {
	var creationDate string
	if work.Metadata.Finished != "" {
		creationDate = work.Metadata.Finished
	} else {
		creationDate = work.Metadata.Started
	}
	if creationDate == "" {
		return time.Date(9999, time.January, 1, 0, 0, 0, 0, time.Local)
	}
	parsedDate, err := ParseCreationDate(creationDate)
	if err != nil {
		LogError("Error while parsing creation date of %v:", work.ID)
		panic(err)
	}
	return parsedDate
}

// IsWIP returns true if the work is a work in progress or has no starting date nor creation or finish date
func (work WorkOneLang) IsWIP() bool {
	return work.Metadata.WIP || (work.Metadata.Started != "" && work.Metadata.Finished == "")
}

// InLanguage returns a Work object with data from only the selected language (or the default if not found)
func (work Work) InLanguage(lang string) WorkOneLang {
	result := WorkOneLang{
		ID:       work.ID,
		Metadata: work.Metadata,
	}
	if localized, ok := work.Content[lang]; ok {
		result.Language = lang
		result.LocalizedWorkContent = localized
	} else {
		result.Language = "default"
		result.LocalizedWorkContent = work.Content["default"]
	}
	return result
}

// GetOneLang returns an array of works with .InLanguage applied to each
func GetOneLang(lang string, works ...Work) []WorkOneLang {
	result := make([]WorkOneLang, 0, len(works))
	for _, work := range works {
		result = append(result, work.InLanguage(lang))
	}
	return result
}

func GeneralContentType(media db.Media) string {
	if media.ContentType == "application/pdf" {
		return "pdf"
	}
	return strings.Split(media.ContentType, "/")[0]

}

// MarkdownParagraphToHTML returns the HTML equivalent of the given markdown string, without the outer <p>…</p> tag
func MarkdownParagraphToHTML(markdown string) string {
	return regexp.MustCompile(`^(?m)<p>(.+)</p>$`).ReplaceAllString(ortfodb.MarkdownToHTML(markdown), "${1}")
}

// PublicWorks returns Works that are not private
func (g *GlobalData) PublicWorks() (works []Work) {
	for _, w := range g.Works {
		if !w.Metadata.Private {
			works = append(works, w)
		}
	}
	return
}
