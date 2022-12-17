package ortfomk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateDynamicPath(t *testing.T) {
	result, err := EvaluateDynamicPath(&Hydration{
		work: Work{
			ID: "neptune",
		},
		language: "fr",
	}, "/home/ewen/projects/portfolio/src/:language/:work/player.pug")
	assert.NoError(t, err)
	assert.Equal(t, "/home/ewen/projects/portfolio/src/fr/neptune/player.pug", result)
}

func TestDynamicPathExpressions(t *testing.T) {
	result := DynamicPathExpressions("/home/ewen/projects/portfolio/src/:language/:work/player.pug")
	assert.Equal(t, []string{"language", "work"}, result)
}
