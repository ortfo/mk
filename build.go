package ortfomk

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	exprVM "github.com/antonmedv/expr/vm"
	"github.com/stoewer/go-strcase"
	"gopkg.in/yaml.v3"
	v8 "rogchap.com/v8go"
)

var g GlobalData = GlobalData{}
var DynamicPathExpressionsCahe = map[string]*exprVM.Program{}

type Translations map[string]*TranslationsOneLang

// GlobalData holds data that is used throughout the whole build process
type GlobalData struct {
	mu sync.Mutex

	Translations Translations
	Database
	// Maps each link to the pages in which they appear
	HTTPLinks         map[string][]string
	Spinner           Spinner
	CurrentObjectID   string
	CurrentOutputFile string
	CurrentLanguage   string
	Progress          struct {
		Step       BuildStep
		Resolution int
		File       string
		Current    int
		Total      int
	}
	Flags              Flags
	OutputDirectory    string
	TemplatesDirectory string
	AdditionalData     map[string]interface{}
}

type Flags struct {
	ProgressFile string
	Silent       bool
}

// WarmUp needs to be run before any building starts.
// It sets the global data, scans the template directory
// to determine the total number of pages to build, and starts the spinner.
func WarmUp(data GlobalData) {
	g = data
	g.Spinner = CreateSpinner()
	g.Spinner.Start()
}

// CoolDown needs to be stop before the program exits.
// It properly stops the spinner.
func CoolDown() {
	g.Spinner.Stop()
}

func SetGlobalData(data GlobalData) {
	g = data
}

func SetTranslationsOnGlobalData(translations map[string]*TranslationsOneLang) {
	g.Translations = translations
}

func SetDatabaseOnGlobalData(database Database) {
	g.Database = database
}

func LoadAdditionalData(filesToLoad []string) (additionalData map[string]interface{}, err error) {
	additionalData = make(map[string]interface{})
	for _, file := range filesToLoad {
		var loaded interface{}
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return additionalData, fmt.Errorf("while reading %s: %w", file, err)
		}

		err = yaml.Unmarshal([]byte(content), &loaded)
		if err != nil {
			return additionalData, fmt.Errorf("while parsing %s: %w", file, err)
		}

		if loaded == nil {
			LogWarning("Loaded data from %s is null", file)
		}

		additionalData[strcase.LowerCamelCase(filepathStem(file))] = loaded
	}
	return additionalData, nil
}

func ComputeTotalToBuildCount() {
	g.Progress.Total = ToBuildTotalCount(g.TemplatesDirectory)
}

func ToBuildTotalCount(in string) (count int) {
	err := filepath.WalkDir(in, func(path string, entry fs.DirEntry, err error) error {
		currentDirectory := filepath.Dir(path)
		if strings.Contains(path, "/mixins/") {
			return nil
		}
		if !(strings.HasSuffix(path, ".pug") || strings.HasSuffix(path, ".html")) {
			return nil
		}
		if err != nil {
			return err
		}
		ortfoignore, err := closestOrtfoignore(currentDirectory)
		if err != nil {
			return err
		}
		if ortfoignore != nil && ortfoignore.Ignore(path) {
			LogDebug("ignoring %s because of ortfoignore at %s", path, filepath.Join(ortfoignore.Base(), ".ortfoignore"))
			return nil
		}
		LogDebug("walking into %s", path)
		countForPath := 1
		for _, expression := range DynamicPathExpressions(path) {
			LogDebug("Adding expression %s to count", expression)
			switch expression {
			case "work":
				countForPath *= len(g.Works)
			case "tag":
				countForPath *= len(g.Tags)
			case "technology":
				fallthrough
			case "tech":
				countForPath *= len(g.Technologies)
			case "site":
				countForPath *= len(g.Sites)
			default:
				if regexp.MustCompile(`^lang(uage)?\s+is\s+.+$`).MatchString(expression) {
					countForPath *= 1 // bruh moment
				} else if expression == "language" || expression == "lang" {
					countForPath *= len([]string{"fr", "en"})
				}
			}
		}
		count += countForPath
		LogDebug("count is now %d", count)
		return nil
	})
	if err != nil {
		LogError("couldn't count the total number of pages to build: %s", err)
	}
	return
}

// BuildAll builds pages from templates found in the given directory in parallel, using the given number of goroutines (workersCount).
// if workersCount is 0 or less, it is set to the number of page templates to compile.
func BuildAll(in string, workersCount int) (built []string, httpLinks map[string][]string, err error) {
	toBuildChannel := make(chan string)
	httpLinks = g.HTTPLinks
	LogDebug("scanning for things to build")
	toBuild, err := ScanAll(in)
	if err != nil {
		return built, httpLinks, fmt.Errorf("while scanning templates directory: %w", err)
	}

	if workersCount <= 0 {
		workersCount = len(toBuild)
	}

	var builtMutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(workersCount)

	LogDebug("launching 5 parallel build subroutines")
	for i := 0; i < workersCount; i++ {
		go func(toBuildChannel chan string) {
			for {
				newlyBuilt := make([]string, 0)
				path, more := <-toBuildChannel

				if !more {
					wg.Done()
					return
				}

				for _, expr := range DynamicPathExpressions(path) {
					switch expr {
					case "work":
						newlyBuilt = append(newlyBuilt, BuildWorkPages(path)...)
					case "tag":
						newlyBuilt = append(newlyBuilt, BuildTagPages(path)...)
					case "technology":
						newlyBuilt = append(newlyBuilt, BuildTechPages(path)...)
					case "site":
						newlyBuilt = append(newlyBuilt, BuildSitePages(path)...)
					}
				}

				newlyBuilt = append(newlyBuilt, BuildRegularPage(path)...)
				builtMutex.Lock()
				built = append(built, newlyBuilt...)
				builtMutex.Unlock()
			}
		}(toBuildChannel)
	}

	LogDebug("starting to fill toBuild channel")
	for _, path := range toBuild {
		toBuildChannel <- path
	}
	close(toBuildChannel)
	wg.Wait()
	return
}

// ScanAll scans the given directory for paths to build, recursively.
func ScanAll(in string) (toBuild []string, err error) {
	err = filepath.WalkDir(in, func(path string, entry fs.DirEntry, err error) error {
		// LogDebug("Walking into %s", path)
		currentDirectory := filepath.Dir(path)
		if strings.Contains(path, "/mixins/") {
			return nil
		}
		if !(strings.HasSuffix(path, ".pug") || strings.HasSuffix(path, ".html")) {
			return nil
		}
		if err != nil {
			return err
		}
		ortfoignore, err := closestOrtfoignore(currentDirectory)
		if err != nil {
			return err
		}
		if ortfoignore != nil && ortfoignore.Ignore(path) {
			LogDebug("ignoring %s because of ortfoignore at %s", path, filepath.Join(ortfoignore.Base(), ".ortfoignore"))
			return nil
		}

		toBuild = append(toBuild, path)
		return err
	})
	return
}

// BuildTechPages builds all technology pages using `using`
func BuildTechPages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("couldn't read the template: %s", err)
		return
	}
	javascriptRuntime := v8.NewIsolate()
	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("could build technology pages’ template: %s", err)
		return
	}
	for _, tech := range g.Technologies {
		SetCurrentObjectID(tech.URLName)
		built = append(built, BuildPage(javascriptRuntime, using, compiledTemplate, &Hydration{tech: tech})...)
		SetCurrentObjectID("")
	}
	javascriptRuntime.Dispose()
	return
}

// BuildSitePages builds all site pages using the template at the given filename
func BuildSitePages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("couldn't read the template: %s", err)
		return
	}

	javascriptRuntime := v8.NewIsolate()
	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("could build site pages’ template: %s", err)
		return
	}
	for _, site := range g.Sites {
		SetCurrentObjectID(site.Name)
		built = append(built, BuildPage(javascriptRuntime, using, compiledTemplate, &Hydration{site: site})...)
		SetCurrentObjectID("")
	}
	javascriptRuntime.Dispose()
	return
}

// BuildTagPages builds all tag pages using the given filename
func BuildTagPages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("couldn't read the template: %s", err)
		return
	}

	javascriptRuntime := v8.NewIsolate()
	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("could build tag pages’ template: %s", err)
		return
	}
	for _, tag := range g.Tags {
		SetCurrentObjectID(tag.Singular)
		built = append(built, BuildPage(javascriptRuntime, using, compiledTemplate, &Hydration{tag: tag})...)
		SetCurrentObjectID("")
	}
	javascriptRuntime.Dispose()
	return
}

// BuildWorkPages builds all work pages using the given filepath
func BuildWorkPages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("coudln't read template: %s", err)
	}

	javascriptRuntime := v8.NewIsolate()
	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("couldn't build work pages’ template: %s", err)
		return
	}
	for _, work := range g.Works {
		SetCurrentObjectID(work.ID)
		built = append(built, BuildPage(javascriptRuntime, using, compiledTemplate, &Hydration{work: work})...)
		SetCurrentObjectID("")
	}
	javascriptRuntime.Dispose()
	return
}

// BuildRegularPage builds a given page that isn't dynamic (i.e. does not require object data,
// as opposed to work, tag and tech pages)
func BuildRegularPage(path string) (built []string) {
	SetCurrentObjectID(strings.TrimSuffix(filepath.Base(g.Progress.File), filepath.Ext(g.Progress.File)))
	templateContent, err := os.ReadFile(path)
	if err != nil {
		LogError("couldn't read the template: %s", err)
		return
	}

	javascriptRuntime := v8.NewIsolate()
	compiledTemplate, err := CompileTemplate(path, templateContent)
	if err != nil {
		LogError("could not build the page’s template: %s", err)
		return
	}
	LogDebug("finished compiling")

	built = BuildPage(javascriptRuntime, path, compiledTemplate, &Hydration{})
	javascriptRuntime.Dispose()
	return built
}

// BuildPage builds a single page
func BuildPage(javascriptRuntime *v8.Isolate, pageName string, compiledTemplate []byte, hydration *Hydration) (built []string) {
	// Add additional data to hydration
	for _, language := range []string{"fr", "en"} {
		hydration.language = language
		outPath, err := hydration.GetDistFilepath(pageName)
		if err != nil {
			LogError("Invalid path: %s", err)
			continue
		}
		if outPath == "" {
			// LogDebug("Skipped path %s", pageName)
			continue
		}

		Status(StepBuildPage, ProgressDetails{
			File:     pageName,
			Language: language,
			OutFile:  outPath,
		})
		content, err := RunTemplate(
			javascriptRuntime,
			hydration,
			pageName,
			compiledTemplate,
		)
		if err != nil {
			// PrintTemplateErrorMessage("executing template", NameOfTemplate(pageName, *hydration), string(compiledTemplate), err, "js")
			LogError("couldn't execute template %s with %s: %s", pageName, hydration.Name(), err)
			continue
		}
		content = g.Translations[language].TranslateHydrated(content)
		g.mu.Lock()
		for _, link_ := range AllLinks(content).ToSlice() {
			link := link_.(string)
			if _, exists := g.HTTPLinks[link]; exists {
				g.HTTPLinks[link] = append(g.HTTPLinks[link], outPath)
			}
			g.HTTPLinks[link] = []string{outPath}
		}
		g.mu.Unlock()
		os.MkdirAll(filepath.Dir(outPath), 0777)
		LogDebug("outputting to %s", outPath)
		if strings.HasSuffix(outPath, ".pdf") {
			WritePDF(content, outPath)
			ioutil.WriteFile(strings.TrimSuffix(outPath, ".pdf")+".html", []byte(content), 0777)
		} else {
			ioutil.WriteFile(outPath, []byte(content), 0777)
		}
		built = append(built, outPath)
		progressWriteErr := IncrementProgress()
		if progressWriteErr != nil {
			LogError("couldn't write progress to file: %s", progressWriteErr)
		}
	}
	return
}
