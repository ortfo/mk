package ortfomk

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var g GlobalData = GlobalData{}

// GlobalData holds data that is used throughout the whole build process
type GlobalData struct {
	Translations
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

func SetTranslationsOnGlobalData(translations Translations) {
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
		if strings.HasPrefix(entry.Name(), ":work") || strings.HasPrefix(entry.Name(), ":if(work") {
			incrementBy := len(g.Works) * len([]string{"fr", "en"})
			LogDebug("incrementing total tobuild count[:tag]: %d + %d -> %d", incrementBy, count, count+incrementBy)
			count += incrementBy
		} else if strings.HasPrefix(entry.Name(), ":tag") {
			incrementBy := len(g.Tags) * len([]string{"fr", "en"})
			LogDebug("incrementing total tobuild count[:work]: %d + %d -> %d", incrementBy, count, count+incrementBy)
			count += incrementBy
		} else if strings.HasPrefix(entry.Name(), ":technology") {
			incrementBy := len(g.Technologies) * len([]string{"fr", "en"})
			LogDebug("incrementing total tobuild count[:technology]: %d + %d -> %d", incrementBy, count, count+incrementBy)
			count += incrementBy
		} else if strings.HasPrefix(entry.Name(), ":site") {
			incrementBy := len(g.Sites) * len([]string{"fr", "en"})
			LogDebug("incrementing total tobuild count[:site]: %d + %d -> %d", incrementBy, count, count+incrementBy)
			count += incrementBy
		} else if strings.HasPrefix(entry.Name(), ":") {
			return nil
		} else {
			incrementBy := 1 * len([]string{"fr", "en"})
			LogDebug("incrementing total tobuild count: %d + %d -> %d", incrementBy, count, count+incrementBy)
			count += incrementBy
		}
		return err
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
		if strings.HasPrefix(entry.Name(), ":work") || strings.HasPrefix(entry.Name(), ":if(work") {
			built = append(built, BuildWorkPages(path)...)
		} else if strings.HasPrefix(entry.Name(), ":tag") {
			built = append(built, BuildTagPages(path)...)
		} else if strings.HasPrefix(entry.Name(), ":technology") {
			built = append(built, BuildTechPages(path)...)
		} else if strings.HasPrefix(entry.Name(), ":site") {
			built = append(built, BuildSitePages(path)...)
		} else if strings.HasPrefix(entry.Name(), ":") {
			return fmt.Errorf("dynamic path %s uses unknown variable %s", path, strings.TrimPrefix(entry.Name(), ":"))
		} else {
			built = append(built, BuildRegularPage(path)...)
		}
		return err
	})
	return
}

// BuildTechPages builds all technology pages using `using`
func BuildTechPages(using string) (built []string) {
	SetCurrentObjectID(using)
	templateHTML, err := ConvertTemplateIfNeeded(using)
	if err != nil {
		LogError("couldn't convert technology template: %s", err)
		return
	}
	for _, tech := range g.Technologies {
		SetCurrentObjectID(tech.URLName)
		built = append(built, BuildPage(using, templateHTML, &Hydration{tech: tech})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildSitePages builds all site pages using the template at the given filename
func BuildSitePages(using string) (built []string) {
	SetCurrentObjectID(using)
	templateHTML, err := ConvertTemplateIfNeeded(using)
	if err != nil {
		LogError("couldn't convert site template: %s", err)
		return
	}
	for _, site := range g.Sites {
		SetCurrentObjectID(site.Name)
		built = append(built, BuildPage(using, templateHTML, &Hydration{site: site})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildTagPages builds all tag pages using the given filename
func BuildTagPages(using string) (built []string) {
	SetCurrentObjectID(using)
	templateHTML, err := ConvertTemplateIfNeeded(using)
	if err != nil {
		LogError("couldn't convert template: %s", err)
		return
	}
	for _, tag := range g.Tags {
		SetCurrentObjectID(tag.Singular)
		built = append(built, BuildPage(using, templateHTML, &Hydration{tag: tag})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildWorkPages builds all work pages using the given filepath
func BuildWorkPages(using string) (built []string) {
	SetCurrentObjectID(using)
	templateHTML, err := ConvertTemplateIfNeeded(using)
	if err != nil {
		LogError("couldn't build work pages: %s", err)
		return
	}
	for _, work := range g.Works {
		SetCurrentObjectID(work.ID)
		built = append(built, BuildPage(using, templateHTML, &Hydration{work: work})...)
		SetCurrentObjectID("")
	}
	return
}

// BuildRegularPage builds a given page that isn't dynamic (i.e. does not require object data,
// as opposed to work, tag and tech pages)
func BuildRegularPage(filepath string) (built []string) {
	templateContent, err := ConvertTemplateIfNeeded(filepath)
	if err != nil {
		LogError("could not convert the template: %s", err)
		return
	}
	return BuildPage(filepath, templateContent, &Hydration{})
}

// BuildPage builds a single page
func BuildPage(using string, templateHTML string, hydration *Hydration) (built []string) {
	for _, language := range []string{"fr", "en"} {
		hydration.language = language
		outPath := hydration.GetDistFilepath(using)

		Status(StepBuildPage, ProgressDetails{
			File:     using,
			Language: language,
			OutFile:  outPath,
		})
		templ, err := ParseTemplate(
			language,
			using,
			templateHTML,
		)
		if err != nil {
			PrintTemplateErrorMessage("parsing template", using, templateHTML, err, "html")
			return
		}
		content, err := ExecuteTemplate(
			templ,
			language,
			*hydration,
		)
		if err != nil {
			PrintTemplateErrorMessage("executing template", NameOfTemplate(templ, *hydration), templateHTML, err, "html")
			continue
		}
		content = g.Translations.TranslateHydrated(content, language)
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
