package main

import (
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
)

var ImportDBCommand = cli.Command{
	Name:  "import-db",
	Usage: "Import data from MySQL database",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "from-db",
			Usage: "MySQL database name to import from",
		},
	},
	Action: func(c *cli.Context) error {
		return nil
	},
}
