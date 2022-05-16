package ortfomk

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	exprVM "github.com/antonmedv/expr/vm"
)

var g GlobalData = GlobalData{}
var DynamicPathExpressionsCahe = map[string]*exprVM.Program{}

type Translations map[string]TranslationsOneLang

// GlobalData holds data that is used throughout the whole build process
type GlobalData struct {
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

func SetTranslationsOnGlobalData(translations map[string]TranslationsOneLang) {
	g.Translations = translations
}

func SetDatabaseOnGlobalData(database Database) {
	g.Database = database
}

func ComputeTotalToBuildCount() {
	g.Progress.Total = ToBuildTotalCount(g.TemplatesDirectory)
}

func ToBuildTotalCount(in string) (count int) {
	err := filepath.WalkDir(in, func(path string, entry fs.DirEntry, err error) error {
		if strings.Contains(path, "/mixins/") {
			return nil
		}
		if !(strings.HasSuffix(path, ".pug") || strings.HasSuffix(path, ".html")) {
			return nil
		}
		if err != nil {
			return err
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

// BuildAll builds all page in the given directory, recursively.
func BuildAll(in string) (built []string, err error) {
	err = filepath.WalkDir(in, func(path string, entry fs.DirEntry, err error) error {
		LogDebug("Walking into %s", path)
		if strings.Contains(path, "/mixins/") {
			return nil
		}
		if !(strings.HasSuffix(path, ".pug") || strings.HasSuffix(path, ".html")) {
			return nil
		}
		if err != nil {
			return err
		}

		for _, expr := range DynamicPathExpressions(entry.Name()) {
			switch expr {
			case "work":
				built = append(built, BuildWorkPages(path)...)
				return err
			case "tag":
				built = append(built, BuildTagPages(path)...)
				return err
			case "technology":
				fallthrough
			case "tech":
				built = append(built, BuildTechPages(path)...)
				return err
			case "site":
				built = append(built, BuildSitePages(path)...)
				return err
			}
		}

		built = append(built, BuildRegularPage(path)...)
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

	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("could build technology pages’ template: %s", err)
		return
	}
	for _, tech := range g.Technologies {
		SetCurrentObjectID(tech.URLName)
		built = append(built, BuildPage(using, compiledTemplate, &Hydration{tech: tech})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildSitePages builds all site pages using the template at the given filename
func BuildSitePages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("couldn't read the template: %s", err)
		return
	}

	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("could build site pages’ template: %s", err)
		return
	}
	for _, site := range g.Sites {
		SetCurrentObjectID(site.Name)
		built = append(built, BuildPage(using, compiledTemplate, &Hydration{site: site})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildTagPages builds all tag pages using the given filename
func BuildTagPages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("couldn't read the template: %s", err)
		return
	}

	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("could build tag pages’ template: %s", err)
		return
	}
	for _, tag := range g.Tags {
		SetCurrentObjectID(tag.Singular)
		built = append(built, BuildPage(using, compiledTemplate, &Hydration{tag: tag})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildWorkPages builds all work pages using the given filepath
func BuildWorkPages(using string) (built []string) {
	templateContent, err := os.ReadFile(using)
	if err != nil {
		LogError("coudln't read template: %s", err)
	}

	compiledTemplate, err := CompileTemplate(using, templateContent)
	if err != nil {
		LogError("couldn't build work pages’ template: %s", err)
		return
	}
	for _, work := range g.Works {
		SetCurrentObjectID(work.ID)
		built = append(built, BuildPage(using, compiledTemplate, &Hydration{work: work})...)
		SetCurrentObjectID("")
	}
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

	compiledTemplate, err := CompileTemplate(path, templateContent)
	if err != nil {
		LogError("could not build the page’s template: %s", err)
		return
	}
	LogDebug("finished compiling")

	return BuildPage(path, compiledTemplate, &Hydration{})
}

// BuildPage builds a single page
func BuildPage(pageName string, compiledTemplate []byte, hydration *Hydration) (built []string) {
	for _, language := range []string{"fr", "en"} {
		hydration.language = language
		outPath, err := hydration.GetDistFilepath(pageName)
		if err != nil {
			LogError("Invalid path: %s", err)
			continue
		}
		if outPath == "" {
			LogDebug("Skipped path %s", pageName)
			continue
		}

		Status(StepBuildPage, ProgressDetails{
			File:     pageName,
			Language: language,
			OutFile:  outPath,
		})
		content, err := RunTemplate(
			hydration,
			pageName,
			compiledTemplate,
		)
		if err != nil {
			// PrintTemplateErrorMessage("executing template", NameOfTemplate(pageName, *hydration), string(compiledTemplate), err, "js")
			LogError("couldn't execute template: %s", err)
			continue
		}
		content = g.Translations[language].TranslateHydrated(content)
		for _, link_ := range AllLinks(content).ToSlice() {
			link := link_.(string)
			if _, exists := g.HTTPLinks[link]; exists {
				g.HTTPLinks[link] = append(g.HTTPLinks[link], outPath)
			}
			g.HTTPLinks[link] = []string{outPath}
		}
		os.MkdirAll(filepath.Dir(outPath), 0777)
		if strings.HasSuffix(outPath, ".pdf") {
			WritePDF(content, outPath)
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
