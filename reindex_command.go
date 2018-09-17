package main

import (
	"database/sql"
	"github.com/blevesearch/bleve"
	// "github.com/davecgh/go-spew/spew"
	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/urfave/cli.v2" // imports as package "cli"
)

var ReindexCommand = cli.Command{
	Name:  "reindex",
	Usage: "Reindex data",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "batch-size",
			Usage: "Index batch size",
			Value: 100,
		},
		&cli.StringFlag{
			Name:  "from-db",
			Usage: "MySQL database URI string",
			Value: "root@/tili",
		},
		&cli.StringFlag{
			Name:  "index-path",
			Usage: "Path where index is stored",
			Value: "bleve.search",
		},
	},
	Action: func(c *cli.Context) error {
		type record struct {
			Type    string
			Id      string
			Keyword string
			Value   string
		}

		db, err := sql.Open("mysql", c.String("from-db"))
		if err != nil {
			panic(err.Error()) // Just for example purpose. You should use proper error handling instead of panic
		}
		defer db.Close()

		batchSize := c.Int("batch-size")

		indexMapping, err := buildIndexMapping()
		if err != nil {
			panic(err)
		}

		index, err := bleve.New(c.String("index-path"), indexMapping)
		if err != nil {
			panic(err)
		}

		batch := index.NewBatch()
		batchCount := 1

		p := record{Type: "word"}

		rows, err := db.Query("select id, keyword, value from dict_kw")
		if err != nil {
			panic(err.Error())
		}
		defer rows.Close()

		for rows.Next() {
			err := rows.Scan(&p.Id, &p.Keyword, &p.Value)
			if err != nil {
				panic(err.Error())
			}

			// spew.Dump(p)

			batch.Index(p.Id, p)

			batchCount++
			if batchCount >= batchSize {
				err = index.Batch(batch)
				if err != nil {
					panic(err)
				}
				batch = index.NewBatch()
				batchCount = 0
			}
		}

		if batchCount > 0 {
			err = index.Batch(batch)
			if err != nil {
				panic(err)
			}
		}

		err = rows.Err()
		if err != nil {
			panic(err.Error())
		}
		return nil
	},
}
