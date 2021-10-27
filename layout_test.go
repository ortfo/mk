package ortfomk

import (
	"fmt"
	"strings"
	"testing"

	ortfodb "github.com/ortfo/db"
	"github.com/stretchr/testify/assert"
)

func TestLayedOutElementCSS(t *testing.T) {
	assert.Equal(t, `grid-row: 1 / 2; grid-column: 1 / 2;`, LayedOutElement{Positions: [][]int{{0, 0}}}.CSS())
	assert.Equal(t, `grid-row: 3 / 8; grid-column: 1 / 2;`, LayedOutElement{Positions: [][]int{{2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}}}.CSS())
	assert.Equal(t, `grid-row: 4 / 5; grid-column: 4 / 7;`, LayedOutElement{Positions: [][]int{{3, 3}, {3, 4}, {3, 5}}}.CSS())
	assert.Equal(t, `grid-row: 6 / 9; grid-column: 10 / 14;`, LayedOutElement{Positions: [][]int{{5, 9}, {5, 10}, {5, 11}, {5, 12}, {6, 9}, {6, 10}, {6, 11}, {6, 12}, {7, 9}, {7, 10}, {7, 11}, {7, 12}}}.CSS())
}

func TestWorkMetadataLayoutHomogeneous(t *testing.T) {
	layout := []interface{}{"p", []interface{}{"p", "l"}, []interface{}{"l"}, []interface{}{"m", "m2", "m2"}}
	val, err := WorkMetadata{Layout: layout}.LayoutHomogeneous()
	assert.NoError(t, err)
	assert.Equal(t, [][]string{{"p"}, {"p", "l"}, {"l"}, {"m", "m2", "m2"}}, val)
}

func TestWorkOneLangLayedOut(t *testing.T) {
	val, err := workWithLayout(`p; l, l; m1, p; m1, l; m`).LayedOut()
	assert.NoError(t, err)
	assert.Equal(t, map[string][][]int{
		"paragraph0": {{0, 0}, {0, 1}},
		"link0":      {{1, 0}},
		"link1":      {{1, 1}},
		"media0":     {{2, 0}, {3, 0}},
		"paragraph1": {{2, 1}},
		"link2":      {{3, 1}},
		"media1":     {{4, 0}, {4, 1}},
	}, val.PositionsMap())

	val, err = workWithLayout(`p; m1, p; m1, p; p; p; p; l, l; p; m; p, .; p, .; p, .`).LayedOut()
	assert.NoError(t, err)
	assert.Equal(t, map[string][][]int{
		"paragraph0":  {{0, 0}, {0, 1}},
		"media0":      {{1, 0}, {2, 0}},
		"paragraph1":  {{1, 1}},
		"paragraph2":  {{2, 1}},
		"paragraph3":  {{3, 0}, {3, 1}},
		"paragraph4":  {{4, 0}, {4, 1}},
		"paragraph5":  {{5, 0}, {5, 1}},
		"link0":       {{6, 0}},
		"link1":       {{6, 1}},
		"paragraph6":  {{7, 0}, {7, 1}},
		"media1":      {{8, 0}, {8, 1}},
		"paragraph7":  {{9, 0}},
		"paragraph8":  {{10, 0}},
		"paragraph9": {{11, 0}},
	},
		val.PositionsMap())
}

func TestLayoutSorted(t *testing.T) {
	val, _ := workWithLayout(`p; m1, p; m1, p; p; p; p; l, l; p; m; p, .; p, .; p, .`).LayedOut()

	assert.Equal(t, []string{
		"paragraph0",
		"media0",
		"paragraph1",
		"paragraph2",
		"paragraph3",
		"paragraph4",
		"paragraph5",
		"link0",
		"link1",
		"paragraph6",
		"media1",
		"paragraph7",
		"paragraph8",
		"paragraph9",
	}, layoutCellStrings(val))
}

func TestAutoLayout(t *testing.T) {
	assert.Equal(t, Layout{
		{
			Type:        "paragraph",
			LayoutIndex: 0,
			Positions:   [][]int{{0, 0}},
			Paragraph:   ortfodb.Paragraph{ID: "a"},
		},
		{
			Type:        "paragraph",
			LayoutIndex: 1,
			Positions:   [][]int{{1, 0}},
			Paragraph:   ortfodb.Paragraph{ID: "b"},
		},
		{
			Type:        "media",
			LayoutIndex: 0,
			Positions:   [][]int{{2, 0}},
			Media:       ortfodb.Media{ID: "a"},
		},
		{
			Type:        "media",
			LayoutIndex: 1,
			Positions:   [][]int{{3, 0}},
			Media:       ortfodb.Media{ID: "b"},
		},
		{
			Type:        "media",
			LayoutIndex: 2,
			Positions:   [][]int{{4, 0}},
			Media:       ortfodb.Media{ID: "c"},
		},
		{
			Type:        "link",
			LayoutIndex: 0,
			Positions:   [][]int{{5, 0}},
			Link:        ortfodb.Link{ID: "a"},
		},
	}, AutoLayout(&WorkOneLang{
		Paragraphs: []ortfodb.Paragraph{{ID: "a"}, {ID: "b"}},
		Media:      []ortfodb.Media{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		Links:      []ortfodb.Link{{ID: "a"}},
	}))
}

// layoutCellStrings maps a Layout to a list of <ElementType><ElementLayoutIndex> strings, e.g. []string{"paragraph1", "media1"}
func layoutCellStrings(layout Layout) []string {
	cellStrings := make([]string, 0)
	for _, element := range layout {
		cellStrings = append(cellStrings, element.Type+fmt.Sprint(element.LayoutIndex))
	}
	return cellStrings
}

// workWithLayout creates a WorkOneLang, given a textual representation of the layout.
// It adds enough empty paragraphs/media/links to not run out of elements while laying them out.
// The textual representation of the layout is a string where "; " separates rows and ", " separates columns:
//
//		p, l; m; .
//
// Represents
//
//		[][]string{{"p", "l"}, {"m"}, {"."}}
//
func workWithLayout(layoutString string) WorkOneLang {
	layout := make([][]string, 0)

	for _, row := range strings.Split(layoutString, "; ") {
		layout = append(layout, strings.Split(row, ", "))
	}

	elementsCount := 0
	for _, row := range layout {
		elementsCount += len(row)
	}

	work := WorkOneLang{
		Metadata: WorkMetadata{LayoutProper: layout},
	}

	for i := 0; i < elementsCount; i++ {
		work.Paragraphs = append(work.Paragraphs, ortfodb.Paragraph{ID: fmt.Sprint(i)})
		work.Links = append(work.Links, ortfodb.Link{ID: fmt.Sprint(i)})
		work.Media = append(work.Media, ortfodb.Media{ID: fmt.Sprint(i)})
	}

	return work
}
