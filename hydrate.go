package ortfomk

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	chromaQuick "github.com/alecthomas/chroma/quick"
	"github.com/joho/godotenv"
	"github.com/yosssi/gohtml"
	"golang.org/x/net/html"
	v8 "rogchap.com/v8go"
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

func (h *Hydration) IsSite() bool {
	return h.site.URL != ""
}

// Name returns the identifier of the object in the hydration,
// and defaults to the empty string if the current hydration is empty
func (h *Hydration) Name() string {
	if h.IsWork() {
		return h.work.ID + "@" + h.language
	}
	if h.IsTag() {
		return h.tag.URLName() + "@" + h.language
	}
	if h.IsTech() {
		return h.tech.URLName + "@" + h.language
	}
	return h.language
}

// BuildingForProduction returns true if the environment file declares ENVIRONMENT to not "dev"
func BuildingForProduction() bool {
	err := godotenv.Load(".env")
	if err != nil {
		panic("Could not load the .env file")
	}
	return os.Getenv("ENVIRONMENT") != "dev"
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
		message += fmt.Sprintf("%d %s\n", lineIndexOffset+i+1, line)
	}
	LogError(message)
}

// CompileTemplate compiles a pug template using the CLI tool pug.
func CompileTemplate(templateName string, templateContent []byte) ([]byte, error) {
	command := exec.Command("pug", "--client", "--path", templateName)
	LogDebug("compiling template: running %s", command)
	command.Stdin = bytes.NewReader(templateContent)
	command.Stderr = os.Stderr

	return command.Output()
}

// RunTemplate parses a given (HTML) template.
func RunTemplate(hydration *Hydration, templateName string, compiledTemplate []byte) (string, error) {
	compiledJSFile, err := GenerateJSFile(hydration, templateName, string(compiledTemplate))
	if os.Getenv("DEBUG") == "1" {
		os.WriteFile(templateName+"."+hydration.Name()+".js", []byte(compiledJSFile), 0644)
	}
	if err != nil {
		return "", fmt.Errorf("while generating template: %w", err)
	}

	LogDebug("executing template")
	ctx := v8.NewContext()
	jsValue, err := ctx.RunScript(compiledJSFile, templateName+".js")
	LogDebug("finished executing")
	if err, ok := err.(*v8.JSError); ok {
		line, column := lineAndColumn(err)
		codeSpinnet := codeSpinnetAround(compiledJSFile, line, column)
		return "", fmt.Errorf("while running template: %w\n%s\nStack trace:\n%s", err, codeSpinnet, err.StackTrace)
	}
	return jsValue.String(), nil
}

func lineAndColumn(err *v8.JSError) (line uint64, column uint64) {
	parts := strings.Split(err.Location, ":")
	for i, part := range parts {
		var parseErr error
		if i == len(parts)-1 {
			line, parseErr = strconv.ParseUint(part, 10, 64)
		} else if i == len(parts)-2 {
			column, parseErr = strconv.ParseUint(part, 10, 64)
		}
		if parseErr != nil {
			panic(fmt.Sprintf("while parsing positive integer %s: %s", part, parseErr))
		}
	}
	return
}

// TranslateHydrated translates an hydrated HTML page, removing i18n tags and attributes
// and replacing translatable content with their translations
func (t TranslationsOneLang) TranslateHydrated(content string) string {
	parsedContent, err := html.Parse(strings.NewReader(content))
	if err != nil {
		LogError("An error occured while parsing the hydrated HTML for translation: %s", err)
		return ""
	}
	return t.Translate(parsedContent)
}

// NameOfTemplate returns the name given to a template that is applied to multiple objects, e.g. :work.pug<portfolio>.
// Falls back to template.Name() if hydration is empty
func NameOfTemplate(name string, hydration Hydration) string {
	if hydration.Name() != "" {
		return fmt.Sprintf("%s<%s>", name, hydration.Name())
	}
	return name
}

// WriteDistFile writes the given content to the dist/ equivalent of the given fileName and returns that equivalent's path
func (h *Hydration) WriteDistFile(fileName string, content string, language string) string {
	distFilePath, err := h.GetDistFilepath(fileName)
	if err != nil {
		LogError("Invalid path: %s", err)
		return ""
	}
	distFile, err := os.Create(distFilePath)
	if err != nil {
		LogError("Could not create the destination file %s: %s ", distFilePath, err)
		return ""
	}
	defer distFile.Close()
	_, err = distFile.WriteString(content)
	if err != nil {
		LogError("Could not write to the destination file %s: %s", distFilePath, err)
		return ""
	}
	fmt.Printf("\r\033[KWritten %s", distFilePath)
	return distFilePath
}
