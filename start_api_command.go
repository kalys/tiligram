package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/getsentry/sentry-go"
	"github.com/kalys/tiligram/internal/api"
	"github.com/urfave/cli/v2"
)

var StartApiCommand = cli.Command{
	Name:  "start-api",
	Usage: "Start HTTP search API",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "index-path",
			Usage: "Path where index is stored",
			Value: "bleve.search",
		},
		&cli.StringFlag{
			Name:  "listen-addr",
			Usage: "Address to listen on",
			Value: ":8080",
		},
		&cli.StringFlag{
			Name:  "sentry-dsn",
			Usage: "DSN for Sentry error tracking",
		},
		&cli.StringSliceFlag{
			Name:  "cors-origin",
			Usage: "Allowed CORS origin (repeatable)",
			Value: cli.NewStringSlice("https://osmonov.com"),
		},
	},
	Action: func(c *cli.Context) error {
		if err := sentry.Init(sentry.ClientOptions{Dsn: c.String("sentry-dsn")}); err != nil {
			return err
		}
		defer sentry.Flush(2 * time.Second)

		index, err := bleve.Open(c.String("index-path"))
		if err != nil {
			return err
		}
		defer index.Close()

		mux := http.NewServeMux()
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		api.NewHandler(index).Register(mux)

		srv := &http.Server{
			Addr:    c.String("listen-addr"),
			Handler: api.CORS(c.StringSlice("cors-origin"), mux),
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-quit
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			srv.Shutdown(ctx)
		}()

		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
		return nil
	},
}
