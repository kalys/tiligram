package main

import (
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
)

var ReindexCommand = cli.Command{
	Name:  "reindex",
	Usage: "Reindex data",
	Action: func(c *cli.Context) error {
		return nil
	},
}
