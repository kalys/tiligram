package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/dukex/mixpanel"
	"github.com/getsentry/sentry-go"
	"github.com/kalys/tiligram/internal/bot"
	"github.com/urfave/cli/v2"
	tb "gopkg.in/telebot.v3"
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
			Name:  "sentry-dsn",
			Usage: "DSN for Sentry error tracking",
		},
		&cli.StringFlag{
			Name:  "mixpanel-token",
			Usage: "Mixpanel token",
			Value: "some-token",
		},
		&cli.StringFlag{
			Name:  "heartbeat-addr",
			Usage: "Address for the heartbeat health-check listener",
			Value: ":10101",
		},
	},
	Action: func(c *cli.Context) error {
		go func() {
			http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			http.ListenAndServe(c.String("heartbeat-addr"), nil)
		}()

		if err := sentry.Init(sentry.ClientOptions{Dsn: c.String("sentry-dsn")}); err != nil {
			return err
		}
		defer sentry.Flush(2 * time.Second)

		index, err := bleve.OpenUsing(c.String("index-path"), map[string]interface{}{"read_only": true})
		if err != nil {
			return err
		}
		defer index.Close()

		b, err := tb.NewBot(tb.Settings{
			Token:  c.String("bot-token"),
			Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		})
		if err != nil {
			return err
		}

		analytics := mixpanel.New(c.String("mixpanel-token"), "")
		bot.NewBotHandler(index, analytics, b).RegisterHandlers(b)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-quit
			b.Stop()
		}()

		b.Start()
		return nil
	},
}
