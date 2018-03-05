// vim:tabstop=4
package main

import (
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
	"os"
)

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			&StartBotCommand,
			&ImportDBCommand,
			&ReindexCommand,
		},
	}

	app.Run(os.Args)
}
