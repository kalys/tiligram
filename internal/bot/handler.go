package bot

import (
	"strconv"
	"strings"

	"github.com/dukex/mixpanel"
	"github.com/getsentry/raven-go"
	tb "gopkg.in/tucnak/telebot.v2"
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
	index     Searcher
	analytics Analytics
	sender    Sender
}

func NewBotHandler(index Searcher, analytics Analytics, sender Sender) *BotHandler {
	return &BotHandler{index: index, analytics: analytics, sender: sender}
}

func (h *BotHandler) RegisterHandlers(b *tb.Bot) {
	b.Handle("/help", func(m *tb.Message) { h.sender.Send(m.Sender, responseText) })
	b.Handle("/start", func(m *tb.Message) { h.sender.Send(m.Sender, responseText) })

	b.Handle("/tili", func(m *tb.Message) {
		raven.CapturePanic(func() { h.handleTranslate(m.Payload, m) }, nil)
	})

	b.Handle(tb.OnCallback, func(c *tb.Callback) {
		raven.CapturePanic(func() { h.handleCallback(c) }, nil)
	})

	b.Handle(tb.OnText, func(m *tb.Message) {
		if m.Chat.Type != tb.ChatPrivate {
			return
		}
		raven.CapturePanic(func() { h.handleTranslate(m.Text, m) }, nil)
	})
}

func (h *BotHandler) handleTranslate(term string, m *tb.Message) {
	if len([]rune(term)) > maxTermLength {
		h.sender.Send(m.Chat, "Запрос слишком длинный")
		return
	}

	searchResult, err := searchByTerm(h.index, buildBoostedQuery(term))
	if err != nil {
		raven.CaptureError(err, map[string]string{"term": term})
		return
	}

	messageText := firstHit(searchResult)

	if messageText == "" {
		h.analytics.Track(strconv.Itoa(m.Sender.ID), "Not found", &mixpanel.Event{
			Properties: map[string]interface{}{"term": term},
		})
		h.sender.Send(m.Chat, "Перевод не найден")
		return
	}

	h.analytics.Track(strconv.Itoa(m.Sender.ID), "Translate", &mixpanel.Event{
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

	docResult, err := searchByDocID(h.index, wordID)
	if err != nil {
		raven.CaptureError(err, map[string]string{"term": term})
		h.sender.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте выбрать другой перевод",
		})
		return
	}

	termResult, err := searchByTerm(h.index, buildBoostedQuery(term))
	if err != nil {
		raven.CaptureError(err, map[string]string{"term": term})
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

	h.analytics.Track(strconv.Itoa(c.Sender.ID), "Selected", &mixpanel.Event{
		Properties: map[string]interface{}{"keyword": docResult.Hits[0].Fields["Keyword"].(string)},
	})
	h.sender.Edit(c.Message, messageText,
		&tb.SendOptions{ParseMode: tb.ModeHTML},
		&tb.ReplyMarkup{InlineKeyboard: buttons(termResult.Hits, term)})

	h.sender.Respond(c, &tb.CallbackResponse{CallbackID: c.ID})
}
