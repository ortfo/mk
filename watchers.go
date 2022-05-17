package ortfomk

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
)

// StartWatcher starts a watcher that listents for file changes in src/*.pug and i18n/*.mo
// - Re-build only the necessary files when content changes,
// - Stops when gallery.pug is moved
// - Updates references to a file when it is moved
// - Warns when deleting a file that is depended upon
func StartWatcher(db Database) {
	watchPattern := regexp.MustCompile(`^.+\.(pug|mo)`)
	//
	// Content changes (new files or contents modified)
	//
	w := watcher.New()
	w.FilterOps(watcher.Create, watcher.Write, watcher.Move)
	w.AddFilterHook(watcher.RegexFilterHook(watchPattern, false))

	Status("Waiting for changes", ProgressDetails{})
	go func() {
		for {
			select {
			case event := <-w.Event:
				dependents := make([]string, 0)
				if strings.HasSuffix(event.Path, ".pug") {
					dependents = DependentsOf(g.TemplatesDirectory, event.Path, 10)
				}
				switch event.Op {
				case watcher.Create:
					fallthrough
				case watcher.Write:
					if strings.HasSuffix(event.Path, ".mo") {
						LogInfo("Compiled translations changed: re-building everything")
						translations, err := LoadTranslations()
						SetTranslationsOnGlobalData(translations)
						if err != nil {
							LogError("Couldn't load the translation files: %s", err)
						}
						BuildAll("src")
					} else if strings.HasSuffix(event.Path, ".pug") {
						LogInfo("Building file [bold]%s[/bold] and its dependents [bold]%s[/bold]", GetPathRelativeToSrcDir(event.Path), strings.Join(dependents, ", "))
						for _, filePath := range append(dependents, event.Path) {
							if strings.Contains(filePath, ":work") {
								BuildWorkPages(filePath)
							} else if strings.Contains(filePath, ":tag") {
								BuildTagPages(filePath)
							} else if strings.Contains(filePath, ":technology") {
								BuildTechPages(filePath)
							} else {
								BuildRegularPage(filePath)
							}
						}
						for _, lang := range []string{"fr", "en"} {
							g.Translations[lang].SavePO()
						}
					}
				case watcher.Remove:
					if len(dependents) > 0 {
						LogWarning("Files %s depended on %s, which was removed", strings.Join(dependents, ", "), event.Path)
					}
				case watcher.Rename:
					if GetPathRelativeToSrcDir(event.OldPath) == "gallery.pug" {
						LogWarning("gallery.pug was renamed, exiting: you'll need to update references to the filename in Go files.")
						w.Close()
					}
					LogDebug("%s -> %s, checking dependents %v", event.OldPath, event.Path, dependents)
					if len(dependents) > 0 {
						LogInfo("%s was renamed to %s: Updating references in %s", GetPathRelativeToSrcDir(event.OldPath), GetPathRelativeToSrcDir(event.Path), strings.Join(dependents, ", "))
						for _, filePath := range dependents {
							UpdateExtendsStatement(filePath, event.OldPath, event.Path)
						}
					}
				}
				fmt.Println("\r\033[K")
			case err := <-w.Error:
				LogError("An errror occured while watching changes in src/: %s", err)
			case <-w.Closed:
				return
			}
		}
	}()

	if err := w.AddRecursive(g.TemplatesDirectory); err != nil {
		LogError("Couldn't add src/ to watcher: %s", err)
	}

	if err := w.AddRecursive("i18n"); err != nil {
		LogError("Couldn't add i18n/ to watcher: %s", err)
	}

	if err := w.Start(100 * time.Millisecond); err != nil {
		LogError("Couldn't start the watcher: %s", err)
	}

}

// UpdateExtendsStatement renames the file referenced by an extends statement
func UpdateExtendsStatement(in string, from string, to string) {
	extendsPattern := regexp.MustCompile(`(?m)^extends\s+(?:src/)?` + from + `(?:\.pug)?\s*$`)
	file, err := os.Open(in)
	if err != nil {
		LogError(fmt.Sprintf("While updating the extends statement in %s from %s to %s: could not open file %s", in, from, to, in), err)
	}
	defer file.Close()
	contents, err := os.ReadFile(in)
	if err != nil {
		LogError(fmt.Sprintf("While updating the extends statement in %s from %s to %s: could not read file %s", in, from, to, in), err)
	}
	_, err = file.Write(
		extendsPattern.ReplaceAll(contents, []byte("extends "+to)),
	)
	if err != nil {
		LogError(fmt.Sprintf("While updating the extends statement in %s from %s to %s: could not write to file %s", in, from, to, in), err)
	}
}

// GetPathRelativeToSrcDir takes an _absolute_ path and returns the part after (not containing) source
func GetPathRelativeToSrcDir(absPath string) string {
	relative, err := filepath.Rel(g.TemplatesDirectory, absPath)
	if err != nil {
		panic(err)
	}

	return relative
}

// Dependencies returns all the .pug files referenced in extends or include statements by content.
// All the dependencies' paths are as-is (meaning relative to content's file's parent directory), except that .pug is added when it's missing.
func Dependencies(content string) []string {
	dependencies := make([]string, 0)
	for _, match := range regexp.MustCompile(`(?m)^(?:extends|include)\s+(.+)\s*$`).FindAllStringSubmatch(content, -1) {
		withExtension := match[1]
		if !strings.HasSuffix(withExtension, ".pug") {
			withExtension += ".pug"
		}
		dependencies = append(dependencies, withExtension)
	}
	return dependencies
}

// DependentsOf returns an array of pages' filepaths that depend
// on the given filepath (through `extends` or `intoGallery`)
// This function is recursive, dependents of dependents are also included.
// The returned array is has the same order as the build order required to correctly update dependencies before their dependents
// maxDepth is used to specify how deeply it should recurse (i.e. how many times it should call itself)
func DependentsOf(searchIn string, pageFilepath string, maxDepth uint) (dependents []string) {
	err := filepath.WalkDir(searchIn, func(path string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !dirEntry.IsDir() && strings.HasSuffix(dirEntry.Name(), ".pug") {
			content, err := os.ReadFile(path)

			if err != nil {
				LogError("Not checking for dependence on "+pageFilepath+": could not read file "+path, err)
				return nil
			}

			// If this file extends the given file,
			// or if the given file is src/gallery.pug and it uses | intoGallery (and therefore depends on src/gallery.pug)
			// add this file to the dependents
			for _, dep := range Dependencies(string(content)) {
				absoluteDep := filepath.Join(filepath.Dir(path), dep)
				LogDebug("testing %s: %s == %s? checking with %s", path, dep, pageFilepath, absoluteDep)
				if err != nil {
					LogError("while analyzing %s's dependencies: %s", pageFilepath, err)
					continue
				}

				if pageFilepath == absoluteDep {
					dependents = append(dependents, path)
					// Add dependents of dependent after (they need to be built _after_ the dependent because they themselves depend on the former)
					if maxDepth > 1 {
						dependents = append(dependents, DependentsOf(searchIn, path, maxDepth-1)...)
					} else {
						LogWarning("While looking for dependents for %s: Maximum recursion depth reached, not recursing any further. You might have a circular dependency.", GetPathRelativeToSrcDir(pageFilepath))
					}
				}
			}
		}
		return nil
	})

	if err != nil {
		LogError("While looking for dependents on "+GetPathRelativeToSrcDir(pageFilepath), err)
	}
	return
}
