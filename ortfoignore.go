package ortfomk

import (
	"errors"
	"github.com/denormal/go-gitignore"
	"os"
	"path/filepath"
)

// closestOrtfoignoreFile parses the .ortfoignore closest to the currentDirectory.
// If no .ortfoignore file is found in currentDirectory, it will be searched for in currentDirectory's parent, recursively,
// until the currentDirectory is the templatesDirectory. If currentDirectory is templatesDirectory, and no .ortfoignore file is found,
// return (nil, nil). If templatesDirectory is not a parent of currentDirectory, an error is returned.
func closestOrtfoignore(currentDirectory string) (gitignore.GitIgnore, error) {
	relativeToTemplates, err := filepath.Rel(g.TemplatesDirectory, currentDirectory)
	if err != nil {
		return nil, errors.New("attempted to find ortfoignore from outside the templates directory")
	}
	checkParent := relativeToTemplates != "."
	ortfoignorePath := filepath.Join(currentDirectory, ".ortfoignore")
	_, err = os.Stat(ortfoignorePath)
	if os.IsNotExist(err) {
		if checkParent {
			return closestOrtfoignore(filepath.Dir(currentDirectory))
		}

		return nil, nil
	}
	return gitignore.NewFromFile(ortfoignorePath)
}
