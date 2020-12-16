package main

import (
	"github.com/metal3d/go-slugify"
)

// Tag represents a tag
type Tag struct {
	Singular string   // Plural form display name
	Plural   string   // Singular form display name
	Aliases  []string // Works with a tag name in this array will be considered as tagged by the Tag
}

// URLName computes the identifier to use in the tag's page's URL
func (t Tag) URLName() string {
	return slugify.Marshal(t.Plural)
}

// ReferredToBy returns whether the given name refers to the tag
func (t *Tag) ReferredToBy(name string) bool {
	return StringsLooselyMatch(name, t.Plural, t.Singular, t.URLName()) || StringsLooselyMatch(name, t.Aliases...)
}

// KnownTags defines which tags are valid. Each Tag will get its correspoding page generated from _tag.pug.
var KnownTags = [...]Tag{
	{
		// TODO: deprecate
		Singular: "school",
		Plural:   "school",
	},
	{
		Singular: "science",
		Plural:   "science",
	},
	{
		Singular: "card",
		Plural:   "cards",
	},
	{
		// TODO: deprecate
		Singular: "cover art",
		Plural:   "cover arts",
	},
	{
		Singular: "game",
		Plural:   "games",
	},
	{
		Singular: "graphism",
		Plural:   "graphism",
	},
	{
		Singular: "poster",
		Plural:   "posters",
	},
	{
		Singular: "automation",
		Plural:   "automation",
	},
	{
		Singular: "web",
		Plural:   "web",
	},
	{
		Singular: "intro",
		Plural:   "intros",
	},
	{
		Singular: "music",
		Plural:   "music",
	},
	{
		Singular: "app",
		Plural:   "apps",
	},
	{
		Singular: "book",
		Plural:   "books",
	},
	{
		Singular: "api",
		Plural:   "APIs",
	},
	{
		Singular: "program",
		Plural:   "programs",
	},
	{
		Singular: "cli",
		Plural:   "CLIs",
	},
	{
		Singular: "motion design",
		Plural:   "motion design",
	},
	{
		Singular: "compositing",
		Plural:   "compositing",
	},
	{
		Singular: "visual identity",
		Plural:   "visual identities",
		Aliases:  []string{"logo", "logos", "banner", "banners"},
	},
	{
		Singular: "illustration",
		Plural:   "illustrations",
	},
	{
		Singular: "typography",
		Plural:   "typography",
	},
	{
		Singular: "drawing",
		Plural:   "drawings",
	},
	{
		Singular: "icons",
		Plural:   "icons",
	},
	{
		Singular: "site",
		Plural:   "sites",
	},
	{
		Singular: "language",
		Plural:   "languages",
	},
	{
		Singular: "math",
		Plural:   "math",
		Aliases:  []string{"maths", "mathematics"},
	},
}
