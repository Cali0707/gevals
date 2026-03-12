package mcpclient

import (
	"net/http"
)

// HeaderRoundTripper wraps an http.RoundTripper and adds custom headers to every request.
type HeaderRoundTripper struct {
	// Headers are the multi-value headers to add to each request.
	Headers http.Header
	// Transport is the underlying RoundTripper to use for the actual request
	Transport http.RoundTripper
}

// NewHeaderRoundTripper creates a new HeaderRoundTripper with the given headers.
// If transport is nil, http.DefaultTransport is used.
func NewHeaderRoundTripper(headers http.Header, transport http.RoundTripper) *HeaderRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &HeaderRoundTripper{
		Headers:   headers,
		Transport: transport,
	}
}

// RoundTrip implements the http.RoundTripper interface.
// It adds the configured headers to the request before passing it to the underlying transport.
func (h *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, values := range h.Headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	return h.Transport.RoundTrip(req)
}
