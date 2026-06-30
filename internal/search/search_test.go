package search

import (
	"path/filepath"
	"testing"

	"github.com/blevesearch/bleve"
)

// -- EscapeSpecialChars --

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
		got := EscapeSpecialChars(tc.input)
		if got != tc.want {
			t.Errorf("EscapeSpecialChars(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// -- BuildBoostedQuery --

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
		got := BuildBoostedQuery(tc.term)
		if got != tc.want {
			t.Errorf("BuildBoostedQuery(%q) = %q, want %q", tc.term, got, tc.want)
		}
	}
}

// -- integration: ByTerm / ByDocID --

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

func TestByTerm_found(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"1": {Keyword: "собака", Value: "ит"},
		"2": {Keyword: "кот", Value: "мышык"},
	})

	result, err := ByTerm(idx, "собака")
	if err != nil {
		t.Fatalf("ByTerm: %v", err)
	}
	if len(result.Hits) == 0 {
		t.Fatal("expected at least one hit, got none")
	}
	if result.Hits[0].ID != "1" {
		t.Errorf("top hit ID = %q, want %q", result.Hits[0].ID, "1")
	}
}

func TestByTerm_notFound(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"1": {Keyword: "собака", Value: "ит"},
	})

	result, err := ByTerm(idx, "nonexistentterm99999")
	if err != nil {
		t.Fatalf("ByTerm: %v", err)
	}
	if len(result.Hits) != 0 {
		t.Errorf("expected no hits, got %d", len(result.Hits))
	}
}

func TestByDocID_found(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"doc42": {Keyword: "zeppelin", Value: "airship"},
	})

	result, err := ByDocID(idx, "doc42")
	if err != nil {
		t.Fatalf("ByDocID: %v", err)
	}
	if len(result.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(result.Hits))
	}
	if result.Hits[0].ID != "doc42" {
		t.Errorf("hit ID = %q, want %q", result.Hits[0].ID, "doc42")
	}
}

func TestByDocID_notFound(t *testing.T) {
	idx := newTestIndex(t, map[string]testRecord{
		"doc1": {Keyword: "test", Value: "value"},
	})

	result, err := ByDocID(idx, "doesnotexist")
	if err != nil {
		t.Fatalf("ByDocID: %v", err)
	}
	if len(result.Hits) != 0 {
		t.Errorf("expected no hits, got %d", len(result.Hits))
	}
}
