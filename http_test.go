package judas

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestRequestClonePreservesOriginalBody(t *testing.T) {
	req, _ := http.NewRequest("POST", "", strings.NewReader("body"))
	request := &Request{req}
	clonedRequest, err := request.CloneBody(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// Read the cloned body first to make sure the original body doesn't get consumed.
	clonedBody, _ := ioutil.ReadAll(clonedRequest.Body)
	body, _ := ioutil.ReadAll(request.Body)

	if len(body) == 0 {
		t.Fatalf("Original body was drained with the clone")
	}

	if !bytes.Equal(body, clonedBody) {
		t.Fatalf("Cloned body does not match original, expected %s, got %s", string(body), string(clonedBody))
	}

	if req.RequestURI != "" {
		t.Fatalf("RequestURI was not removed before clone")
	}

	if request.Host != clonedRequest.URL.Host {
		t.Fatalf("Host was not copied into URL")
	}
}

func TestResponseClonePreservesOriginalBody(t *testing.T) {
	body := "body"
	resp := &http.Response{
		Status:        "200 OK",
		StatusCode:    200,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		Body:          ioutil.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	response := &Response{resp}
	clonedResponse, err := response.CloneBody()
	if err != nil {
		t.Fatal(err)
	}

	// Read the cloned body first to make sure the original body doesn't get consumed.
	clonedBody, _ := ioutil.ReadAll(clonedResponse.Body)
	originalBody, _ := ioutil.ReadAll(response.Body)

	if len(originalBody) == 0 {
		t.Fatal("Original body was consumed when clone body was read")
	}

	if !bytes.Equal(clonedBody, originalBody) {
		t.Fatalf("Cloned body does not match original, expected %s, got %s", string(originalBody), string(clonedBody))
	}

	if response.Status != clonedResponse.Status {
		t.Fatalf("Cloned response status does not match original response status")
	}
}
