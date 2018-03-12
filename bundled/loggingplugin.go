package main

import (
	"log"
	"net/http/httputil"

	"github.com/joncooperworks/judas/plugins"
)

type loggingPlugin struct{}

// Name of our plugin
func (l loggingPlugin) Name() string {
	return "LoggingPlugin"
}

// Initialize does nothing since this is a logging plugin.
// Print something to console so it at least serves some use.
func (l loggingPlugin) Initialize() (plugins.PluginArguments, error) {
	log.Println("Initializing", l.Name())
	return map[string]interface{}{}, nil
}

// ProcessTransactions logs each HTTP request - response to console.
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

// Plugin will be picked up by Judas.
var Plugin loggingPlugin

func main() {
	// Shut the IDE up.
}
