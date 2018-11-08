package main

import (
	"errors"
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/davecgh/go-spew/spew"
	"github.com/getsentry/raven-go"
	tb "gopkg.in/tucnak/telebot.v2"
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
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
	},
	Action: func(c *cli.Context) error {
		spew.Dump()
		raven.SetDSN("https://3c6494f634b9481d80fef1f3473c1ef1:78ed97859e6043dd82f6c9015ab11c05@sentry.io/1318463")

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

		searchFunction := func(term string) *bleve.SearchResult {
			query := bleve.NewQueryStringQuery(term)

			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"Keyword", "Value"}
			searchResult, _ := index.Search(searchRequest)
			return searchResult
		}

		firstHit := func(sr *bleve.SearchResult) (string, error) {
			if len(sr.Hits) == 0 {
				return "", errors.New("No Hits")
			}
			hitValue := strings.Replace(sr.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
			hitKeyword := strings.Join([]string{"<b>", sr.Hits[0].Fields["Keyword"].(string), "</b>"}, "")

			return strings.Join([]string{hitKeyword, hitValue}, "\n"), nil
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
				docIDSearchResult, _ := index.Search(searchRequest)

				searchResult := searchFunction(term)
				buttons := buttons(searchResult.Hits, term)

				messageText, err := firstHit(docIDSearchResult)

				if err == nil {
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
				term := fmt.Sprintf("Keyword:%s^5 Value:%s", m.Text, m.Text)

				searchResult := searchFunction(term)
				buttons := buttons(searchResult.Hits, term)

				messageText, err := firstHit(searchResult)

				if err == nil {
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
