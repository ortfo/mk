package main

import (
	"fmt"
	"reflect"
	"strings"
)

// LayoutElement represents a work layout element: paragraph, media, link or spacer.
type LayoutElement struct {
	IsParagraph bool
	IsMedia     bool
	IsLink      bool
	IsSpacer    bool
}

type usedCounts struct {
	p int
	m int
	l int
}

// Layout is a 2d array of layout elements (rows and columns)
type Layout = [][]LayoutElement

// LayoutElementRepr returns the strings representation of a Layout
func layoutElementRepr(layoutElement LayoutElement) string {
	if layoutElement.IsLink {
		return "l"
	}
	if layoutElement.IsMedia {
		return "m"
	}
	if layoutElement.IsParagraph {
		return "p"
	}
	if layoutElement.IsSpacer {
		return " "
	}
	panic("unexpected layoutElement: is neither link nor media nor parargraph nor spacer")
}

func layoutRepr(layout Layout) string {
	repr := ""
	for _, row := range layout {
		for _, element := range row {
			repr += layoutElementRepr(element)
		}
		repr += "\n"
	}
	return repr
}

func buildLayoutErrorMessage(whatsMissing string, work *WorkOneLang, usedCount int, layout Layout) string {
	return fmt.Sprintf(`not enough %s to satisfy the given layout:

	· Layout is:
	%v

	· work has only %d %s
	`, whatsMissing, layoutRepr(layout), usedCount, whatsMissing)
}

// BuildLayout builds an pug layout filled with content, ready to inject in a .pug file.
func (work *WorkOneLang) BuildLayout() string {
	var layout Layout
	if len(work.Metadata.Layout) >= 1 {
		layout = loadLayout(work.Metadata.Layout)
	} else {
		layout = autoLayout(work)
	}
	usedCounts := usedCounts{}
	var built string
	for _, layoutRow := range layout {
		var row string
		for _, layoutElement := range layoutRow {
			var element string
			if layoutElement.IsSpacer {
				element = `div.spacer`
			} else if layoutElement.IsLink {
				if len(work.Links) <= usedCounts.l {
					panic(buildLayoutErrorMessage("links", work, usedCounts.l, layout))
				}
				data := work.Links[usedCounts.l]
				usedCounts.l++
				element = fmt.Sprintf(`a(href="%v" id="%v" title="%v") %v`, data.URL, data.ID, data.Title, data.Name)
			} else if layoutElement.IsMedia {
				if len(work.Media) <= usedCounts.m {
					panic(buildLayoutErrorMessage("media", work, usedCounts.m, layout))
				}
				data := work.Media[usedCounts.m]
				usedCounts.m++
				mediaGeneralContentType := strings.Split(data.ContentType, "/")[0]
				if data.ContentType == "application/pdf" {
					mediaGeneralContentType = "pdf"
				}
				if data.Duration <= 5 && !data.HasAudio && data.Duration > 0 {
					data.Attributes = MediaAttributes{
						Playsinline: true,
						Loop:        true,
						Autoplay:    true,
						Muted:       true,
						Controls:    false,
					}
				}
				switch mediaGeneralContentType {
				case "video":
					element = fmt.Sprintf(
						`<video src="%v" id="%v" title="%v" %v>%v</video>`,
						media(data.Source),
						data.ID,
						data.Title,
						data.Attributes.String(),
						data.Alt,
					)
				case "audio":
					element = fmt.Sprintf(
						`<audio src="%v" id="%v" title="%v" %v>%v</audio>`,
						media(data.Source),
						data.ID,
						data.Title,
						data.Attributes.String(),
						data.Alt,
					)
				case "image":
					element = fmt.Sprintf(
						`<img src="%v" id="%v" title="%v" alt="%v" />`,
						media(data.Source),
						data.ID,
						data.Title,
						data.Alt,
					)
				case "pdf":
					element = fmt.Sprintf(
						`<div class"pdf-frame-container"><iframe class="pdf-frame" src="%v" id="%v" title="%v" width="100%%" height="100%%">%v</iframe></div>`,
						media(data.Source),
						data.ID,
						data.Title,
						data.Alt,
					)
				default:
					element = fmt.Sprintf(
						`<a href="%v" id="%v" title="%v">%v</a>`,
						media(data.Source),
						data.ID,
						data.Title,
						data.Alt,
					)
				}
				if data.Title != "" {
					element += fmt.Sprintf("<figcaption>%s</figcaption>", data.Title)
				}
				element = fmt.Sprintf(`<figure data-enable-media-closeup="%s">%s</figure>`, data.ContentType, element)
			} else if layoutElement.IsParagraph {
				if len(work.Paragraphs) <= usedCounts.p {
					panic(buildLayoutErrorMessage("paragraphs", work, usedCounts.p, layout))
				}
				data := work.Paragraphs[usedCounts.p]
				usedCounts.p++
				// element = fmt.Sprintf(`<p id="%v">%v</p>`, data.ID, data.Content)
				element = "p(id=\"" + data.ID + "\")." + IndentWithTabsNewline(2, data.Content)
			}
			row += "\t" + element + "\n"
		}
		built += fmt.Sprintf("div.row(data-columns=%d)\n%s\n", len(layoutRow), row)
	}
	return built
}

func autoLayout(work *WorkOneLang) Layout {
	layout := make(Layout, 0)
	for range work.Paragraphs {
		layout = append(layout, []LayoutElement{{IsParagraph: true}})
	}
	for range work.Media {
		layout = append(layout, []LayoutElement{{IsMedia: true}})
	}
	for range work.Links {
		layout = append(layout, []LayoutElement{{IsLink: true}})
	}
	return layout
}

func loadLayout(layout []interface{}) Layout {
	loaded := make([][]LayoutElement, 0)
	for _, layoutRowMaybeSlice := range layout {
		loadedRow := make([]LayoutElement, 0)
		var layoutRow []interface{}
		if reflect.TypeOf(layoutRowMaybeSlice).Kind() != reflect.Slice {
			layoutRow = []interface{}{layoutRowMaybeSlice}
		} else {
			layoutRow = layoutRowMaybeSlice.([]interface{})
		}
		for _, layoutElement := range layoutRow {
			loadedRow = append(loadedRow, loadLayoutElement(layoutElement))
		}
		loaded = append(loaded, loadedRow)
	}
	return loaded
}

func loadLayoutElement(layoutElement interface{}) LayoutElement {
	return LayoutElement{
		IsParagraph: layoutElement == "p",
		IsMedia:     layoutElement == "m",
		IsLink:      layoutElement == "l",
		IsSpacer:    layoutElement == nil,
	}
}
