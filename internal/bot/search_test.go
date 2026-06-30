package bot

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
)

// -- firstHit --

func TestFirstHit_empty(t *testing.T) {
	sr := &bleve.SearchResult{Hits: search.DocumentMatchCollection{}}
	if got := firstHit(sr); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestFirstHit_singleHit(t *testing.T) {
	sr := &bleve.SearchResult{
		Hits: search.DocumentMatchCollection{
			{Fields: map[string]interface{}{"Keyword": "кошка", "Value": "мышык"}},
		},
	}
	got := firstHit(sr)
	want := "<b>кошка</b>\nмышык"
	if got != want {
		t.Errorf("firstHit() = %q, want %q", got, want)
	}
}

func TestFirstHit_stripsNbsp(t *testing.T) {
	sr := &bleve.SearchResult{
		Hits: search.DocumentMatchCollection{
			{Fields: map[string]interface{}{"Keyword": "тест", "Value": "te&nbsp;st"}},
		},
	}
	got := firstHit(sr)
	want := "<b>тест</b>\ntest"
	if got != want {
		t.Errorf("firstHit() = %q, want %q", got, want)
	}
}

func TestFirstHit_multipleHitsUsesFirst(t *testing.T) {
	sr := &bleve.SearchResult{
		Hits: search.DocumentMatchCollection{
			{Fields: map[string]interface{}{"Keyword": "первый", "Value": "биринчи"}},
			{Fields: map[string]interface{}{"Keyword": "второй", "Value": "экинчи"}},
		},
	}
	got := firstHit(sr)
	want := "<b>первый</b>\nбиринчи"
	if got != want {
		t.Errorf("firstHit() = %q, want %q", got, want)
	}
}
