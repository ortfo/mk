package ortfomk

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	po "github.com/chai2010/gettext-go/po"
	mapset "github.com/deckarep/golang-set"
	"github.com/yosssi/gohtml"
	"golang.org/x/net/html"
)

const SourceLanguage = "en"

// TranslationsOneLang holds both the gettext catalog from the .mo file
// and a po file object used to update the .po file (e.g. when discovering new translatable strings)
type TranslationsOneLang struct {
	poFile          po.File
	seenMsgIds      mapset.Set
	missingMessages []po.Message
	language        string
}

func (t TranslationsOneLang) WriteUnusedMsgIds() error {
	to := fmt.Sprintf("i18n/%s-unused-msgids.yaml", t.language)
	ioutil.WriteFile(to, []byte("# Generated at "+time.Now().String()+"\n"), 0644)
	file, err := os.OpenFile(to, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, message := range t.poFile.Messages {
		if !t.seenMsgIds.Contains(message.MsgId) {
			_, err = file.WriteString(fmt.Sprintf("- %#v\n", message.MsgId))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// TranslateToLanguage translates the given html node to french or english, removing translation-related attributes
func Translate(language string, root *html.Node) string {
	// Open files
	doc := goquery.NewDocumentFromNode(root)
	doc.Find("i18n, [i18n]").Each(func(_ int, element *goquery.Selection) {
		element.RemoveAttr("i18n")
		msgContext, _ := element.Attr("i18n-context")
		element.RemoveAttr("i18n-context")
		if language != SourceLanguage {
			innerHTML, _ := element.Html()
			innerHTML = html.UnescapeString(innerHTML)
			innerHTML = strings.TrimSpace(innerHTML)
			if innerHTML == "" {
				return
			}
			if translated := g.Translations[language].GetTranslation(innerHTML); translated != "" {
				element.SetHtml(translated)
			} else {
				LogDebug("adding missing message %q", innerHTML)
				g.Translations[language].missingMessages = append(g.Translations[language].missingMessages, po.Message{
					MsgId:      innerHTML,
					MsgContext: msgContext,
				})
			}
		}
	})
	htmlString, _ := doc.Html()
	htmlString = strings.ReplaceAll(htmlString, "<i18n>", "")
	htmlString = strings.ReplaceAll(htmlString, "</i18n>", "")
	return gohtml.Format(htmlString)
}

// LoadTranslations reads from i18n/fr.po to load translations
func LoadTranslations() (Translations, error) {
	translations := make(Translations)
	for _, languageCode := range []string{"fr", "en"} {
		translationsFilepath := fmt.Sprintf("i18n/%s.po", languageCode)
		Status(StepLoadTranslations, ProgressDetails{
			File: translationsFilepath,
		})
		poFile, err := po.LoadFile(translationsFilepath)
		if err != nil {
			fmt.Printf("Couldn't load translations for %s (%s): %s\n", languageCode, translationsFilepath, err.Error())
			poFile = &po.File{}
		}

		translations[languageCode] = &TranslationsOneLang{
			poFile:          *poFile,
			seenMsgIds:      mapset.NewSet(),
			missingMessages: make([]po.Message, 0),
			language:        languageCode,
		}
	}
	return translations, nil
}

// SavePO writes the .po file to the disk, with its potential modifications
// It removes duplicate msgids beforehand
func (t TranslationsOneLang) SavePO() {
	// TODO: sort file after saving, (po.File).Save is not stable... (creates unecessary diffs in git)
	// Remove unused messages with empty msgstrs
	uselessRemoved := make([]po.Message, 0)
	for _, msg := range t.poFile.Messages {
		if !t.seenMsgIds.Contains(msg.MsgId) && msg.MsgStr == "" {
			t.seenMsgIds.Remove(msg.MsgId)
			continue
		}
		uselessRemoved = append(uselessRemoved, msg)
	}
	t.poFile.Messages = uselessRemoved
	// Add missing msgids
	t.poFile.Messages = append(t.poFile.Messages, t.missingMessages...)
	// Remove duplicate msgids
	dedupedMessages := make([]po.Message, 0)
	for _, msg := range t.poFile.Messages {
		var isDupe bool
		for _, msg2 := range dedupedMessages {
			if msg.MsgId == msg2.MsgId && msg.MsgContext == msg2.MsgContext {
				isDupe = true
			}
		}
		if !isDupe {
			dedupedMessages = append(dedupedMessages, msg)
		}
	}
	t.poFile.Messages = dedupedMessages
	// Sort them to guarantee a stable write
	sort.Sort(ByMsgId(t.poFile.Messages))
	t.poFile.Save(fmt.Sprintf("i18n/%s.po", t.language))
}

// ByMsgId implement sorting gettext messages by their msgid
type ByMsgId []po.Message

func (b ByMsgId) Len() int {
	return len(b)
}

func (b ByMsgId) Less(i, j int) bool {
	return b[i].MsgId < b[j].MsgId
}

func (b ByMsgId) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// GetTranslation returns the msgstr corresponding to msgid from the .po file
// If not found, it returns the empty string
func (t TranslationsOneLang) GetTranslation(msgid string) string {
	t.seenMsgIds.Add(msgid)
	for _, message := range t.poFile.Messages {
		if message.MsgId == msgid && message.MsgStr != "" {
			return message.MsgStr
		}
	}
	return ""
}
