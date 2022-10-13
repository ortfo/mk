package ortfomk

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/antonmedv/expr"
	exprVM "github.com/antonmedv/expr/vm"
	"github.com/gobwas/glob"
	"gopkg.in/yaml.v3"
)

type HTMLString = string
type URLString = string
type AntonmedvExpression = string

type Collection struct {
	ID          string
	Title       map[string]string
	Description map[string]HTMLString
	LearnMoreAt URLString `yaml:"learn more at"`
	Includes    AntonmedvExpression
	Aliases     []string
	Works       []Work
}

type CollectionOneLang struct {
	Language    string
	ID          string
	Title       string
	Description HTMLString
	LearnMoreAt URLString `yaml:"learn more at"`
	Includes    AntonmedvExpression
	Aliases     []string
	Works       []WorkOneLang
}

func LoadCollections(filename string, works []Work, tags []Tag, technologies []Technology) (collections []Collection, err error) {
	Status(StepLoadCollections, ProgressDetails{File: filename})
	collectionsMap := make(map[string]Collection)
	raw, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(raw, &collectionsMap)
	for id, collection := range collectionsMap {
		collection.ID = id
		descriptionsHTML := make(map[string]HTMLString)
		for lang, description := range collection.Description {
			descriptionsHTML[lang] = MarkdownParagraphToHTML(description)
		}
		collection.Description = descriptionsHTML
		worksInCollection := make([]Work, 0)
		for _, work := range works {
			contained, err := collection.InLanguage("-").Contains(work.InLanguage("-"), works, tags, technologies)
			if err != nil {
				return collections, fmt.Errorf("while checking if %s is in %s: %w", work.ID, collection.ID, err)
			}
			if contained {
				worksInCollection = append(worksInCollection, work)
			}
		}
		collection.Works = worksInCollection
		LogDebug("filled collection %s with works %v", collection.ID, collection.Works)
		collections = append(collections, collection)
	}
	return
}

func (c *Collection) InLanguage(language string) CollectionOneLang {
	var title string
	var description HTMLString
	var works []WorkOneLang
	if translated, ok := c.Title[language]; ok {
		title = translated
	}
	if translated, ok := c.Description[language]; ok {
		description = translated
	}
	for _, w := range c.Works {
		works = append(works, w.InLanguage(language))
	}
	return CollectionOneLang{
		Language:    language,
		ID:          c.ID,
		Title:       title,
		Description: description,
		LearnMoreAt: c.LearnMoreAt,
		Includes:    c.Includes,
		Works:       works,
	}
}
func (c Collection) String() string {
	return c.ID
}

func (c CollectionOneLang) String() string {
	return c.ID
}

func (c CollectionOneLang) Contains(work WorkOneLang, works []Work, tags []Tag, technologies []Technology) (bool, error) {
	LogDebug("starting containance check: %s ∈ %s?", work.ID, c.ID)
	context := map[string]interface{}{"work": work}
	for _, w := range works {
		context[strings.ReplaceAll(w.ID, "-", "_")] = w.ID == work.ID
	}
	for _, t := range tags {
		isOnW := false
		for _, name := range work.Metadata.Tags {
			if t.ReferredToBy(name) {
				isOnW = true
				break
			}
		}
		context["tag_"+strings.ReplaceAll(t.URLName(), "-", "_")] = isOnW
	}
	for _, t := range technologies {
		isOnW := false
		for _, name := range work.Metadata.MadeWith {
			if t.ReferredToBy(name) {
				isOnW = true
				break
			}
		}
		context["technology_"+strings.ReplaceAll(t.URLName, "-", "_")] = isOnW
	}
	predicate, err := preprocessContainsPredicate(c.Includes, keys(context))
	LogDebug("work collection predicate preprocessed: %s -> %s", c.Includes, predicate)
	if err != nil {
		return false, fmt.Errorf("while pre-processing work collection predicate: %w", err)
	}

	result, err := evaluateContainsPredicate(predicate, context)
	if err != nil {
		return false, fmt.Errorf("while evaluating work collection predicate %q (from %q): %w", predicate, c.Includes, err)
	}

	return result, nil
}

func preprocessContainsPredicate(expr AntonmedvExpression, variables []string) (AntonmedvExpression, error) {
	LogDebug("preprocessing %q with variable names %v", expr, variables)
	expr = regexp.MustCompile(`(\S)-(\S)`).ReplaceAllString(expr, "${1}_${2}")
	expr = regexp.MustCompile(`(\s|^)#(\S+)(\s|$)`).ReplaceAllString(expr, "${1}tag_${2}${3}")
	expr = regexp.MustCompile(`(\s|^)made with (\S+)(\s|$)`).ReplaceAllString(expr, "${1}technology_${2}${3}")
	for _, captures := range regexp.MustCompile(`(\s|^)((?:.*\*.*)+)(\s|$)`).FindAllStringSubmatch(expr, -1) {
		globPattern, err := glob.Compile(captures[2])
		if err != nil {
			return "", fmt.Errorf("while compiling glob pattern %s: %w", captures[2], err)
		}

		matchingVariables := make([]string, 0)
		LogDebug("looking for context keys that match %s", globPattern)
		for _, variable := range variables {
			if globPattern.Match(variable) {
				matchingVariables = append(matchingVariables, variable)
			}
		}

		expr = strings.ReplaceAll(expr, captures[0], captures[1]+"("+strings.Join(matchingVariables, " or ")+")"+captures[3])
	}
	return expr, nil
}

func evaluateContainsPredicate(preprocessedExpr AntonmedvExpression, context map[string]interface{}) (bool, error) {
	var compiledExpr *exprVM.Program
	var err error
	if cached, ok := DynamicPathExpressionsCache[preprocessedExpr]; ok {
		compiledExpr = cached
	} else {
		LogDebug("compiling work collection predicate %q", preprocessedExpr)
		compiledExpr, err = expr.Compile(preprocessedExpr)
		DynamicPathExpressionsCache[preprocessedExpr] = compiledExpr
	}
	if err != nil {
		return false, fmt.Errorf("invalid work collection predicate: %w", err)
	}

	value, err := expr.Run(compiledExpr, context)
	if err != nil {
		return false, fmt.Errorf("couldn't evaluate predicate: %w", err)
	}

	switch coerced := value.(type) {
	case bool:
		return coerced, nil
	default:
		return false, fmt.Errorf("predicate does not evaluate to a boolean, but to %#v", value)
	}
}
