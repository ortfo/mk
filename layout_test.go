package ortfomk

import (
	"fmt"
	"strings"
	"testing"

	ortfodb "github.com/ortfo/db"
	"github.com/stretchr/testify/assert"
)

func TestLayedOutElementCSS(t *testing.T) {
	assert.Equal(t, `grid-column: 1 / 2; grid-row: 1 / 2;`, LayedOutElement{Positions: [][]int{{0, 0}}}.CSS())
	assert.Equal(t, `grid-column: 1 / 2; grid-row: 3 / 8;`, LayedOutElement{Positions: [][]int{{2, 0}, {3, 0}, {4, 0}, {5, 0}, {6, 0}}}.CSS())
	assert.Equal(t, `grid-column: 4 / 7; grid-row: 4 / 5;`, LayedOutElement{Positions: [][]int{{3, 3}, {3, 4}, {3, 5}}}.CSS())
	assert.Equal(t, `grid-column: 10 / 14; grid-row: 6 / 9;`, LayedOutElement{Positions: [][]int{{5, 9}, {5, 10}, {5, 11}, {5, 12}, {6, 9}, {6, 10}, {6, 11}, {6, 12}, {7, 9}, {7, 10}, {7, 11}, {7, 12}}}.CSS())
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
		"paragraph1": {{0, 0}, {0, 1}},
		"link1":      {{1, 0}},
		"link2":      {{1, 1}},
		"media1":     {{2, 0}, {3, 0}},
		"paragraph2": {{2, 1}},
		"link3":      {{3, 1}},
		"media2":     {{4, 0}, {4, 1}},
	}, val.PositionsMap())

	val, err = workWithLayout(`p; m1, p; m1, p; p; p; p; l, l; p; m; p, .; p, .; p, .`).LayedOut()
	assert.NoError(t, err)
	assert.Equal(t, map[string][][]int{
		"paragraph1":  {{0, 0}, {0, 1}},
		"media1":      {{1, 0}, {2, 0}},
		"paragraph2":  {{1, 1}},
		"paragraph3":  {{2, 1}},
		"paragraph4":  {{3, 0}, {3, 1}},
		"paragraph5":  {{4, 0}, {4, 1}},
		"paragraph6":  {{5, 0}, {5, 1}},
		"link1":       {{6, 0}},
		"link2":       {{6, 1}},
		"paragraph7":  {{7, 0}, {7, 1}},
		"media2":      {{8, 0}, {8, 1}},
		"paragraph8":  {{9, 0}},
		"paragraph9":  {{10, 0}},
		"paragraph10": {{11, 0}},
	},
		val.PositionsMap())
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
