package main

import (
	"log"
	"net/http/httputil"

	"github.com/joncooperworks/judas/plugins"
)

type loggingPlugin struct{}

func (l loggingPlugin) Name() string {
	return "LoggingPlugin"
}

func (l loggingPlugin) Initialize() (plugins.PluginArguments, error) {
	log.Println("Initializing", l.Name())
	return map[string]interface{}{}, nil
}

func (l loggingPlugin) ProcessTransactions(transactions <-chan plugins.HTTPTransaction, arguments plugins.PluginArguments) {
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

var Plugin loggingPlugin

func main() {
	// Shut the IDE up.
}
