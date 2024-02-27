package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/james-elicx/gontentful"
)

func init() {
	dropCmd.AddCommand(pgDropCmd)
}

var pgDropCmd = &cobra.Command{
	Use:   "pg",
	Short: "Drop [content] postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("dropping %s schema...", schemaName)
		query := gontentful.NewPGDrop(schemaName)
		err := query.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Printf("%s schema dropped successfully", schemaName)
	},
}
