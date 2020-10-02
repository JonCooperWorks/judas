package judas

import (
	"log"
	"net/http"
	"net/url"
)

// Config holds all program configuration required to spin up a functioning instance of Judas.
type Config struct {
	TargetURL            *url.URL
	ResponseTransformers []ResponseTransformer
	SourceInsecure       bool
	Logger               *log.Logger
	Transport            http.RoundTripper
}
