package main

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
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
		db, err := sql.Open("mysql", "root@/tili")
		if err != nil {
			return err
		}
		defer db.Close()

		rows, err := db.Query("select id, name from dict_d")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var id int
		var name string
		for rows.Next() {
			if err := rows.Scan(&id, &name); err != nil {
				log.Fatal(err)
			}
			log.Println(id, name)
		}
		return rows.Err()
	},
}
