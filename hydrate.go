package ortfomk

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/Joker/jade"
	"github.com/Masterminds/sprig"
	chromaQuick "github.com/alecthomas/chroma/quick"
	"github.com/joho/godotenv"
	"github.com/yosssi/gohtml"
	"golang.org/x/net/html"
)

// Hydration represents a Tag, Technology or Work
type Hydration struct {
	language string
	tag      Tag
	tech     Technology
	work     Work
	site     ExternalSite
}

// IsWork returns true if the current hydration contains a Work
func (h *Hydration) IsWork() bool {
	return h.work.ID != ""
}

// IsTag returns true if the current hydration contains a Tag
func (h *Hydration) IsTag() bool {
	return h.tag.URLName() != ""
}

// IsTech returns true if the current hydration contains a Tech
func (h *Hydration) IsTech() bool {
	return h.tech.URLName != ""
}

// Name returns the identifier of the object in the hydration,
// and defaults to the empty string if the current hydration is empty
func (h *Hydration) Name() string {
	if h.IsWork() {
		return h.work.ID
	}
	if h.IsTag() {
		return h.tag.URLName()
	}
	if h.IsTech() {
		return h.tech.URLName
	}
	return ""
}

// ConvertTemplateIfNeeded checks if the given filename ends with .pug, and converts the template to HTML
// Otherwise, it leaves it as it is and simply reads it.
func ConvertTemplateIfNeeded(filename string) (content string, err error) {
	if strings.HasSuffix(filename, ".pug") {
		return ConvertTemplate(filename)
	}
	contentBytes, err := ioutil.ReadFile(filename)
	content = string(contentBytes)
	return
}

// BuildingForProduction returns true if the environment file declares ENVIRONMENT to not "dev"
func BuildingForProduction() bool {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Could not load the .env file")
	}
	return os.Getenv("ENVIRONMENT") != "dev"
}

// ConvertTemplate converts a .pug template to an HTML one
func ConvertTemplate(absFilepath string) (string, error) {
	raw, err := ioutil.ReadFile(absFilepath)
	if err != nil {
		return "", err
	}

	// raw = FixExtendsIncludeStatements(raw, absFilepath)

	template, err := jade.Parse(absFilepath, raw)
	template = strings.ReplaceAll(template, "&#34;", "\"")

	if err != nil {
		PrintTemplateErrorMessage("converting template to HTML", absFilepath, string(raw), err, "pug")
		return "", fmt.Errorf("error while converting to HTML")
	}

	return template, nil
}

// FixExtendsIncludeStatements fixes `extends` statement (and `include` statements).
// From joker/jade's point of view, the current work dir is just the project's root,
// thus (project root)/layout.pug does not exist.
// Fix that by adding src/ in front.
// Joker/jade also requires the .pug extension, add it if it's missing.
// filepath needs to be absolute.
func FixExtendsIncludeStatements(raw []byte, filepath string) []byte {
	extendsPattern := regexp.MustCompile(`(?m)^(extends|include) (.+)$`)
	return extendsPattern.ReplaceAllFunc(raw, func(line []byte) []byte {
		// printfln("transforming %s", line)
		keyword, argument := strings.SplitN(string(line), " ", 2)[0], strings.SplitN(string(line), " ", 2)[1]
		if strings.HasPrefix(argument, "src/") {
			return line
		}
		return []byte(fmt.Sprintf("%s %s/%s.pug", keyword, path.Clean(path.Dir(filepath)), argument))
	})
}

// PrintTemplateErrorMessage prints a nice error message with a preview of the code where the error occured
func PrintTemplateErrorMessage(whileDoing string, templateName string, templateContent string, err error, templateLanguage string) {
	// TODO when error occurs in a subtemplate, show code snippet from the innermost subtemplate instead of the outermost
	lineIndexPattern := regexp.MustCompile(`:(\d+)`)
	listIndices := lineIndexPattern.FindStringSubmatch(err.Error())
	if listIndices == nil {
		LogError("While %s %s: %s", whileDoing, templateName, err.Error())
		return
	}
	lineIndex64, _ := strconv.ParseInt(listIndices[1], 10, 64)
	lineIndex := int(lineIndex64)
	message := fmt.Sprintf("While %s %s:%d: %s\n", whileDoing, templateName, lineIndex, strings.SplitN(err.Error(), listIndices[1], 2)[1])
	lineIndex-- // Lines start at 1, arrays of line are indexed from 0
	highlightedWriter := bytes.NewBufferString("")
	chromaQuick.Highlight(highlightedWriter, gohtml.Format(templateContent), templateLanguage, "terminal16m", "pygments")
	lines := strings.Split(highlightedWriter.String(), "\n")
	var lineIndexOffset int
	if len(lines) >= lineIndex+5+1 {
		if lineIndex >= 5 {
			lines = lines[lineIndex-5 : lineIndex+5]
			lineIndexOffset = lineIndex - 5
		} else {
			lines = lines[0 : lineIndex+5]
		}
	}
	for i, line := range lines {
		if i+lineIndexOffset == lineIndex {
			message += "→ "
		} else {
			message += "  "
		}
		message += fmt.Sprintf("%d %s\n", i+1, line)
	}
	LogError(message)
}

// ParseTemplate parses a given (HTML) template.
func ParseTemplate(language string, templateName string, templateContent string) (*template.Template, error) {
	tmpl := template.New(templateName)
	tmpl = tmpl.Funcs(sprig.TxtFuncMap())
	tmpl = tmpl.Funcs(g.Translations.GetTemplateFuncMap(language))
	tmpl, err := tmpl.Parse(gohtml.Format(templateContent))

	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

// ExecuteTemplate executes a parsed HTML template to hydrate it with data, potentially with a tag, tech or work.
func ExecuteTemplate(tmpl *template.Template, language string, currentlyHydrated Hydration) (string, error) {
	// Inject Funcs now, since they depend on language
	tmpl = tmpl.Funcs(g.Translations.GetTemplateFuncMap(language))

	var buf bytes.Buffer

	err := tmpl.Execute(&buf, TemplateData{
		KnownTags:         g.Tags,
		KnownTechnologies: g.Technologies,
		KnownSites:        g.Sites,
		Works:             GetOneLang(language, g.Works...),
		CurrentTag:        currentlyHydrated.tag,
		CurrentTech:       currentlyHydrated.tech,
		CurrentWork:       currentlyHydrated.work.InLanguage(language),
		CurrentSite:       currentlyHydrated.site,
	})

	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// TranslateHydrated translates an hydrated HTML page, removing i18n tags and attributes
// and replacing translatable content with their translations
func (t *Translations) TranslateHydrated(content string, language string) string {
	parsedContent, err := html.Parse(strings.NewReader(content))
	if err != nil {
		LogError("An error occured while parsing the hydrated HTML for translation: %s", err)
		return ""
	}
	return t.TranslateToLanguage(language == "fr", parsedContent)
}

// NameOfTemplate returns the name given to a template that is applied to multiple objects, e.g. :work.pug<portfolio>.
// Falls back to template.Name() if hydration is empty
func NameOfTemplate(tmpl *template.Template, hydration Hydration) string {
	if hydration.Name() != "" {
		return fmt.Sprintf("%s<%s>", tmpl.Name(), hydration.Name())
	}
	return tmpl.Name()
}

// WriteDistFile writes the given content to the dist/ equivalent of the given fileName and returns that equivalent's path
func (h *Hydration) WriteDistFile(fileName string, content string, language string) string {
	distFilePath := h.GetDistFilepath(fileName)
	distFile, err := os.Create(distFilePath)
	if err != nil {
		LogError("Could not create the destination file "+distFilePath, err)
		return ""
	}
	_, err = distFile.WriteString(content)
	if err != nil {
		LogError("Could not write to the destination file "+distFilePath, err)
		return ""
	}
	fmt.Printf("\r\033[KWritten %s", distFilePath)
	return distFilePath
}
