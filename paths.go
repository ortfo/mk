package ortfomk

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
)

// GetDistFilepath replaces :placeholders in srcFilepath and replaces src/ with dist/
func (h *Hydration) GetDistFilepath(srcFilepath string) string {
	// Turn into a dist/ path
	outPath := "dist/" + GetPathRelativeToSrcDir(srcFilepath)
	// Replace stuff
	if h.work.ID != "" {
		outPath = strings.ReplaceAll(outPath, ":work.created", h.work.Metadata.Created)
		outPath = strings.ReplaceAll(outPath, ":work.started", h.work.Metadata.Started)
		outPath = strings.ReplaceAll(outPath, ":work.finished", h.work.Metadata.Finished)
		outPath = strings.ReplaceAll(outPath, ":work", h.work.ID)
		outPath = regexp.MustCompile(`:if\(([\w.]+)\)\(([^)]+)\)\(([^)]+)\)`).ReplaceAllString(outPath, "{{ if .$1 }}$2{{ else }}$3{{ end }}")
		var outPathBuf bytes.Buffer
		template.Must(template.New(outPath).Parse(outPath)).Execute(&outPathBuf, h)
		outPath = outPathBuf.String()
	}
	if h.tag.URLName() != "" {
		outPath = strings.ReplaceAll(outPath, ":tag", h.tag.URLName())
	}
	if h.tech.URLName != "" {
		outPath = strings.ReplaceAll(outPath, ":technology", h.tech.URLName)
	}
	outPath = strings.ReplaceAll(outPath, ":language", h.language)
	if strings.HasSuffix(outPath, ".pug") {
		outPath = strings.TrimSuffix(outPath, ".pug") + ".html"
	}
	return outPath
}
