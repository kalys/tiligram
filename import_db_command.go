package main

import (
	"database/sql"
	//"github.com/davecgh/go-spew/spew"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
	"log"
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
			panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		}
		defer db.Close()

		var (
			id   int
			name string
		)

		rows, err := db.Query("select id, name from dict_d")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		for rows.Next() {
			err := rows.Scan(&id, &name)
			if err != nil {
				log.Fatal(err)
			}
			log.Println(id, name)
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		// spew.Dump(rows)
		return nil
	},
}
