package search

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve"
)

type Searcher interface {
	Search(req *bleve.SearchRequest) (*bleve.SearchResult, error)
}

func ByTerm(index Searcher, term string) (*bleve.SearchResult, error) {
	req := bleve.NewSearchRequest(bleve.NewQueryStringQuery(term))
	req.Fields = []string{"Keyword", "Value"}
	return index.Search(req)
}

func ByDocID(index Searcher, id string) (*bleve.SearchResult, error) {
	req := bleve.NewSearchRequest(bleve.NewDocIDQuery([]string{id}))
	req.Fields = []string{"Keyword", "Value"}
	return index.Search(req)
}

func BuildBoostedQuery(term string) string {
	escaped := EscapeSpecialChars(term)
	return fmt.Sprintf("Keyword:%s^5 Value:%s", escaped, escaped)
}

func EscapeSpecialChars(term string) string {
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
