// vim:tabstop=4
package main

import (
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
	"os"
)

func ImportDB(c *cli.Context) error {
	return nil
}

func Reindex(c *cli.Context) error {
	return nil
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			&StartBotCommand,
			{
				Name:   "import-db",
				Usage:  "Import data from MySQL database",
				Action: ImportDB,
			},
			{
				Name:   "reindex",
				Usage:  "Reindex data",
				Action: Reindex,
			},
		},
	}

	app.Run(os.Args)
}
