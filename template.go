package ortfomk

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jaytaylor/html2text"
	jsoniter "github.com/json-iterator/go"
	ortfodb "github.com/ortfo/db"
)

//go:embed template.js
var staticTemplateFunctions string

type workOneLangFrozen struct {
	WorkOneLang

	// Frozen methods
	ColorsCSS string
	ColorsMap map[string]string
	Created   time.Time
	IsWIP     bool
	Summary   string
}

type tagFrozen struct {
	Tag

	// Frozen methods
	URLName string
}

func (t Tag) Freeze() tagFrozen {
	return tagFrozen{
		Tag:     t,
		URLName: t.URLName(),
	}
}

// layedOutElementFrozen stores a layed out element along with the result of calling
// some niladic methods such as .Title, .ID, etc. so that they can be marshalled into JSON.
// This solution is clunky, but works until https://github.com/json-iterator/go/issues/616 is fixed.
type layedOutElementFrozen struct {
	Type               string
	LayoutIndex        int
	Positions          [][]int
	GeneralContentType string
	Metadata           *WorkMetadata

	// ortfodb.Media
	Alt         string
	Source      string
	Path        string
	ContentType string
	Size        uint64 // In bytes
	Dimensions  ortfodb.ImageDimensions
	Duration    uint // In seconds
	Online      bool // Whether the media is hosted online (referred to by an URL)
	Attributes  ortfodb.MediaAttributes
	HasSound    bool // The media is either an audio file or a video file that contains an audio stream

	// ortfodb.Paragraph
	Content string

	// ortfodb.Link
	Name string
	URL  string

	// frozen methods
	Title  string
	ID     string
	CSS    string
	String string
}

func GenerateJSFile(hydration *Hydration, templateName string, compiledPugTemplate string) (string, error) {
	var assetsTemplate string
	var mediaTemplate string

	if os.Getenv("ENV") == "dev" {
		assetsTemplate = "/assets/<path>"
		mediaTemplate = "/dist/media/<path>"
	} else {
		assetsTemplate = "https://assets.ewen.works/<path>"
		mediaTemplate = "https://media.ewen.works/<path>"
	}

	prelude := fmt.Sprintf(`
		const media = path => %q.replace('<path>', path);
		const asset = path => %q.replace('<path>', path);
	`, mediaTemplate, assetsTemplate)

	dataToInject := map[string]interface{}{
		"all_tags": func() []tagFrozen {
			frozenTags := make([]tagFrozen, len(g.Tags))
			for _, tag := range g.Tags {
				frozenTags = append(frozenTags, tag.Freeze())
			}
			return frozenTags
		}(),
		"all_technologies": g.Technologies,
		"all_sites":        g.Sites,
		"all_works": func() []interface{} {
			works := make([]interface{}, len(g.Works))
			for i, work := range g.Works {
				works[i] = work.InLanguage(hydration.language).Freeze()
			}
			return works
		}(),
		"_translations": func() map[string]string {
			out := make(map[string]string)
			for _, message := range g.Translations[hydration.language].poFile.Messages {
				out[message.MsgId] = message.MsgStr
			}
			return out
		}(),
		"current_language": hydration.language,
		"current_path": func() string {
			p, err := hydration.GetDistFilepath(templateName)
			if err != nil {
				LogError("Could not get dist filepath for template %s, this shouldn't happen.", templateName)
			}
			return strings.TrimPrefix(p, "dist/")
		}(),
	}

	if hydration.IsTag() {
		dataToInject["CurrentTag"] = hydration.tag.Freeze()
	}
	if hydration.IsTech() {
		dataToInject["CurrentTech"] = hydration.tech
	}
	if hydration.IsSite() {
		dataToInject["CurrentSite"] = hydration.site
	}
	if hydration.IsWork() {
		work := hydration.work.InLanguage(hydration.language)
		dataToInject["CurrentWork"] = work.Freeze()
		layedout, err := work.LayedOut()
		if err != nil {
			return "", fmt.Errorf("while laying out %s: %w", hydration.Name(), err)
		}

		frozenLayout := make([]layedOutElementFrozen, len(layedout))
		for i, element := range layedout {
			frozenLayout[i] = layedOutElementFrozen{
				Type:               element.Type,
				LayoutIndex:        element.LayoutIndex,
				Positions:          element.Positions,
				GeneralContentType: element.GeneralContentType,
				Title:              element.Title(),
				ID:                 element.ID(),
				CSS:                element.CSS(),
				String:             element.String(),
				Alt:                element.Alt,
				Source:             element.Source,
				Path:               element.Path,
				ContentType:        element.ContentType,
				Size:               element.Size,
				Dimensions:         element.Dimensions,
				Duration:           element.Duration,
				Online:             element.Online,
				Attributes:         element.Attributes,
				HasSound:           element.HasSound,
				Content:            element.Content,
				Name:               element.Name,
				URL:                element.URL,
				Metadata:           element.Metadata,
			}
		}

		dataToInject["CurrentWorkLayedOut"] = frozenLayout
	}

	for key, value := range g.AdditionalData {
		dataToInject[key] = value
	}

	dataDeclarations := make([]string, 0)
	for name, value := range dataToInject {
		// Don't use JSON tags, use the Go struct field names
		jsoned, err := jsoniter.Config{TagKey: "notjson"}.Froze().MarshalToString(value)
		if err != nil {
			return "", fmt.Errorf("while converting %s JSON: %w", name, err)
		}
		dataDeclarations = append(dataDeclarations, fmt.Sprintf("const %s = %s;", name, jsoned))
	}
	templateCall := "template({ " + strings.Join(keys(dataToInject), ", ") + " });"

	return "/*prelude*/" + prelude + "\n/*data*/\n" + strings.Join(dataDeclarations, "\n") + "\n/*static template functions*/\n" + staticTemplateFunctions + "\n/*compiled pug template*/\n" + compiledPugTemplate + "\n/*template call*/\n" + templateCall, nil
}

func (w WorkOneLang) ColorsCSS() string {
	var cssDeclaration string
	for key, value := range w.ColorsMap() {
		cssDeclaration += fmt.Sprintf("--%s:%s;", key, value)
	}
	return cssDeclaration
}

// getColorsMap returns a mapping of "primary", "secondary", etc to the color values,
// with an added "#" prefix if needed
func (w WorkOneLang) ColorsMap() map[string]string {
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

func (w WorkOneLang) Freeze() workOneLangFrozen {
	return workOneLangFrozen{
		WorkOneLang: w,
		ColorsCSS:   w.ColorsCSS(),
		ColorsMap:   w.ColorsMap(),
		Created:     w.Created(),
		IsWIP:       w.IsWIP(),
		Summary:     w.Summary(),
	}
}

func (w WorkOneLang) Summary() string {
	if len(w.Paragraphs) == 0 {
		return ""
	}

	summary := w.Paragraphs[0].Content
	html, err := goquery.NewDocumentFromReader(strings.NewReader(summary))
	if err != nil {
		LogError("while creating plain text of first paragraph: %s", err)
		return ""
	}
	html.Find("sup.footnote-ref").Each(func(i int, s *goquery.Selection) {
		s.ReplaceWithHtml(superscriptize(s.Find("a").First().Text()))
	})
	summary, err = html.Html()
	if err != nil {
		LogError("while creating plain text of first paragraph: %s", err)
		return ""
	}
	summary, err = html2text.FromString(summary, html2text.Options{
		OmitLinks: true,
		TextOnly:  true,
	})
	if err != nil {
		LogError("while creating plain text of first paragraph: %s", err)
		return ""
	}
	return summary
}

// superscriptize replaces all number characters in given text with their unicode superscript equivalent.
func superscriptize(text string) string {
	for original, superscript := range map[string]string{
		"0": "⁰",
		"1": "¹",
		"2": "²",
		"3": "³",
		"4": "⁴",
		"5": "⁵",
		"6": "⁶",
		"7": "⁷",
		"8": "⁸",
		"9": "⁹",
	} {
		text = strings.ReplaceAll(text, original, superscript)
	}
	return text
}
