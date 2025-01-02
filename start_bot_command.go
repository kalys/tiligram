package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/davecgh/go-spew/spew"
	"github.com/dukex/mixpanel"
	"github.com/enbritely/heartbeat-golang"
	"github.com/getsentry/raven-go"
	tb "gopkg.in/tucnak/telebot.v2"
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
)

const responseText = `Как пользоваться:
1) отправляем слово боту, получаем перевод.
2) если бот в группе, то "/tili слово"

Вебсайт: http://tili.kg
Обратная связь по боту: @kalys, kalys@osmonov.com`

var StartBotCommand = cli.Command{
	Name:  "start-bot",
	Usage: "Start Telegram bot",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "bot-token",
			Usage: "Token for Telegram API",
		},
		&cli.StringFlag{
			Name:  "index-path",
			Usage: "Path where index is stored",
			Value: "bleve.search",
		},
		&cli.StringFlag{
			Name:  "raven-dsn",
			Usage: "DSN for sentry",
			Value: "some-dsn",
		},
		&cli.StringFlag{
			Name:  "mixpanel-token",
			Usage: "Mixpanel token",
			Value: "some-token",
		},
	},
	Action: func(c *cli.Context) error {
		go heartbeat.RunHeartbeatService(":10101")

		spew.Dump()
		if err := raven.SetDSN(c.String("raven-dsn")); err != nil {
			return err
		}

		mixpanelClient := mixpanel.New(c.String("mixpanel-token"), "")
		index, err := bleve.Open(c.String("index-path"))
		if err != nil {
			return err
		}

		b, err := tb.NewBot(tb.Settings{
			Token:  c.String("bot-token"),
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			return err
		}

		b.Handle("/help", func(m *tb.Message) { b.Send(m.Sender, responseText) })
		b.Handle("/start", func(m *tb.Message) { b.Send(m.Sender, responseText) })

		b.Handle("/tili", func(m *tb.Message) {
			term := m.Payload
			raven.CapturePanic(func() { handleTranslate(mixpanelClient, index, term, m, b) }, nil)
		})

		b.Handle(tb.OnCallback, func(c *tb.Callback) {
			raven.CapturePanic(func() {
				handleCallback(b, c, index, mixpanelClient)
			}, nil)
		})

		b.Handle(tb.OnText, func(m *tb.Message) {
			if m.Chat.Type != tb.ChatPrivate {
				return
			}

			term := m.Text
			raven.CapturePanic(func() { handleTranslate(mixpanelClient, index, term, m, b) }, nil)
		})

		b.Start()

		return nil
	},
}

func buttons(hits search.DocumentMatchCollection, query string) [][]tb.InlineButton {
	inlineButtons := make([]tb.InlineButton, len(hits))

	for index, hit := range hits {
		uniqueString := strings.Join([]string{hit.ID, query}, ",")
		inlineBtn := tb.InlineButton{
			Unique: uniqueString,
			Text:   hit.Fields["Keyword"].(string),
		}

		inlineButtons[index] = inlineBtn
	}

	batchSize := 3
	var batches [][]tb.InlineButton

	for batchSize < len(inlineButtons) {
		inlineButtons, batches = inlineButtons[batchSize:], append(batches, inlineButtons[0:batchSize:batchSize])
	}
	batches = append(batches, inlineButtons)

	return batches
}

func searchFunction(index bleve.Index, term string) (*bleve.SearchResult, error) {
	query := bleve.NewQueryStringQuery(term)

	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"Keyword", "Value"}
	return index.Search(searchRequest)
}

func firstHit(sr *bleve.SearchResult) string {
	if len(sr.Hits) == 0 {
		return ""
	}
	hitValue := strings.Replace(sr.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
	hitKeyword := strings.Join([]string{"<b>", sr.Hits[0].Fields["Keyword"].(string), "</b>"}, "")

	return strings.Join([]string{hitKeyword, hitValue}, "\n")
}

func escapeSpecialChars(term string) string {
	// List of special characters to escape
	specialChars := "+-=&|><!(){}[]^\"~*?:\\/"
	var escaped strings.Builder

	for _, char := range term {
		if strings.ContainsRune(specialChars, char) {
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(char)
	}

	return escaped.String()
}

func handleTranslate(mixpanelClient mixpanel.Mixpanel, index bleve.Index, term string, m *tb.Message, b *tb.Bot) {
	escapedTerm := escapeSpecialChars(term)
	queryString := fmt.Sprintf("Keyword:%s^5 Value:%s", escapedTerm, escapedTerm)

	searchResult, err := searchFunction(index, queryString)
	if err != nil {
		raven.CaptureError(err, map[string]string{"term": term})
		return
	}

	buttons := buttons(searchResult.Hits, term)

	messageText := firstHit(searchResult)

	if messageText == "" {
		mixpanelClient.Track(strconv.Itoa(m.Sender.ID), "Not found", &mixpanel.Event{
			Properties: map[string]interface{}{
				"term": term,
			},
		})
		b.Send(m.Chat, "Перевод не найден")
		return
	}

	mixpanelClient.Track(strconv.Itoa(m.Sender.ID), "Translate", &mixpanel.Event{
		Properties: map[string]interface{}{
			"term": term,
		},
	})

	b.Send(m.Chat,
		messageText,
		&tb.SendOptions{ParseMode: tb.ModeHTML},
		&tb.ReplyMarkup{InlineKeyboard: buttons})
}

func handleCallback(b *tb.Bot, c *tb.Callback, index bleve.Index, mixpanelClient mixpanel.Mixpanel) {
	// on inline button pressed (callback!)
	splittedStrings := strings.SplitN(c.Data, ",", 2)
	wordID := strings.TrimSpace(splittedStrings[0])
	term := splittedStrings[1]

	query := bleve.NewDocIDQuery([]string{wordID})
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"Keyword", "Value"}
	docIDSearchResult, err := index.Search(searchRequest)
	if err != nil {
		raven.CaptureError(err, map[string]string{"term": term})
		b.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте выбрать другой перевод",
		})
		return
	}

	searchResult, err := searchFunction(index, term)
	if err != nil {
		raven.CaptureError(err, map[string]string{"term": term})
		b.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте ввести другое ключевое слово",
		})
		return
	}

	buttons := buttons(searchResult.Hits, term)
	messageText := firstHit(docIDSearchResult)
	if messageText == "" {
		b.Respond(c, &tb.CallbackResponse{
			CallbackID: c.ID,
			Text:       "Произошла ошибка. Попробуйте выбрать другой перевод",
		})
		return
	}

	selectedKeyword := docIDSearchResult.Hits[0].Fields["Keyword"].(string)
	mixpanelClient.Track(strconv.Itoa(c.Sender.ID), "Selected", &mixpanel.Event{
		Properties: map[string]interface{}{
			"keyword": selectedKeyword,
		},
	})
	b.Edit(c.Message,
		messageText,
		&tb.SendOptions{ParseMode: tb.ModeHTML},
		&tb.ReplyMarkup{InlineKeyboard: buttons})

	// always respond!
	b.Respond(c, &tb.CallbackResponse{
		CallbackID: c.ID,
	})
}
