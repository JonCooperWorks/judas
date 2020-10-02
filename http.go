package judas

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
)

// Request is a deep cloneable *http.Request.
type Request struct {
	*http.Request
}

// CloneBody makes a copy of a request, including its body, while leaving the original body intact.
func (r *Request) CloneBody(ctx context.Context) (*Request, error) {
	req := &Request{Request: r.Request.Clone(ctx)}

	// We have to manually set the host in the URL.
	req.URL.Host = r.Request.Host

	// Prevent an error when sending the request
	req.RequestURI = ""
	if req.Body == nil {
		return req, nil
	}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return req, err
	}
	defer req.Body.Close()

	// Put back the original body
	r.Request.Body = ioutil.NopCloser(bytes.NewReader(body))

	// Clone the request body
	req.Request.Body = ioutil.NopCloser(bytes.NewReader(body))
	return req, nil
}

// Response is a *http.Response that allows cloning its body.
type Response struct {
	*http.Response
}

// CloneBody makes a copy of a response, including its body, while leaving the original body intact.
func (r *Response) CloneBody() (*Response, error) {
	newResponse := new(http.Response)

	if r.Response.Header != nil {
		newResponse.Header = r.Response.Header.Clone()
	}

	if r.Response.Trailer != nil {
		newResponse.Trailer = r.Response.Trailer.Clone()
	}

	newResponse.ContentLength = r.Response.ContentLength
	newResponse.Uncompressed = r.Response.Uncompressed
	newResponse.Request = r.Response.Request
	newResponse.TLS = r.Response.TLS
	newResponse.Status = r.Response.Status
	newResponse.StatusCode = r.Response.StatusCode
	newResponse.Proto = r.Response.Proto
	newResponse.ProtoMajor = r.Response.ProtoMajor
	newResponse.ProtoMinor = r.Response.ProtoMinor
	newResponse.Close = r.Response.Close
	copy(newResponse.TransferEncoding, r.Response.TransferEncoding)

	if r.Response.Body == nil {
		return &Response{Response: newResponse}, nil
	}

	body, err := ioutil.ReadAll(r.Response.Body)
	if err != nil {
		return &Response{Response: newResponse}, err
	}
	defer r.Response.Body.Close()

	// Put back the original body
	r.Response.Body = ioutil.NopCloser(bytes.NewReader(body))

	// Clone the request body
	newResponse.Body = ioutil.NopCloser(bytes.NewReader(body))
	return &Response{Response: newResponse}, nil
}
