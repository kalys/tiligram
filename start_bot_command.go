package main

import (
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/davecgh/go-spew/spew"
	"github.com/dukex/mixpanel"
	"github.com/getsentry/raven-go"
	tb "gopkg.in/tucnak/telebot.v2"
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
	"strconv"
	_ "strconv"
	"strings"
	"time"
)

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
		spew.Dump()
		raven.SetDSN(c.String("raven-dsn"))
		mixpanelClient := mixpanel.New(c.String("mixpanel-token"), "")

		index, err := bleve.Open(c.String("index-path"))
		if err != nil {
			panic(err)
		}

		b, err := tb.NewBot(tb.Settings{
			Token:  c.String("bot-token"),
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			panic(err)
		}

		searchFunction := func(term string) (*bleve.SearchResult, error) {
			query := bleve.NewQueryStringQuery(term)

			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"Keyword", "Value"}
			return index.Search(searchRequest)
		}

		firstHit := func(sr *bleve.SearchResult) string {
			if len(sr.Hits) == 0 {
				return ""
			}
			hitValue := strings.Replace(sr.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
			hitKeyword := strings.Join([]string{"<b>", sr.Hits[0].Fields["Keyword"].(string), "</b>"}, "")

			return strings.Join([]string{hitKeyword, hitValue}, "\n")
		}

		buttons := func(hits search.DocumentMatchCollection, query string) [][]tb.InlineButton {
			var inlineButtons []tb.InlineButton
			for _, hit := range hits {
				uniqueString := strings.Join([]string{hit.ID, query}, ",")
				inlineBtn := tb.InlineButton{
					Unique: uniqueString,
					Text:   hit.Fields["Keyword"].(string),
				}

				inlineButtons = append(inlineButtons, inlineBtn)
			}

			return [][]tb.InlineButton{inlineButtons}
		}

		startHelpHandler := func(m *tb.Message) {
			responseText := "Вебсайт: http://tili.kg\nКонтакты: @kalys, kalys@osmonov.com"
			b.Send(m.Sender, responseText)
		}

		b.Handle("/help", startHelpHandler)
		b.Handle("/start", startHelpHandler)

		// b.Handle("/translate", func(m *tb.Message) {
		// 	term := fmt.Sprintf("Keyword:%s^5 Value:%s", m.Payload, m.Payload)
		// 	searchResult := searchFunction(term)
		// 	responseText := firstHit(searchResult)
		// 	b.Send(m.Sender, responseText)
		// })

		b.Handle(tb.OnCallback, func(c *tb.Callback) {
			raven.CapturePanic(func() {
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

				searchResult, err := searchFunction(term)
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
				} else {
					selectedKeyword := docIDSearchResult.Hits[0].Fields["Keyword"].(string)
					mixpanelClient.Track(strconv.Itoa(c.Sender.ID), "Selected", &mixpanel.Event{
						Properties: map[string]interface{}{
							"keyword": selectedKeyword,
						},
					})
					b.Edit(c.Message,
						messageText,
						&tb.SendOptions{
							ParseMode: tb.ModeHTML,
						},
						&tb.ReplyMarkup{
							InlineKeyboard: buttons,
						})
				}

				// always respond!
				b.Respond(c, &tb.CallbackResponse{
					CallbackID: c.ID,
				})
			}, nil)
		})

		b.Handle(tb.OnText, func(m *tb.Message) {
			raven.CapturePanic(func() {
				term := m.Text
				queryString := fmt.Sprintf("Keyword:%s^5 Value:%s", term, term)

				searchResult, err := searchFunction(queryString)
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
					b.Send(m.Sender, "Перевод не найден")
				} else {
					mixpanelClient.Track(strconv.Itoa(m.Sender.ID), "Translate", &mixpanel.Event{
						Properties: map[string]interface{}{
							"term": term,
						},
					})
					b.Send(m.Sender,
						messageText,
						&tb.SendOptions{
							ParseMode: tb.ModeHTML,
						},
						&tb.ReplyMarkup{
							InlineKeyboard: buttons,
						})
				}
			}, nil)
		})

		b.Start()

		return nil
	},
}
