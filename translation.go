package ortfomk

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	po "github.com/chai2010/gettext-go/po"
	mapset "github.com/deckarep/golang-set"
	"golang.org/x/net/html"
)

const SourceLanguage = "en"

// This is the ugliest delimiter pair I could come up with. The idea is to prevent conflicts with any potential input.
const (
	TranslationStringDelimiterOpen  = "[=[=[={{{"
	TranslationStringDelimiterClose = "}}}=]=]=]"
)

// TranslationsOneLang holds both the gettext catalog from the .mo file
// and a po file object used to update the .po file (e.g. when discovering new translatable strings)
type TranslationsOneLang struct {
	poFile          po.File
	seenMessages    mapset.Set
	missingMessages []po.Message
	language        string
}

func (t TranslationsOneLang) WriteUnusedMessages() error {
	to := fmt.Sprintf("i18n/%s-unused-messages.yaml", t.language)
	ioutil.WriteFile(to, []byte("# Generated at "+time.Now().String()+"\n"), 0644)
	file, err := os.OpenFile(to, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, message := range t.poFile.Messages {
		if !t.seenMessages.Contains(message.MsgId + message.MsgContext) {
			if message.MsgContext != "" {
				_, err = file.WriteString(fmt.Sprintf("- {msgid: %q, msgctxt: %q}\n", message.MsgId, message.MsgContext))
			} else {
				_, err = file.WriteString(fmt.Sprintf("- %q\n", message.MsgId))
			}
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
			if translated := g.Translations[language].GetTranslation(innerHTML, msgContext); translated != "" {
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
	return htmlString
}

type translationString struct {
	Value   string
	Args    []interface{}
	Context string
}

// TranslateTranslationStrings applies translations to a string containing translation strings as substrings.
// See TranslationStringDelimiterOpen, TranslationStringDelimiterClose. If those two are respectively [ and ],
// this function replaces
//
//	you have [{value: "%d friends", args: [8]}] online
//
// with, given that t.GetTranslation("%d friends") returns "%d amis":
//
//	you have 8 amis
//
// TODO: use ICU message syntax instead.
func (t TranslationsOneLang) TranslateTranslationStrings(content string) string {
	startsAt := strings.Index(content, TranslationStringDelimiterOpen)
	if startsAt < 0 {
		return content
	}
	endsAt := strings.Index(content, TranslationStringDelimiterClose)
	if endsAt < 0 {
		return content
	}
	innerJSON := html.UnescapeString(content[startsAt+len(TranslationStringDelimiterOpen) : endsAt])
	translation := translationString{}
	err := json.Unmarshal([]byte(innerJSON), &translation)
	if err != nil {
		LogFatal("couldn't parse JSON translation string %q: %s", innerJSON, err)
		panic(err)
	}

	LogDebug("translating dynamic message %s", innerJSON)
	return content[:startsAt] + fmt.Sprintf(t.GetTranslation(translation.Value, translation.Context), translation.Args...) + t.TranslateTranslationStrings(content[endsAt+len(TranslationStringDelimiterClose):])
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
			seenMessages:    mapset.NewSet(),
			missingMessages: make([]po.Message, 0),
			language:        languageCode,
		}
	}
	return translations, nil
}

// SavePO writes the .po file to the disk, with its potential modifications
// It removes duplicate messages beforehand
func (t TranslationsOneLang) SavePO() {
	// TODO: sort file after saving, (po.File).Save is not stable... (creates unecessary diffs in git)
	// Remove unused messages with empty msgstrs
	uselessRemoved := make([]po.Message, 0)
	for _, msg := range t.poFile.Messages {
		if !t.seenMessages.Contains(msg.MsgId+msg.MsgContext) && msg.MsgStr == "" {
			t.seenMessages.Remove(msg.MsgId + msg.MsgContext)
			continue
		}
		uselessRemoved = append(uselessRemoved, msg)
	}
	t.poFile.Messages = uselessRemoved
	// Add missing messages
	t.poFile.Messages = append(t.poFile.Messages, t.missingMessages...)
	// Remove duplicate messages
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
	sort.Sort(ByMsgIdAndCtx(t.poFile.Messages))
	t.poFile.Save(fmt.Sprintf("i18n/%s.po", t.language))
}

// ByMsgIdAndCtx implement sorting gettext messages by their msgid+msgctxt
type ByMsgIdAndCtx []po.Message

func (b ByMsgIdAndCtx) Len() int {
	return len(b)
}

func (b ByMsgIdAndCtx) Less(i, j int) bool {
	return b[i].MsgId < b[j].MsgId || (b[i].MsgId == b[j].MsgId && b[i].MsgContext < b[j].MsgContext)
}

func (b ByMsgIdAndCtx) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

// GetTranslation returns the msgstr corresponding to msgid and msgctxt from the .po file
// If not found, it returns the empty string
func (t TranslationsOneLang) GetTranslation(msgid string, msgctxt string) string {
	t.seenMessages.Add(msgid + msgctxt)
	for _, message := range t.poFile.Messages {
		if message.MsgId == msgid && message.MsgStr != "" && message.MsgContext == msgctxt {
			return message.MsgStr
		}
	}
	return ""
}
