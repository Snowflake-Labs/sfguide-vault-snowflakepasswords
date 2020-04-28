package main

import (
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/sanderiam/snowflakepasswords-database-plugin"
)

// this just encapsulates the rest of the program
func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	err := snowflakepasswords.Run(apiClientMeta.GetTLSConfig())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
