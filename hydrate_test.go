package ortfomk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"rogchap.com/v8go"
)

func TestLineAndColumn(t *testing.T) {
	line, column := lineAndColumn(&v8go.JSError{Location: "/home/ewen/projects/portfolio/src/:language/:work.pug.js:154:3515"})
	assert.Equal(t, 154, line)
	assert.Equal(t, 3515, column)
}
