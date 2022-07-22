package ortfomk

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

type OutputTemplates struct {
	Media      string
	Translated string
	Rest       string
}

type Configuration struct {
	Development struct {
		OutputTo OutputTemplates `yaml:"output to"`
	}
	Production struct {
		UploadTo    OutputTemplates `yaml:"upload to"`
		AvailableAt OutputTemplates `yaml:"available at"`
	}
	AdditionalData []string `yaml:"additional data"`
}

func LoadConfiguration(path string) (Configuration, error) {
	if path == "" {
		path = "ortfomk.yaml"
		if _, err := os.Stat(path); os.IsNotExist(err) {
			LogWarning("No ortfomk.yaml found, using default configuration. A ortfomk.yaml file will be generated.")
			defaultConfig, err := yaml.Marshal(DefaultConfiguration())
			if err != nil {
				panic(err)
			}
			ioutil.WriteFile("ortfomk.yaml", []byte(defaultConfig), 0o644)
			return DefaultConfiguration(), nil
		}
	}

	config := Configuration{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Configuration{}, fmt.Errorf("while reading configuration file: %w", err)
	}

	err = yaml.Unmarshal(raw, &config)
	if err != nil {
		return Configuration{}, fmt.Errorf("while parsing configuration file: %w", err)
	}

	LogDebug("Loaded configuration: %#v", config)
	return config, nil
}

func DefaultConfiguration() Configuration {
	return Configuration{
		Development: struct {
			OutputTo OutputTemplates "yaml:\"output to\""
		}{
			OutputTo: OutputTemplates{
				Media:      "media/",
				Translated: "<language>/",
				Rest:       "/",
			},
		},
		Production: struct {
			UploadTo    OutputTemplates "yaml:\"upload to\""
			AvailableAt OutputTemplates "yaml:\"available at\""
		}{
			UploadTo:    OutputTemplates{},
			AvailableAt: OutputTemplates{},
		},
		AdditionalData: []string{},
	}
}
