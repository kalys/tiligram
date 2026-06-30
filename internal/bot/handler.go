package bot

import (
	"strconv"
	"strings"

	"github.com/dukex/mixpanel"
	"github.com/getsentry/sentry-go"
	"github.com/kalys/tiligram/internal/search"
	tb "gopkg.in/telebot.v3"
)

const maxTermLength = 200

const responseText = `Как пользоваться:
1) отправляем слово боту, получаем перевод.
2) если бот в группе, то "/tili слово"

Вебсайт: http://tili.kg
Обратная связь по боту: @kalys, kalys@osmonov.com`

type Analytics interface {
	Track(distinctID, eventName string, e *mixpanel.Event) error
}

type Sender interface {
	Send(to tb.Recipient, what interface{}, options ...interface{}) (*tb.Message, error)
	Edit(msg tb.Editable, what interface{}, options ...interface{}) (*tb.Message, error)
	Respond(c *tb.Callback, resp ...*tb.CallbackResponse) error
}

type BotHandler struct {
	index     search.Searcher
	analytics Analytics
	sender    Sender
}

func NewBotHandler(index search.Searcher, analytics Analytics, sender Sender) *BotHandler {
	return &BotHandler{index: index, analytics: analytics, sender: sender}
}

func (h *BotHandler) RegisterHandlers(b *tb.Bot) {
	b.Handle("/help", func(c tb.Context) error {
		_, err := h.sender.Send(c.Sender(), responseText)
		return err
	})
	b.Handle("/start", func(c tb.Context) error {
		_, err := h.sender.Send(c.Sender(), responseText)
		return err
	})

	b.Handle("/tili", func(c tb.Context) error {
		recoverAndCapture(func() { h.handleTranslate(c.Message().Payload, c.Message()) })
		return nil
	})

	b.Handle(tb.OnCallback, func(c tb.Context) error {
		recoverAndCapture(func() { h.handleCallback(c.Callback()) })
		return nil
	})

	b.Handle(tb.OnText, func(c tb.Context) error {
		if c.Message().Chat.Type != tb.ChatPrivate {
			return nil
		}
		recoverAndCapture(func() { h.handleTranslate(c.Message().Text, c.Message()) })
		return nil
	})
}

func recoverAndCapture(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			sentry.CurrentHub().Recover(r)
		}
	}()
	fn()
}

func (h *BotHandler) handleTranslate(term string, m *tb.Message) {
	if len([]rune(term)) > maxTermLength {
		h.sender.Send(m.Chat, "Запрос слишком длинный")
		return
	}

	searchResult, err := search.ByTerm(h.index, search.BuildBoostedQuery(term))
	if err != nil {
		sentry.WithScope(func(scope *sentry.Scope) {
			scope.SetTag("term", term)
			sentry.CaptureException(err)
		})
		return
	}

	messageText := firstHit(searchResult)

	if messageText == "" {
		h.analytics.Track(strconv.FormatInt(m.Sender.ID, 10), "Not found", &mixpanel.Event{
			Properties: map[string]interface{}{"term": term},
		})
		h.sender.Send(m.Chat, "Перевод не найден")
		return
	}

	h.analytics.Track(strconv.FormatInt(m.Sender.ID, 10), "Translate", &mixpanel.Event{
		Properties: map[string]interface{}{"term": term},
	})

	h.sender.Send(m.Chat, messageText,
		&tb.SendOptions{ParseMode: tb.ModeHTML},
		&tb.ReplyMarkup{InlineKeyboard: buttons(searchResult.Hits, term)})
}

func (h *BotHandler) handleCallback(c *tb.Callback) {
	if len([]rune(c.Data)) > maxTermLength {
		h.sender.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Запрос слишком длинный",
		})
		return
	}

	parts := strings.SplitN(c.Data, ",", 2)
	wordID := strings.TrimSpace(parts[0])
	term := parts[1]

	docResult, err := search.ByDocID(h.index, wordID)
	if err != nil {
		sentry.CaptureException(err)
		h.sender.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте выбрать другой перевод",
		})
		return
	}

	termResult, err := search.ByTerm(h.index, search.BuildBoostedQuery(term))
	if err != nil {
		sentry.CaptureException(err)
		h.sender.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте ввести другое ключевое слово",
		})
		return
	}

	messageText := firstHit(docResult)
	if messageText == "" {
		h.sender.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте выбрать другой перевод",
		})
		return
	}

	h.analytics.Track(strconv.FormatInt(c.Sender.ID, 10), "Selected", &mixpanel.Event{
		Properties: map[string]interface{}{"keyword": docResult.Hits[0].Fields["Keyword"].(string)},
	})
	h.sender.Edit(c.Message, messageText,
		&tb.SendOptions{ParseMode: tb.ModeHTML},
		&tb.ReplyMarkup{InlineKeyboard: buttons(termResult.Hits, term)})

	h.sender.Respond(c, &tb.CallbackResponse{CallbackID: c.ID})
}
