package main

import (
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
)

var StartBotCommand = cli.Command{
	Name:  "start-bot",
	Usage: "Start Telegram bot",
	Flags: StartBotFlags,
	Action: func(c *cli.Context) error {
		return nil
	},
}

var StartBotFlags = []cli.Flag{
	&cli.StringFlag{
		Name:  "bot-token",
		Usage: "Token for Telegram API",
	},
}
