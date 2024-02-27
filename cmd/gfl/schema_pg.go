package main

import (
	"log"

	"github.com/spf13/cobra"

	"github.com/james-elicx/gontentful"
)

func init() {
	schemaCmd.AddCommand(pgSchemaCmd)
}

var pgSchemaCmd = &cobra.Command{
	Use:   "pg",
	Short: "Creates postgres schema",

	Run: func(cmd *cobra.Command, args []string) {
		log.Println("creating postgres schema...")

		client := gontentful.NewClient(&gontentful.ClientOptions{
			CdnURL:        apiURL,
			SpaceID:       spaceID,
			EnvironmentID: environmentID,
			CdnToken:      cdnToken,
			CmaURL:        cmaURL,
			CmaToken:      cmaToken,
		})

		log.Println("get space...")
		space, err := client.Spaces.GetSpace()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("get cma types...")
		cmaTypes, err := client.ContentTypes.GetCMATypes()
		if err != nil {
			log.Fatal(err)
		}

		log.Println("executing postgres schema...")
		schema := gontentful.NewPGSQLSchema(schemaName, space.Locales, "", cmaTypes.Items, 0)
		err = schema.Exec(databaseURL)
		if err != nil {
			log.Fatal(err)
		}

		// log.Println("creating references...")
		// refs := gontentful.NewPGReferences(schema)
		// err = refs.Exec(databaseURL)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		// log.Println("creating postgres functions...")

		// funcs := gontentful.NewPGFunctions(schemaName)
		// err = funcs.Exec(databaseURL)
		// if err != nil {
		// 	log.Fatal(err)
		// }

		log.Println("postgres schema successfully executed")
	},
}
