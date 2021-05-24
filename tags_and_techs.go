package main

import (
	"io/ioutil"

	"github.com/metal3d/go-slugify"
	"gopkg.in/yaml.v2"
)

// Tag represents a tag
type Tag struct {
	Singular     string   `yaml:"singular"`      // Plural form display name
	Plural       string   `yaml:"plural"`        // Singular form display name
	Aliases      []string `yaml:"aliases"`       // Works with a tag name in this array will be considered as tagged by the Tag
	Description  string   `yaml:"description"`   // A description of what works that have this tag are.
	LearnMoreURL string   `yaml:"learn more at"` // A URL to a page where more about that tag can be learnt
}

// Technology represents something that a work was made with
// used for the /using/_technology path
type Technology struct {
	URLName      string   `yaml:"slug"`          // (unique) identifier used in the URL
	DisplayName  string   `yaml:"name"`          // name displayed to the user
	Aliases      []string `yaml:"aliases"`       // aliases pointing to the canonical URL (built from URLName)
	Author       string   `yaml:"by"`            // What company is behind the tech? (to display i.e. 'Adobe Photoshop' instead of 'Photoshop')
	LearnMoreURL string   `yaml:"learn more at"` // The technology's website
	Description  string   `yaml:"description"`   // A short description of the technology
}

// URLName computes the identifier to use in the tag's page's URL
func (t Tag) URLName() string {
	return slugify.Marshal(t.Plural)
}

// ReferredToBy returns whether the given name refers to the tag
func (t *Tag) ReferredToBy(name string) bool {
	return StringsLooselyMatch(name, t.Plural, t.Singular, t.URLName()) || StringsLooselyMatch(name, t.Aliases...)
}

// ReferredToBy returns whether the given name refers to the tech
func (t *Technology) ReferredToBy(name string) bool {
	return StringsLooselyMatch(name, t.URLName, t.DisplayName) || StringsLooselyMatch(name, t.Aliases...)
}

// LoadTechnologies loads the technologies from the given yaml file into a []Technology
func LoadTechnologies(filename string) (technologies []Technology, err error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &technologies)
	return
}

// LoadTags loads the tags from the given yaml file into a []Tag
func LoadTags(filename string) (tags []Tag, err error) {
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &tags)
	return
}
