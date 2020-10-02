package judas

import (
	"context"
	"log"
	"net/url"
	"sync"
)

// InitializerFunc is a go function that should be exported by a function package.
// It should be named "New".
// Your InitializerFunc should return an instance of your Plugin with a reference to judas's logger for consistent logging.
type InitializerFunc func(*log.Logger) (Plugin, error)

// PluginBroker handles sending messages to plugins.
type PluginBroker struct {
	plugins   []*pluginInfo
	waitGroup sync.WaitGroup
}

// SendResult sends a *Result to all loaded plugins for further processing.
func (p *PluginBroker) SendResult(exchange *HTTPExchange) error {
	for _, plugin := range p.plugins {
		// Give each plugin its own request.
		req, err := exchange.Request.CloneBody(context.Background())
		if err != nil {
			return err
		}

		resp, err := exchange.Response.CloneBody()
		if err != nil {
			return err
		}

		exchange.Request = req
		exchange.Response = resp

		plugin.Input <- exchange
	}
	return nil
}

// Wait blocks the goroutine until all plugins have finished executing.
func (p *PluginBroker) Wait() {
	p.waitGroup.Wait()
}

// SignalDone closes all plugin chans that are waiting on results.
// Call only after all results have been sent.
func (p *PluginBroker) SignalDone() {
	for _, plugin := range p.plugins {
		close(plugin.Input)
	}
}

func (p *PluginBroker) add(plugin *pluginInfo) {
	p.plugins = append(p.plugins, plugin)
	p.waitGroup.Add(1)
}

func (p *PluginBroker) run(plugin *pluginInfo, exchanges <-chan *HTTPExchange) {
	go func() {
		plugin.Listen(exchanges)
		p.waitGroup.Done()
	}()
}

// Plugin implementations will be given a stream of HTTPExchanges to let plugins capture valuable information out of request-response transactions.
type Plugin interface {
	Listen(<-chan *HTTPExchange)
}

type pluginInfo struct {
	Input chan<- *HTTPExchange
	Plugin
}

// HTTPExchange contains the request sent by the user to us and the response received from the target server.
// Plugins can use this struct to pull information out of requests and responses.
type HTTPExchange struct {
	Request  *Request
	Response *Response
	Target   *url.URL
}
