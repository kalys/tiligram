package bot

import (
	"strings"

	"github.com/blevesearch/bleve"
)

func firstHit(sr *bleve.SearchResult) string {
	if len(sr.Hits) == 0 {
		return ""
	}
	hitValue := strings.Replace(sr.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
	hitKeyword := "<b>" + sr.Hits[0].Fields["Keyword"].(string) + "</b>"
	return hitKeyword + "\n" + hitValue
}
