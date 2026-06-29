package bot

import (
	"strings"

	"github.com/blevesearch/bleve/search"
	tb "gopkg.in/telebot.v3"
)

func buttons(hits search.DocumentMatchCollection, query string) [][]tb.InlineButton {
	inlineButtons := make([]tb.InlineButton, len(hits))
	for i, hit := range hits {
		inlineButtons[i] = tb.InlineButton{
			Unique: strings.Join([]string{hit.ID, query}, ","),
			Text:   hit.Fields["Keyword"].(string),
		}
	}

	batchSize := 3
	var batches [][]tb.InlineButton
	for batchSize < len(inlineButtons) {
		inlineButtons, batches = inlineButtons[batchSize:], append(batches, inlineButtons[0:batchSize:batchSize])
	}
	batches = append(batches, inlineButtons)

	return batches
}
