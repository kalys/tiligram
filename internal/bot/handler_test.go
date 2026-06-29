package bot

import (
	"errors"
	"strings"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/dukex/mixpanel"
	tb "gopkg.in/telebot.v3"
)

// -- mocks --

type mockSearcher struct {
	results []*bleve.SearchResult
	errors  []error
	idx     int
}

func (m *mockSearcher) Search(_ *bleve.SearchRequest) (*bleve.SearchResult, error) {
	i := m.idx
	m.idx++
	if i < len(m.errors) && m.errors[i] != nil {
		return nil, m.errors[i]
	}
	if i < len(m.results) {
		return m.results[i], nil
	}
	return &bleve.SearchResult{}, nil
}

type trackCall struct {
	distinctID string
	eventName  string
}

type mockAnalytics struct {
	calls []trackCall
}

func (m *mockAnalytics) Track(distinctID, eventName string, _ *mixpanel.Event) error {
	m.calls = append(m.calls, trackCall{distinctID, eventName})
	return nil
}

type sendCall struct{ what interface{} }
type respondCall struct{ text string }
type editCall struct{ what interface{} }

type mockSender struct {
	sends    []sendCall
	responds []respondCall
	edits    []editCall
}

func (m *mockSender) Send(_ tb.Recipient, what interface{}, _ ...interface{}) (*tb.Message, error) {
	m.sends = append(m.sends, sendCall{what})
	return nil, nil
}

func (m *mockSender) Edit(_ tb.Editable, what interface{}, _ ...interface{}) (*tb.Message, error) {
	m.edits = append(m.edits, editCall{what})
	return nil, nil
}

func (m *mockSender) Respond(_ *tb.Callback, resp ...*tb.CallbackResponse) error {
	text := ""
	if len(resp) > 0 {
		text = resp[0].Text
	}
	m.responds = append(m.responds, respondCall{text})
	return nil
}

// -- helpers --

func newHandler(s *mockSearcher, a *mockAnalytics, sender *mockSender) *BotHandler {
	return NewBotHandler(s, a, sender)
}

func testMessage(userID int64) *tb.Message {
	return &tb.Message{
		Chat:   &tb.Chat{ID: 1, Type: tb.ChatPrivate},
		Sender: &tb.User{ID: userID},
	}
}

func testCallback(data string, userID int64) *tb.Callback {
	return &tb.Callback{
		ID:     "cb1",
		Data:   data,
		Sender: &tb.User{ID: userID},
		Message: &tb.Message{
			ID:   1,
			Chat: &tb.Chat{ID: 1},
		},
	}
}

func hitWithFields(id, keyword, value string) *search.DocumentMatch {
	return &search.DocumentMatch{
		ID:     id,
		Fields: map[string]interface{}{"Keyword": keyword, "Value": value},
	}
}

func resultWith(hits ...*search.DocumentMatch) *bleve.SearchResult {
	return &bleve.SearchResult{Hits: search.DocumentMatchCollection(hits)}
}

// -- handleTranslate tests --

func TestHandleTranslate_termTooLong(t *testing.T) {
	s, a, sender := &mockSearcher{}, &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleTranslate(strings.Repeat("а", maxTermLength+1), testMessage(1))

	if s.idx != 0 {
		t.Error("search should not be called for oversized term")
	}
	if len(sender.sends) != 1 || sender.sends[0].what != "Запрос слишком длинный" {
		t.Errorf("expected oversized-term message, got %v", sender.sends)
	}
}

func TestHandleTranslate_noResults(t *testing.T) {
	s := &mockSearcher{results: []*bleve.SearchResult{resultWith()}}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleTranslate("кошка", testMessage(42))

	if len(a.calls) != 1 || a.calls[0].eventName != "Not found" {
		t.Errorf("expected 'Not found' event, got %v", a.calls)
	}
	if len(sender.sends) != 1 || sender.sends[0].what != "Перевод не найден" {
		t.Errorf("expected not-found message, got %v", sender.sends)
	}
}

func TestHandleTranslate_found(t *testing.T) {
	hit := hitWithFields("1", "кошка", "мышык")
	s := &mockSearcher{results: []*bleve.SearchResult{resultWith(hit)}}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleTranslate("кошка", testMessage(42))

	if len(a.calls) != 1 || a.calls[0].eventName != "Translate" {
		t.Errorf("expected 'Translate' event, got %v", a.calls)
	}
	if len(sender.sends) != 1 {
		t.Fatalf("expected 1 send, got %d", len(sender.sends))
	}
	if !strings.Contains(sender.sends[0].what.(string), "кошка") {
		t.Errorf("sent message should contain keyword, got %q", sender.sends[0].what)
	}
}

func TestHandleTranslate_searchError(t *testing.T) {
	s := &mockSearcher{errors: []error{errors.New("index failure")}}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleTranslate("кошка", testMessage(1))

	if len(sender.sends) != 0 {
		t.Errorf("expected no sends on search error, got %v", sender.sends)
	}
	if len(a.calls) != 0 {
		t.Errorf("expected no analytics on search error, got %v", a.calls)
	}
}

// -- handleCallback tests --

func TestHandleCallback_dataTooLong(t *testing.T) {
	s, a, sender := &mockSearcher{}, &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	longData := "id," + strings.Repeat("а", maxTermLength)
	h.handleCallback(testCallback(longData, 1))

	if s.idx != 0 {
		t.Error("search should not be called for oversized data")
	}
	if len(sender.responds) != 1 || sender.responds[0].text != "Запрос слишком длинный" {
		t.Errorf("expected oversized-term response, got %v", sender.responds)
	}
}

func TestHandleCallback_docIDSearchError(t *testing.T) {
	s := &mockSearcher{errors: []error{errors.New("db error")}}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleCallback(testCallback("id1,кошка", 1))

	if len(sender.responds) != 1 || !strings.Contains(sender.responds[0].text, "другой перевод") {
		t.Errorf("expected doc-error response, got %v", sender.responds)
	}
}

func TestHandleCallback_termSearchError(t *testing.T) {
	docHit := hitWithFields("id1", "кошка", "мышык")
	s := &mockSearcher{
		results: []*bleve.SearchResult{resultWith(docHit)},
		errors:  []error{nil, errors.New("term search failed")},
	}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleCallback(testCallback("id1,кошка", 1))

	if len(sender.responds) != 1 || !strings.Contains(sender.responds[0].text, "ключевое слово") {
		t.Errorf("expected term-error response, got %v", sender.responds)
	}
}

func TestHandleCallback_emptyDocResult(t *testing.T) {
	s := &mockSearcher{
		results: []*bleve.SearchResult{resultWith(), resultWith()},
	}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleCallback(testCallback("id1,кошка", 1))

	if len(sender.responds) != 1 || !strings.Contains(sender.responds[0].text, "другой перевод") {
		t.Errorf("expected empty-doc response, got %v", sender.responds)
	}
}

func TestHandleCallback_success(t *testing.T) {
	docHit := hitWithFields("id1", "кошка", "мышык")
	termHit := hitWithFields("id1", "кошка", "мышык")
	s := &mockSearcher{
		results: []*bleve.SearchResult{resultWith(docHit), resultWith(termHit)},
	}
	a, sender := &mockAnalytics{}, &mockSender{}
	h := newHandler(s, a, sender)

	h.handleCallback(testCallback("id1,кошка", 42))

	if len(a.calls) != 1 || a.calls[0].eventName != "Selected" {
		t.Errorf("expected 'Selected' event, got %v", a.calls)
	}
	if len(sender.edits) != 1 {
		t.Errorf("expected 1 edit, got %d", len(sender.edits))
	}
	if len(sender.responds) != 1 || sender.responds[0].text != "" {
		t.Errorf("expected empty ack respond, got %v", sender.responds)
	}
}
