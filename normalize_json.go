package ortfomk

import (
	"strings"
	"unicode"

	jsoniter "github.com/json-iterator/go"
)

// SetJSONNamingStrategy rename struct fields uniformly
func SetJSONNamingStrategy(decoder jsoniter.API, translate func(string) string) {
	decoder.RegisterExtension(&namingStrategyExtension{jsoniter.DummyExtension{}, translate})
}

type namingStrategyExtension struct {
	jsoniter.DummyExtension
	translate func(string) string
}

func (extension *namingStrategyExtension) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	for _, binding := range structDescriptor.Fields {
		if unicode.IsLower(rune(binding.Field.Name()[0])) || binding.Field.Name()[0] == '_' {
			continue
		}
		tag, hastag := binding.Field.Tag().Lookup("json")
		if hastag {
			tagParts := strings.Split(tag, ",")
			if tagParts[0] == "-" {
				continue // hidden field
			}
			if tagParts[0] != "" {
				continue // field explicitly named
			}
		}
		binding.ToNames = []string{extension.translate(binding.Field.Name())}
		binding.FromNames = []string{extension.translate(binding.Field.Name())}
	}
}

// LowerCaseWithUnderscores one strategy to SetNamingStrategy for. It will change HelloWorld to hello_world.
func LowerCaseWithUnderscores(name string) string {
	// Handle acronyms
	if isAllUpper(name) {
		return strings.ToLower(name)
	}
	newName := []rune{}
	for i, c := range name {
		if i == 0 {
			newName = append(newName, unicode.ToLower(c))
		} else {
			if c == ' ' {
				newName = append(newName, '_')
			} else if unicode.IsUpper(c) {
				newName = append(newName, '_')
				newName = append(newName, unicode.ToLower(c))
			} else {
				newName = append(newName, c)
			}
		}
	}
	return string(newName)
}

func isAllUpper(s string) bool {
	for _, c := range s {
		if !unicode.IsUpper(c) {
			return false
		}
	}
	return true
}
