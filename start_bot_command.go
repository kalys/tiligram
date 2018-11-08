package main

import (
	"fmt"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/davecgh/go-spew/spew"
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
		index, err := bleve.Open(c.String("index-path"))
		_ = index // FIXME
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

		firstHit := func(sr *bleve.SearchResult) string {
			return strings.Replace(sr.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
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

		// b.Handle("/translate", func(m *tb.Message) {
		// 	term := fmt.Sprintf("Keyword:%s^5 Value:%s", m.Payload, m.Payload)
		// 	searchResult := searchFunction(term)
		// 	responseText := firstHit(searchResult)
		// 	b.Send(m.Sender, responseText)
		// })

		b.Handle(tb.OnCallback, func(c *tb.Callback) {
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

			translation := strings.Replace(docIDSearchResult.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
			b.Edit(c.Message, translation, &tb.ReplyMarkup{
				InlineKeyboard: buttons,
			})

			// always respond!
			b.Respond(c, &tb.CallbackResponse{
				CallbackID: c.ID,
			})
		})

		b.Handle(tb.OnText, func(m *tb.Message) {
			spew.Dump()
			term := fmt.Sprintf("Keyword:%s^5 Value:%s", m.Text, m.Text)

			searchResult := searchFunction(term)
			buttons := buttons(searchResult.Hits, term)

			responseText := firstHit(searchResult)
			b.Send(m.Sender, responseText, &tb.ReplyMarkup{
				InlineKeyboard: buttons,
			})
		})

		b.Start()

		return nil
	},
}
