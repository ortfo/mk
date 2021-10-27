package ortfomk

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"

	"github.com/davecgh/go-spew/spew"
	ortfodb "github.com/ortfo/db"
)

// Cell represents a cell of a layout
type Cell struct {
	Type  string
	Index int
}

type LayedOutElement struct {
	// Either media, paragraph, link or spacer.
	Type string
	// Index in the layout specification (e.g. 3rd paragraph)
	LayoutIndex int
	// The positions on the grid. List of [row, cell] pairs.
	Positions [][]int
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

type Layout []LayedOutElement

// CSS returns CSS statements to declare the position of that element in the content grid.
func (l LayedOutElement) CSS() string {
	startingRow, endingRow, startingCol, endingCol := l.PositionBounds()
	return fmt.Sprintf(`grid-row: %d / %d; grid-column: %d / %d;`, startingRow+1, endingRow+2, startingCol+1, endingCol+2)
}

// Returns (starting row, ending row, starting column, ending column).
func (l LayedOutElement) PositionBounds() (int, int, int, int) {
	startingColumn := MaxInt
	startingRow := MaxInt
	endingColumn := 0
	endingRow := 0

	// printfln("computing grid position for %s", l)
	for _, row := range l.Positions {
		if len(row) != 2 {
			panic(fmt.Sprintf("A GridArea has an Indices array %#v with a row containing %d != 2 elements", l.Positions, len(row)))
		}
		if row[1] < startingColumn {
			startingColumn = row[1]
		}
		if row[0] < startingRow {
			startingRow = row[0]
		}
		if row[1] > endingColumn {
			endingColumn = row[1]
		}
		if row[0] > endingRow {
			endingRow = row[0]
		}
	}
	return startingRow, endingRow, startingColumn, endingColumn
}

func (c Cell) String() string {
	return fmt.Sprintf("%s[%v]", c.Type, c.Index)
}

// ID Returns a layed out cell's ID, removing ambiguity
// (since a cell cannot be two things at the same time, .ID will be .Paragraph.ID for a paragraph, etc.)
func (l LayedOutElement) ID() string {
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
func (l LayedOutElement) Title() string {
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

func (l LayedOutElement) String() string {
	return fmt.Sprintf("<%s %s @ %v>", l.Type, l.ID(), l.Positions)
}

func buildLayoutErrorMessage(whatsMissing string, work *WorkOneLang, usedCount int, layout [][]string) string {
	return fmt.Sprintf(`not enough %s to satisfy the given layout:

	· Layout is:
	%v

	· work has only %d %s
	`, whatsMissing, layout, usedCount, whatsMissing)
}

// AutoLayout creates a layout from a given work, placing all paragraphs first, then all mediæ, and finally all links.
// No columns are created
func AutoLayout(work *WorkOneLang) (layout Layout) {
	currentPosition := 0
	for i, paragraph := range work.Paragraphs {
		layout = append(layout, LayedOutElement{
			Type:        "paragraph",
			LayoutIndex: i,
			Positions:   [][]int{{currentPosition, 0}},
			Paragraph:   paragraph,
		})
		currentPosition++
	}
	for i, media := range work.Media {
		layout = append(layout, LayedOutElement{
			Type:        "media",
			LayoutIndex: i,
			Positions:   [][]int{{currentPosition, 0}},
			Media:       media,
		})
		currentPosition++
	}
	for i, link := range work.Links {
		layout = append(layout, LayedOutElement{
			Type:        "link",
			LayoutIndex: i,
			Positions:   [][]int{{currentPosition, 0}},
			Link:        link,
		})
		currentPosition++
	}
	return
}

// LayoutHomogeneous turns a an untyped layout from metadata into a [][]string,
// turning string elements into a one-element slice, so that it can be used
// in loops without type errors
func (metadata WorkMetadata) LayoutHomogeneous() (homo [][]string, err error) {
	if len(metadata.LayoutProper) > 0 {
		return metadata.LayoutProper, nil
	}
	// TODO: also support "direct css grid template areas syntax" where the value of metadata.Layout is a string
	// that could be returned by .CSSGridTemplateAreas (except for quotes, not required when parsing here.)
	for _, row := range metadata.Layout {
		rowType := reflect.TypeOf(row)
		if rowType.Kind() == reflect.String {
			// If it is a string, append a 1-element slice
			homo = append(homo, []string{row.(string)})
		} else if rowType.Kind() == reflect.Array || rowType.Kind() == reflect.Slice {
			// If it's a slice of strings, append it normally
			rowString := make([]string, 0)
			for _, element := range row.([]interface{}) {
				if element == nil {
					// null is a spacer
					rowString = append(rowString, ".")
				} else {
					rowString = append(rowString, fmt.Sprint(element))
				}
			}
			homo = append(homo, rowString)
		} else {
			spew.Dump(row, reflect.TypeOf(row))
			err = fmt.Errorf("%#v is neither a list of string(s) nor a string, it is a %s (%T)", row, rowType.Name(), row)
			return
		}
	}
	return
}

// LayedOut fills the LayoutIndices of every work content element (paragraphs, mediæ and links.)
func (work WorkOneLang) LayedOut() (layout Layout, err error) {
	usedCounts := map[string]int{"paragraph": 0, "media": 0, "link": 0}
	elements := map[string]LayedOutElement{}

	// Coerce the layout into a proper [][]string
	layoutString, err := work.Metadata.LayoutHomogeneous()
	if err != nil {
		panic(err)
	}
	// If it's empty, that means the layout was empty all along:
	// auto-create one.
	if len(layoutString) == 0 {
		return AutoLayout(&work), nil
	}

	// Determine the width of every row
	width := 1
	for _, row := range layoutString {
		width = lcm(width, len(row))
	}

	for i, rowString := range layoutString {
		if len(rowString) == 0 {
			continue
		}

		// Repeating with factor repeatCells
		// Guaranteed to be a whole number since width is precisely the lcm of all lengths of rows
		repeatCells := width / len(rowString)

		for j, cellString := range rowString {
			cell, err := ParseStringCell(cellString)
			if err != nil {
				return layout, fmt.Errorf("while parsing layout cell %q: %w", cellString, err)
			}

			if cell.Index == -1 {
				rowString[j] = cellString + fmt.Sprint(usedCounts[cell.Type]+1)
				usedCounts[cell.Type]++
			} else {
				usedCounts[cell.Type] = cell.Index + 1
			}
		}

	cells:
		for j := 0; j < width; j++ {
			cellString := rowString[j/repeatCells]
			// if !regexp.MustCompile(`[mlp.]\d+`).MatchString(cellString) {
			// 	cellIndex := usedCounts[string(cellString[0])]
			// 	cellString += fmt.Sprint(cellIndex)
			// }
			cell, err := ParseStringCell(cellString)
			if err != nil {
				return layout, fmt.Errorf("while parsing cell %q: %w", cellString, err)
			}

			for key, element := range elements {
				// If the element has already been seen, add this position to its positions
				if key == cell.String() {
					element.Positions = append(element.Positions, []int{i, j})
					elements[key] = element
					continue cells
				}
			}

			element := LayedOutElement{Positions: [][]int{{i, j}}, LayoutIndex: cell.Index, Metadata: &work.Metadata, Type: cell.Type}
			switch cell.Type {
			case "paragraph":
				if cell.Index >= len(work.Paragraphs) {
					return layout, fmt.Errorf(buildLayoutErrorMessage(element.Type, &work, cell.Index, layoutString))
				}
				element.Paragraph = work.Paragraphs[cell.Index]
			case "link":
				if cell.Index >= len(work.Links) {
					return layout, fmt.Errorf(buildLayoutErrorMessage(element.Type, &work, cell.Index, layoutString))
				}
				element.Link = work.Links[cell.Index]
			case "media":
				if cell.Index >= len(work.Media) {
					return layout, fmt.Errorf(buildLayoutErrorMessage(element.Type, &work, cell.Index, layoutString))
				}
				element.Media = work.Media[cell.Index]
				element.GeneralContentType = GeneralContentType(element.Media)
			}
			// printfln("%#v", element)
			elements[cell.String()] = element
			// printfln("%#v", usedCounts)
		}
	}

	for _, element := range elements {
		if element.Type != "spacer" {
			layout = append(layout, element)
		}
	}

	sort.Slice(layout, func(i, j int) bool {
		// Lexical sort of (starting column, starting row)
		iCol, _, iRow, _ := layout[i].PositionBounds()
		jCol, _, jRow, _ := layout[j].PositionBounds()

		return iCol < jCol || (iCol == jCol && iRow < jRow)
	})

	return
}

func (l Layout) PositionsMap() map[string][][]int {
	posmap := make(map[string][][]int)
	for _, element := range l {
		posmap[fmt.Sprintf("%s%d", element.Type, element.LayoutIndex)] = element.Positions
	}
	return posmap
}

func ParseStringCell(stringCell string) (cell Cell, err error) {
	switch stringCell[:1] {
	case "p":
		cell.Type = "paragraph"
	case "m":
		cell.Type = "media"
	case "l":
		cell.Type = "link"
	case ".":
		cell.Type = "spacer"
	default:
		return cell, fmt.Errorf("malformed layout element %#v: unknown cell type", stringCell[:1])
	}
	if len(stringCell) > 1 {
		elementIndex, err := strconv.ParseUint(stringCell[1:], 10, 16)
		if err != nil {
			return cell, fmt.Errorf("malformed layout element %#v: element index %#v is not an integer", stringCell, stringCell[1:])
		}
		cell.Index = int(elementIndex) - 1
		if cell.Index == -1 {
			return cell, fmt.Errorf("manuall-indexed layout cells cannot start at index 0, start at 1.")
		}
	} else {
		cell.Index = -1
	}
	return
}
