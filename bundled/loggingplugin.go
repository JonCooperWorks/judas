package main

import (
	"log"
	"net/http/httputil"

	"github.com/joncooperworks/judas/plugins"
)

var Name = "JudasLoggingPlugin"

func Initialize() (plugins.PluginArguments, error) {
	log.Println("Initializing", Name)
	return map[string]interface{}{}, nil
}

func ProcessTransactions(transactions <-chan plugins.HTTPTransaction, arguments plugins.PluginArguments) {
	for transaction := range transactions {
		req, err := httputil.DumpRequest(&transaction.Request, true)
		if err != nil {
			log.Println("Error dumping request to console.")
			return
		}
		log.Println(string(req))

		resp, err := httputil.DumpResponse(&transaction.Response, false)
		if err != nil {
			log.Println("Error dumping response to console.")
			return
		}
		log.Println(string(resp))
	}
}

func main() {
	// Shut the IDE up.
}
