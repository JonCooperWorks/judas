package main

import (
	"log"

	"github.com/joncooperworks/judas"
)

type searchLoggingPlugin struct {
	logger *log.Logger
}

// Listen pulls google search queries out of HTTP exchanges
func (p *searchLoggingPlugin) Listen(exchanges <-chan *judas.HTTPExchange) {
	for exchange := range exchanges {
		searchQuery := exchange.Request.URL.Query().Get("q")
		if searchQuery != "" && exchange.Request.URL.Host == exchange.Target.Host {
			p.logger.Printf("Search query: %s", searchQuery)
		}
	}
}

// New returns a plugin that logs google searches.
func New(logger *log.Logger) (judas.Listener, error) {
	return &searchLoggingPlugin{logger: logger}, nil
}
