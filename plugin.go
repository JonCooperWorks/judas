package judas

import (
	"context"
	"log"
	"net/url"
	"plugin"
)

// InitializerFunc is a go function that should be exported by a function package.
// It should be named "New".
// Your InitializerFunc should return an instance of your Plugin with a reference to judas's logger for consistent logging.
type InitializerFunc func(*log.Logger) (Plugin, error)

// PluginBroker handles sending messages to plugins.
type PluginBroker struct {
	plugins []*pluginInfo
	logger  *log.Logger
}

// SendResult sends a *Result to all loaded plugins for further processing.
func (p *PluginBroker) SendResult(exchange *HTTPExchange) error {
	for _, plugin := range p.plugins {
		// Give each plugin its own request and response.
		req, err := exchange.Request.CloneBody(context.Background())
		if err != nil {
			return err
		}

		resp, err := exchange.Response.CloneBody()
		if err != nil {
			return err
		}

		plugin.Input <- &HTTPExchange{
			Request:  req,
			Response: resp,
			Target:   exchange.Target,
		}
	}
	return nil
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
}

func (p *PluginBroker) run(plugin *pluginInfo, exchanges <-chan *HTTPExchange) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				p.logger.Printf("WARN: panic in loaded plugin")
			}
		}()
		plugin.Listen(exchanges)
	}()
}

// LoadPlugins loads judas plugins from a list of file paths.
func LoadPlugins(logger *log.Logger, paths []string) (*PluginBroker, error) {
	broker := &PluginBroker{logger: logger}

	for _, path := range paths {
		plg, err := plugin.Open(path)
		if err != nil {
			return nil, err
		}

		symbol, err := plg.Lookup("New")
		if err != nil {
			return nil, err
		}

		// Go needs this, InitializerFunc is purely for documentation.
		initializer := symbol.(func(*log.Logger) (Plugin, error))
		judasPlugin, err := initializer(logger)
		if err != nil {
			return nil, err
		}

		input := make(chan *HTTPExchange)
		httpfuzzPlugin := &pluginInfo{
			Input:  input,
			Plugin: judasPlugin,
		}

		// Listen for results in a goroutine for each plugin
		broker.add(httpfuzzPlugin)
		broker.run(httpfuzzPlugin, input)
	}

	return broker, nil
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
