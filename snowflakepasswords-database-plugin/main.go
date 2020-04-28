package main

import (
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"github.com/sanderiam/samplehashivaultsnowflakepasswords"
)

// this just encapsulates the rest of the program
func main() {
	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	err := samplehashivaultsnowflakepasswords.Run(apiClientMeta.GetTLSConfig())
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
