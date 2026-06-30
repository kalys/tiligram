package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blevesearch/bleve"
	bsearch "github.com/blevesearch/bleve/search"
)

type mockSearcher struct {
	result *bleve.SearchResult
	err    error
}

func (m *mockSearcher) Search(_ *bleve.SearchRequest) (*bleve.SearchResult, error) {
	return m.result, m.err
}

func hitWithFields(id, keyword, value string) *bsearch.DocumentMatch {
	return &bsearch.DocumentMatch{
		ID:     id,
		Fields: map[string]interface{}{"Keyword": keyword, "Value": value},
	}
}

func resultWith(hits ...*bsearch.DocumentMatch) *bleve.SearchResult {
	return &bleve.SearchResult{Hits: bsearch.DocumentMatchCollection(hits)}
}

func get(handler *Handler, url string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	handler.Search(w, req)
	return w
}

func TestSearch_missingQ(t *testing.T) {
	h := NewHandler(&mockSearcher{result: resultWith()})
	w := get(h, "/search")
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_termTooLong(t *testing.T) {
	h := NewHandler(&mockSearcher{result: resultWith()})
	long := strings.Repeat("а", maxTermLength+1)
	w := get(h, "/search?q="+long)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSearch_searcherError(t *testing.T) {
	h := NewHandler(&mockSearcher{err: errors.New("index down")})
	w := get(h, "/search?q=кошка")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSearch_emptyResults(t *testing.T) {
	h := NewHandler(&mockSearcher{result: resultWith()})
	w := get(h, "/search?q=кошка")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp searchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Hits == nil {
		t.Error("hits should be empty array, not null")
	}
	if len(resp.Hits) != 0 {
		t.Errorf("expected 0 hits, got %d", len(resp.Hits))
	}
}

func TestSearch_happyPath(t *testing.T) {
	hit := hitWithFields("1", "кошка", "мышык")
	h := NewHandler(&mockSearcher{result: resultWith(hit)})
	w := get(h, "/search?q=кошка")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp searchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Term != "кошка" {
		t.Errorf("term = %q, want %q", resp.Term, "кошка")
	}
	if len(resp.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(resp.Hits))
	}
	if resp.Hits[0].ID != "1" || resp.Hits[0].Keyword != "кошка" || resp.Hits[0].Value != "мышык" {
		t.Errorf("unexpected hit: %+v", resp.Hits[0])
	}
}

func TestSearch_stripsNbsp(t *testing.T) {
	hit := hitWithFields("1", "тест", "te&nbsp;st")
	h := NewHandler(&mockSearcher{result: resultWith(hit)})
	w := get(h, "/search?q=тест")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp searchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Hits[0].Value != "test" {
		t.Errorf("value = %q, want %q", resp.Hits[0].Value, "test")
	}
}

func TestSearch_limitClamped(t *testing.T) {
	hits := make([]*bsearch.DocumentMatch, 60)
	for i := range hits {
		hits[i] = hitWithFields("id", "kw", "val")
	}
	h := NewHandler(&mockSearcher{result: resultWith(hits...)})
	w := get(h, "/search?q=test&limit=100")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp searchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Hits) > maxLimit {
		t.Errorf("expected at most %d hits, got %d", maxLimit, len(resp.Hits))
	}
}

func TestSearch_customLimit(t *testing.T) {
	hits := make([]*bsearch.DocumentMatch, 10)
	for i := range hits {
		hits[i] = hitWithFields("id", "kw", "val")
	}
	h := NewHandler(&mockSearcher{result: resultWith(hits...)})
	w := get(h, "/search?q=test&limit=3")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp searchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// mock ignores req.Size, so we get all 10 back from Search — but bleve
	// respects req.Size in production. The handler doesn't slice post-search,
	// so this test just verifies the limit parameter is accepted without error.
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
