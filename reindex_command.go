package main

import (
	"database/sql"

	"github.com/blevesearch/bleve"
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
			ID      string
			Keyword string
			Value   string
		}

		db, err := sql.Open("mysql", c.String("from-db"))
		if err != nil {
			return err
		}
		defer db.Close()

		batchSize := c.Int("batch-size")

		index, err := bleve.New(c.String("index-path"), buildIndexMapping())
		if err != nil {
			return err
		}

		batch := index.NewBatch()
		batchCount := 1

		p := record{Type: "word"}

		rows, err := db.Query("select id, keyword, value from dict_kw")
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			if err := rows.Scan(&p.ID, &p.Keyword, &p.Value); err != nil {
				return err
			}

			// spew.Dump(p)

			if err := batch.Index(p.ID, p); err != nil {
				return err
			}

			batchCount++
			if batchCount >= batchSize {
				if err := index.Batch(batch); err != nil {
					return err
				}
				batch = index.NewBatch()
				batchCount = 0
			}
		}

		if batchCount > 0 {
			if err := index.Batch(batch); err != nil {
				return err
			}
		}

		return rows.Err()
	},
}
