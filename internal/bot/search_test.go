package bot

import (
	"path/filepath"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
)

// -- escapeSpecialChars --

func TestEscapeSpecialChars(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"c+t", `c\+t`},
		{"a-b", `a\-b`},
		{"a&b", `a\&b`},
		{"a|b", `a\|b`},
		{`a"b`, `a\"b`},
		{"a(b)", `a\(b\)`},
		{"a[b]", `a\[b\]`},
		{"a{b}", `a\{b\}`},
		{"a^b", `a\^b`},
		{"a~b", `a\~b`},
		{"a*b", `a\*b`},
		{"a?b", `a\?b`},
		{`a\b`, `a\\b`},
		{"a/b", `a\/b`},
		{"a:b", `a\:b`},
		{"no special chars", "no special chars"},
		{"", ""},
	}

	for _, tc := range cases {
		got := escapeSpecialChars(tc.input)
		if got != tc.want {
			t.Errorf("escapeSpecialChars(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// -- buildBoostedQuery --

func TestBuildBoostedQuery(t *testing.T) {
	cases := []struct {
		term string
		want string
	}{
		{"мышык", `Keyword:мышык^5 Value:мышык`},
		{"c+t", `Keyword:c\+t^5 Value:c\+t`},
		{"", `Keyword:^5 Value:`},
	}

	for _, tc := range cases {
		got := buildBoostedQuery(tc.term)
		if got != tc.want {
			t.Errorf("buildBoostedQuery(%q) = %q, want %q", tc.term, got, tc.want)
		}
	}
}

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

// -- integration: searchByTerm / searchByDocID --

type testRecord struct {
	Keyword string
	Value   string
}

func newTestIndex(t *testing.T, docs map[string]testRecord) bleve.Index {
	t.Helper()
	idx, err := bleve.New(filepath.Join(t.TempDir(), "test.bleve"), bleve.NewIndexMapping())
	if err != nil {
		t.Fatalf("create index: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	for id, doc := range docs {
		if err := idx.Index(id, doc); err != nil {
			t.Fatalf("index %s: %v", id, err)
		}
	}
	return idx
}

func TestSearchByTerm_found(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"1": {Keyword: "собака", Value: "ит"},
		"2": {Keyword: "кот", Value: "мышык"},
	})

	result, err := searchByTerm(idx, "собака")
	if err != nil {
		t.Fatalf("searchByTerm: %v", err)
	}
	if len(result.Hits) == 0 {
		t.Fatal("expected at least one hit, got none")
	}
	if result.Hits[0].ID != "1" {
		t.Errorf("top hit ID = %q, want %q", result.Hits[0].ID, "1")
	}
}

func TestSearchByTerm_notFound(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"1": {Keyword: "собака", Value: "ит"},
	})

	result, err := searchByTerm(idx, "nonexistentterm99999")
	if err != nil {
		t.Fatalf("searchByTerm: %v", err)
	}
	if len(result.Hits) != 0 {
		t.Errorf("expected no hits, got %d", len(result.Hits))
	}
}

func TestSearchByDocID_found(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"doc42": {Keyword: "zeppelin", Value: "airship"},
	})

	result, err := searchByDocID(idx, "doc42")
	if err != nil {
		t.Fatalf("searchByDocID: %v", err)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(result.Hits))
	}
	if result.Hits[0].ID != "doc42" {
		t.Errorf("hit ID = %q, want %q", result.Hits[0].ID, "doc42")
	}
}

func TestSearchByDocID_notFound(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"doc1": {Keyword: "test", Value: "value"},
	})

	result, err := searchByDocID(idx, "doesnotexist")
	if err != nil {
		t.Fatalf("searchByDocID: %v", err)
	}
	if len(result.Hits) != 0 {
		t.Errorf("expected no hits, got %d", len(result.Hits))
	}
}
