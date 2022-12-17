package ortfomk

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/browser"
)

type devserver struct {
	language string
}

func (s devserver) Open(name string) (http.File, error) {
	LogDebug("handling %s", name)
	// What path to choose ? Is the requested file translated, media or rest?
	// Test them one by one, moving to the next one if not found.

	filename, err := existsOptionalHTMLExtension(filepath.Join(
		g.OutputDirectory,
		strings.ReplaceAll(
			g.Configuration.Development.OutputTo.Translated,
			"<language>",
			s.language,
		),
		name,
	))
	LogDebug("testing(translated) %q", filename)
	if err != nil {
		return nil, fmt.Errorf("while testing for a translated page: %w", err)
	}

	if filename != "" {
		return os.Open(filename)
	} else {
		LogDebug("%s |-> %q not found for translated", name, filename)
	}

	filename, err = existsOptionalHTMLExtension(filepath.Join(g.OutputDirectory, g.Configuration.Development.OutputTo.Media, name))
	LogDebug("testing(media) %q", filename)
	if err != nil {
		return nil, fmt.Errorf("while testing for a media page: %w", err)
	}

	if filename != "" {
		return os.Open(filename)
	} else {
		LogDebug("%s |-> %q not found for media", name, filename)
	}

	filename, err = existsOptionalHTMLExtension(filepath.Join(g.OutputDirectory, g.Configuration.Development.OutputTo.Rest, name))
	LogDebug("testing(rest) %q", filename)
	if err != nil {
		return nil, fmt.Errorf("while testing for a rest page: %w", err)
	}

	return os.Open(filename)
}

// returns path if exists and "" if not.
func existsOptionalHTMLExtension(filename string) (string, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		if !strings.HasSuffix(filename, ".html") {
			return existsOptionalHTMLExtension(filename + ".html")
		} else {
			return "", nil
		}
	}
	if err == nil {
		return filename, nil
	}
	return "", err
}

func StartDevServer(host string, port int, language string) {
	g.DevserverPort = port
	address := fmt.Sprintf("http://%s:%d", host, port)
	LogInfo("Starting development server on %s", address)
	browser.OpenURL(address)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), http.FileServer(devserver{language: language}))
	if err != nil {
		LogError("while starting development server: %s", err)
	}
}
