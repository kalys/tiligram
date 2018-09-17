package main

import (
	"fmt"
	"github.com/blevesearch/bleve"
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

		b.Handle("/translate", func(m *tb.Message) {
			term := fmt.Sprintf("Keyword:%s^5 Value:%s", m.Payload, m.Payload)
			// term := m.Payload
			query := bleve.NewQueryStringQuery(term)

			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"Keyword", "Value"}
			searchResult, _ := index.Search(searchRequest)
			responseText := strings.Replace(searchResult.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
			b.Send(m.Sender, responseText)
		})

		b.Handle(tb.OnText, func(m *tb.Message) {
			spew.Dump(m)
			spew.Dump(m.Payload)
			term := fmt.Sprintf("Keyword:%s^5 Value:%s", m.Text, m.Text)
			// term := m.Payload
			query := bleve.NewQueryStringQuery(term)

			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"Keyword", "Value"}
			searchResult, _ := index.Search(searchRequest)
			responseText := strings.Replace(searchResult.Hits[0].Fields["Value"].(string), "&nbsp;", "", -1)
			b.Send(m.Sender, responseText)
		})

		// b.Handle(tb.OnQuery, func(q *tb.Query) {
		// 	urls := []string{
		// 		"http://photo.jpg",
		// 		"http://photo2.jpg",
		// 	}

		// 	results := make(tb.Results, len(urls)) // []tb.Result
		// 	for i, url := range urls {
		// 		result := &tb.PhotoResult{
		// 			URL: url,

		// 			// required for photos
		// 			ThumbURL: url,
		// 		}

		// 		results[i] = result
		// 		results[i].SetResultID(strconv.Itoa(i)) // It's needed to set a unique string ID for each result
		// 	}

		// 	err := b.Answer(q, &tb.QueryResponse{
		// 		Results:   results,
		// 		CacheTime: 60, // a minute
		// 	})

		// 	if err != nil {
		// 		fmt.Println(err)
		// 	}
		// })

		b.Start()

		return nil
	},
}
