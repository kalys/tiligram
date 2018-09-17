package main

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/lang/ru"
	"github.com/blevesearch/bleve/mapping"
)

func buildIndexMapping() (mapping.IndexMapping, error) {

	// a generic reusable mapping for russian text
	keywordTextFieldMapping := bleve.NewTextFieldMapping()
	keywordTextFieldMapping.Analyzer = ru.AnalyzerName
	valueTextFieldMapping := bleve.NewTextFieldMapping()
	valueTextFieldMapping.Analyzer = ru.AnalyzerName

	wordMapping := bleve.NewDocumentMapping()

	// keyword
	wordMapping.AddFieldMappingsAt("keyword", keywordTextFieldMapping)

	// description
	wordMapping.AddFieldMappingsAt("value", valueTextFieldMapping)

	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("word", wordMapping)

	indexMapping.TypeField = "Type"
	indexMapping.DefaultAnalyzer = "ru"

	return indexMapping, nil
}
