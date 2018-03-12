package plugins

import (
	"errors"
	"net/http"
	"plugin"
)

var (
	// ErrPluginMalformed is returned when the plugin is missing a required export.
	ErrPluginMalformed = errors.New("malformed plugin: MUST export Name string, Intialize func() (map[string]string, error) and ProcessTransactions func(chan<- HTTPTransaction, map[string]string)")
)

// HTTPTransaction represents a complete request - response flow.
type HTTPTransaction struct {
	Request  http.Request
	Response http.Response
}

// PluginArguments is a map[string]interface{} of arguments to be passed to your ProcessTransactions method.
type PluginArguments map[string]interface{}

// JudasPlugin contains functions and variables that Judas will be looking for in your plugin.
// Plugins will be loaded from any .so file in the same directory as the judas executable.
type JudasPlugin struct {
	// Name of the plugin.
	Name string

	// Initialize is where you should do your plugin's setup, like defining command line flags.
	// You are allowed to return a PluginArguments of arguments that will be passed to your ProcessTransactions function.
	Intialize func() (PluginArguments, error)

	// ProcessTransactions takes a chan that produces request - response pair and does something.
	// Judas plugins should implement this method to process request - response pairs as they are generated.
	// Requests and responses will be passed by value, allowing each plugin to run in its own goroutine.
	ProcessTransactions func(<-chan HTTPTransaction, PluginArguments)
}

// New loads a JudasPlugin from a file path.
// TODO: Add code signing.
func New(path string) (*JudasPlugin, error) {
	plugin, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	name, err := plugin.Lookup("Name")
	if err != nil {
		return nil, ErrPluginMalformed
	}

	initialize, err := plugin.Lookup("Initialize")
	if err != nil {
		return nil, ErrPluginMalformed
	}

	processTransactions, err := plugin.Lookup("ProcessTransactions")
	if err != nil {
		return nil, ErrPluginMalformed
	}

	return &JudasPlugin{
		Name:                *name.(*string),
		Intialize:           initialize.(func() (PluginArguments, error)),
		ProcessTransactions: processTransactions.(func(<-chan HTTPTransaction, PluginArguments)),
	}, nil
}
