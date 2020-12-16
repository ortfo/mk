package main

import (
	"fmt"
	"time"

	"github.com/davecgh/go-spew/spew"
	jsoniter "github.com/json-iterator/go"
)

type Database struct {
	Works []Work
}

func LoadDatabase(filename string) (Database, error) {
	var works []Work
	json := jsoniter.ConfigFastest
	SetJSONNamingStrategy(LowerCaseWithUnderscores)
	content, err := ReadFile(filename)
	if err != nil {
		return Database{}, err
	}
	err = json.Unmarshal(content, &works)
	if IsVerbose() {
		spew.Dump(works)
	}
	return Database{Works: works}, nil
}

func (work *WorkOneLang) Created() time.Time {
	var creationDate string
	if work.Metadata.Created != "" {
		creationDate = work.Metadata.Created
	} else {
		creationDate = work.Metadata.Finished
	}
	parsedDate, err := ParseCreationDate(creationDate)
	if err != nil {
		fmt.Printf("Error while parsing creation date of %v:\n", work.ID)
		panic(err)
	}
	return parsedDate
}

func (work WorkOneLang) IsWIP() bool {
	return work.Metadata.WIP || (work.Metadata.Started != "" && (work.Metadata.Created != "" || work.Metadata.Finished != ""))
}

func GetOneLang(lang string, works ...Work) []WorkOneLang {
	result := make([]WorkOneLang, 0, len(works))
	for _, work := range works {
		var title string
		paragraphs := make([]Paragraph, 0)
		media := make([]Media, 0)
		links := make([]Link, 0)
		footnotes := make([]Footnote, 0)
		if len(work.Title[lang]) > 0 {
			title = work.Title[lang]
		} else {
			title = work.Title["default"]
		}
		if len(work.Paragraphs[lang]) > 0 {
			paragraphs = work.Paragraphs[lang]
		} else {
			paragraphs = work.Paragraphs["default"]
		}
		if len(work.Media[lang]) > 0 {
			media = work.Media[lang]
		} else {
			media = work.Media["default"]
		}
		if len(work.Links[lang]) > 0 {
			links = work.Links[lang]
		} else {
			links = work.Links["default"]
		}
		if len(work.Footnotes[lang]) > 0 {
			footnotes = work.Footnotes[lang]
		} else {
			footnotes = work.Footnotes["default"]
		}
		result = append(result, WorkOneLang{
			ID:         work.ID,
			Metadata:   work.Metadata,
			Title:      title,
			Paragraphs: paragraphs,
			Media:      media,
			Links:      links,
			Footnotes:  footnotes,
		})
	}
	return result
}
