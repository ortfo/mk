package ortfomk

import (
	"fmt"
	"net/http"
	"regexp"

	mapset "github.com/deckarep/golang-set"
)

// IsLinkDead returns (true, nil) if the given link is dead (i.e. rotten).
// An non-nil error is returned if an error occured while trying to make a GET request to the link (e.g. no Internet connection).
func IsLinkDead(link string) (bool, error) {
	resp, err := http.Get(link)
	if err != nil {
		return false, fmt.Errorf("while opening %s: %w", link, err)
	}

	return resp.StatusCode >= 400, nil
}

// AllLinks returns set of all HTTP links in a given document (so no duplicates).
func AllLinks(document string) (links mapset.Set) {
	links = mapset.NewSet()
	linksSlice := regexp.MustCompile(`\bhttps?://\S+\b`).FindAllString(document, -1)
	if len(linksSlice) == 0 {
		return
	}

	for _, link := range linksSlice {
		links.Add(link)
	}

	return
}

func Deadlinks(links mapset.Set) (deadlinks []string, err error) {
	for _, link := range links.ToSlice() {
		dead, err := IsLinkDead(link.(string))
		if err != nil {
			return deadlinks, err
		}

		if dead {
			deadlinks = append(deadlinks, link.(string))
		}
	}
	return
}
