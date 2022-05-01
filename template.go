package ortfomk

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/json-iterator/go"
)

//go:embed template.js
var templateJSFile string

func GenerateJSFile(hydration *Hydration, templateName string, compiledTemplate string) (string, error) {
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

	constantsToInject := map[string]interface{}{
		"all_tags":         g.Tags,
		"all_technologies": g.Technologies,
		"all_sites":        g.Sites,
		"all_works": func() []interface{} {
			works := make([]interface{}, len(g.Works))
			for i, work := range g.Works {
				works[i] = work.InLanguage(hydration.language)
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
	}

	if hydration.IsTag() {
		constantsToInject["CurrentTag"] = hydration.tag
	}
	if hydration.IsTech() {
		constantsToInject["CurrentTech"] = hydration.tech
	}
	if hydration.IsSite() {
		constantsToInject["CurrentSite"] = hydration.site
	}
	if hydration.IsWork() {
		constantsToInject["CurrentWork"] = hydration.work.InLanguage(hydration.language)
		layedout, err := hydration.work.InLanguage(hydration.language).LayedOut()
		if err != nil {
			return "", fmt.Errorf("while laying out %s: %w", hydration.Name(), err)
		}

		constantsToInject["CurrentWorkLayedOut"] = layedout
	}

	dataDeclarations := make([]string, 0)
	for name, value := range constantsToInject {
		jsoned, err := jsoniter.Config{TagKey: "notjson"}.Froze().MarshalToString(value)
		if err != nil {
			return "", fmt.Errorf("while converting %s JSON: %w", name, err)
		}
		dataDeclarations = append(dataDeclarations, fmt.Sprintf("const %s = %s;", name, jsoned))
	}
	dataObject := "{ " + strings.Join(keys(constantsToInject), ", ") + " }"

	return "/*prelude*/" + prelude + "\n/*data*/\n" + strings.Join(dataDeclarations, "\n") + "\n/*templateStatic*/\n" + templateJSFile + "\n/*pugCOmpiled*/\n" + compiledTemplate + "\n" + fmt.Sprintf(`template(%s)`, dataObject), nil
}
