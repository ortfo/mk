package ortfomk

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/json-iterator/go"
)

//go:embed template.js
var staticTemplateFunctions string

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
		dataToInject["CurrentTag"] = hydration.tag
	}
	if hydration.IsTech() {
		dataToInject["CurrentTech"] = hydration.tech
	}
	if hydration.IsSite() {
		dataToInject["CurrentSite"] = hydration.site
	}
	if hydration.IsWork() {
		dataToInject["CurrentWork"] = hydration.work.InLanguage(hydration.language)
		layedout, err := hydration.work.InLanguage(hydration.language).LayedOut()
		if err != nil {
			return "", fmt.Errorf("while laying out %s: %w", hydration.Name(), err)
		}

		dataToInject["CurrentWorkLayedOut"] = layedout
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
