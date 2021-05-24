package ortfomk

import (
	"fmt"
	"reflect"
	"strings"

	ortfodb "github.com/ortfo/db"
)

// LayoutElement represents a work layout element: paragraph, media, link or spacer.
type LayoutElement struct {
	IsParagraph bool
	IsMedia     bool
	IsLink      bool
	IsSpacer    bool
}

// LayedOutCell represents a cell of a built layout
// Is it essentially the hydrated equivalent of LayoutElement.
type LayedOutCell struct {
	// The cell's type: spacer, paragraph, media or link
	Type string
	// Convenience content type, first part of content type except
	// for application/, where application/pdf becomes pdf and (maybe) others
	GeneralContentType string
	// The three possible cells
	ortfodb.Media
	ortfodb.Paragraph
	ortfodb.Link
	// Metadata from the work
	Metadata *WorkMetadata
}

// ID Returns a layed out cell's ID, removing ambiguity
// (since a cell cannot be two things at the same time, .ID will be .Paragraph.ID for a paragraph, etc.)
func (l LayedOutCell) ID() string {
	if l.Type == "paragraph" {
		return l.Paragraph.ID
	}
	if l.Type == "media" {
		return l.Media.ID
	}
	if l.Type == "link" {
		return l.Link.ID
	}
	return ""
}

// Title Returns a layed out cell's Title, removing ambiguity
// (since a cell cannot be two things at the same time, .Title will be .Media.Title for a media, etc.)
func (l LayedOutCell) Title() string {
	if l.Type == "paragraph" {
		return ""
	}
	if l.Type == "media" {
		return l.Media.Title
	}
	if l.Type == "link" {
		return l.Link.Title
	}
	return ""
}

func (l LayedOutCell) String() string {
	return fmt.Sprintf("<%s id=%s>", l.Type, l.ID())
}

// Layout is a 2d array of layout elements (rows and columns)
type Layout = [][]LayoutElement

// LayoutElementRepr returns the strings representation of a Layout
func (layoutElement LayoutElement) String() string {
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
			repr += element.String()
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

type usedCounts struct {
	p int
	m int
	l int
}

// LayedOut returns an matrix of dimension 2 of LayedOutCells
// arranaged following the work's 'layout' metadata field
func (work WorkOneLang) LayedOut() [][]LayedOutCell {
	var layout Layout
	if len(work.Metadata.Layout) >= 1 {
		layout = loadLayout(work.Metadata.Layout)
	} else {
		layout = autoLayout(&work)
	}
	cells := make([][]LayedOutCell, 0, len(layout))
	usedCounts := usedCounts{}
	for _, layoutRow := range layout {
		row := make([]LayedOutCell, 0, len(layoutRow))
		for _, layoutElement := range layoutRow {
			var cell LayedOutCell
			if layoutElement.IsSpacer {
				cell = LayedOutCell{
					Type:     "spacer",
					Metadata: &work.Metadata,
				}
			} else if layoutElement.IsLink {
				if len(work.Links) <= usedCounts.l {
					panic(buildLayoutErrorMessage("links", &work, usedCounts.l, layout))
				}
				data := work.Links[usedCounts.l]
				usedCounts.l++
				cell = LayedOutCell{
					Type:     "link",
					Link:     data,
					Metadata: &work.Metadata,
				}
			} else if layoutElement.IsMedia {
				if len(work.Media) <= usedCounts.m {
					panic(buildLayoutErrorMessage("media", &work, usedCounts.m, layout))
				}
				data := work.Media[usedCounts.m]
				usedCounts.m++
				mediaGeneralContentType := strings.Split(data.ContentType, "/")[0]
				if data.ContentType == "application/pdf" {
					mediaGeneralContentType = "pdf"
				}
				if data.Duration <= 5 && !data.HasSound && data.Duration > 0 {
					data.Attributes = ortfodb.MediaAttributes{
						Playsinline: true,
						Looped:      true,
						Autoplay:    true,
						Muted:       true,
						Controls:    false,
					}
				}
				cell = LayedOutCell{
					Type:               "media",
					Media:              data,
					GeneralContentType: mediaGeneralContentType,
					Metadata:           &work.Metadata,
				}
			} else if layoutElement.IsParagraph {
				if len(work.Paragraphs) <= usedCounts.p {
					panic(buildLayoutErrorMessage("paragraphs", &work, usedCounts.p, layout))
				}
				data := work.Paragraphs[usedCounts.p]
				usedCounts.p++
				cell = LayedOutCell{
					Type:      "paragraph",
					Paragraph: data,
					Metadata:  &work.Metadata,
				}
			} else {
				printfln("\nWARN: While layouting %s: element %s has no Type", work.ID, layoutElement)
			}
			row = append(row, cell)
		}
		cells = append(cells, row)
	}
	return cells
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
