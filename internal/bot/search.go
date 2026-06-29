package bot

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
)

type Searcher interface {
	Search(req *bleve.SearchRequest) (*bleve.SearchResult, error)
}

func searchByTerm(index Searcher, term string) (*bleve.SearchResult, error) {
	req := bleve.NewSearchRequest(bleve.NewQueryStringQuery(term))
	req.Fields = []string{"Keyword", "Value"}
	return index.Search(req)
}

func searchByDocID(index Searcher, id string) (*bleve.SearchResult, error) {
	req := bleve.NewSearchRequest(bleve.NewDocIDQuery([]string{id}))
	req.Fields = []string{"Keyword", "Value"}
	return index.Search(req)
}

func buildBoostedQuery(term string) string {
	escaped := escapeSpecialChars(term)
	return fmt.Sprintf("Keyword:%s^5 Value:%s", escaped, escaped)
}

func escapeSpecialChars(term string) string {
	specialChars := "+-=&|><!(){}[]^\"~*?:\\/"
	var escaped strings.Builder
	for _, char := range term {
		if strings.ContainsRune(specialChars, char) {
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(char)
	}
	return escaped.String()
}

func firstHit(sr *bleve.SearchResult) string {
	if len(sr.Hits) == 0 {
		return ""
	}
	hitValue := strings.Replace(sr.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
	hitKeyword := "<b>" + sr.Hits[0].Fields["Keyword"].(string) + "</b>"
	return hitKeyword + "\n" + hitValue
}
