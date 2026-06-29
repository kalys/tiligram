package bot

import (
	"testing"

	"github.com/blevesearch/bleve/search"
)

func makeHits(keywords ...string) search.DocumentMatchCollection {
	hits := make(search.DocumentMatchCollection, len(keywords))
	for i, kw := range keywords {
		hits[i] = &search.DocumentMatch{
			ID:     kw,
			Fields: map[string]interface{}{"Keyword": kw},
		}
	}
	return hits
}

func TestButtons_empty(t *testing.T) {
	rows := buttons(makeHits(), "q")
	if len(rows) != 1 || len(rows[0]) != 0 {
		t.Errorf("empty hits: want [[]], got %v", rows)
	}
}

func TestButtons_fewerThanBatchSize(t *testing.T) {
	rows := buttons(makeHits("a", "b"), "q")
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 2 {
		t.Errorf("want 2 buttons in row, got %d", len(rows[0]))
	}
}

func TestButtons_exactlyBatchSize(t *testing.T) {
	rows := buttons(makeHits("a", "b", "c"), "q")
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if len(rows[0]) != 3 {
		t.Errorf("want 3 buttons in row, got %d", len(rows[0]))
	}
}

func TestButtons_moreThanBatchSize(t *testing.T) {
	rows := buttons(makeHits("a", "b", "c", "d"), "q")
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if len(rows[0]) != 3 {
		t.Errorf("want 3 buttons in first row, got %d", len(rows[0]))
	}
	if len(rows[1]) != 1 {
		t.Errorf("want 1 button in second row, got %d", len(rows[1]))
	}
}

func TestButtons_uniqueContainsIDAndQuery(t *testing.T) {
	rows := buttons(makeHits("word1"), "кошка")
	btn := rows[0][0]
	want := "word1,кошка"
	if btn.Unique != want {
		t.Errorf("Unique = %q, want %q", btn.Unique, want)
	}
	if btn.Text != "word1" {
		t.Errorf("Text = %q, want %q", btn.Text, "word1")
	}
}
