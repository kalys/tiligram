package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/getsentry/sentry-go"
	"github.com/kalys/tiligram/internal/search"
)

const maxTermLength = 200
const defaultLimit = 10
const maxLimit = 50

type Handler struct {
	index search.Searcher
}

func NewHandler(index search.Searcher) *Handler {
	return &Handler{index: index}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/search", h.Search)
}

type hitJSON struct {
	ID      string `json:"id"`
	Keyword string `json:"keyword"`
	Value   string `json:"value"`
}

type searchResponse struct {
	Term string    `json:"term"`
	Hits []hitJSON `json:"hits"`
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	term := r.URL.Query().Get("q")
	if term == "" {
		http.Error(w, "q is required", http.StatusBadRequest)
		return
	}
	if len([]rune(term)) > maxTermLength {
		http.Error(w, "q is too long", http.StatusBadRequest)
		return
	}

	limit := defaultLimit
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	req := bleve.NewSearchRequest(bleve.NewQueryStringQuery(search.BuildBoostedQuery(term)))
	req.Fields = []string{"Keyword", "Value"}
	req.Size = limit

	result, err := h.index.Search(req)
	if err != nil {
		sentry.CaptureException(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hits := make([]hitJSON, 0, len(result.Hits))
	for i, match := range result.Hits {
		if i >= limit {
			break
		}
		value := strings.ReplaceAll(match.Fields["Value"].(string), "&nbsp;", "")
		hits = append(hits, hitJSON{
			ID:      match.ID,
			Keyword: match.Fields["Keyword"].(string),
			Value:   value,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(searchResponse{Term: term, Hits: hits})
}
