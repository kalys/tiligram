package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/dukex/mixpanel"
	"github.com/enbritely/heartbeat-golang"
	"github.com/getsentry/raven-go"
	"github.com/kalys/tiligram/internal/bot"
	tb "gopkg.in/tucnak/telebot.v2"
	"gopkg.in/urfave/cli.v2"
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
		&cli.StringFlag{
			Name:  "heartbeat-addr",
			Usage: "Address for the heartbeat health-check listener",
			Value: ":10101",
		},
	},
	Action: func(c *cli.Context) error {
		go heartbeat.RunHeartbeatService(c.String("heartbeat-addr"))

		if err := raven.SetDSN(c.String("raven-dsn")); err != nil {
			return err
		}

		index, err := bleve.Open(c.String("index-path"))
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
		bot.NewBotHandler(index, analytics, b).RegisterHandlers()

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
