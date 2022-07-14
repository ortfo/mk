package ortfomk

import (
	"io/ioutil"

	"github.com/metal3d/go-slugify"
	"gopkg.in/yaml.v2"
)

// ExternalSite represents an external site (e.g. social media or email address)
type ExternalSite struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Purpose  string `yaml:"purpose"`
	Username string `yaml:"username"`
}

// String returns the string representation of the external site.
// Should be the one used in URLs, as GetDistFilepath uses this.
func (s ExternalSite) String() string {
	return s.Name
}

// Tag represents a tag
type Tag struct {
	Singular     string   `yaml:"singular"`      // Plural form display name
	Plural       string   `yaml:"plural"`        // Singular form display name
	Aliases      []string `yaml:"aliases"`       // Works with a tag name in this array will be considered as tagged by the Tag
	Description  string   `yaml:"description"`   // A description of what works that have this tag are.
	LearnMoreURL string   `yaml:"learn more at"` // A URL to a page where more about that tag can be learnt
}

// String returns the string representation of the tag.
// Should be the one used in URLs, as GetDistFilepath uses this.
func (t Tag) String() string {
	return t.URLName()
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

// String returns the string representation of the technology.
// Should be the one used in URLs, as GetDistFilepath uses this.
func (t Technology) String() string {
	return t.URLName
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
	Status(StepLoadTechnologies, ProgressDetails{File: filename})
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &technologies)
	for idx, tech := range technologies {
		technologies[idx].Description = MarkdownParagraphToHTML(tech.Description)
	}
	return
}

// LoadExternalSites loads the sites from the given yaml file into a []Site
func LoadExternalSites(filename string) (sites []ExternalSite, err error) {
	Status(StepLoadExternalSites, ProgressDetails{File: filename})
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &sites)
	for idx, site := range sites {
		sites[idx].Purpose = MarkdownParagraphToHTML(site.Purpose)
	}
	return
}

// LoadTags loads the tags from the given yaml file into a []Tag
func LoadTags(filename string) (tags []Tag, err error) {
	Status(StepLoadTags, ProgressDetails{File: filename})
	raw, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &tags)
	for idx, tag := range tags {
		tags[idx].Description = MarkdownParagraphToHTML(tag.Description)
	}
	return
}
