package httputil

import (
	"fmt"
	"net/http"
	"strings"
)

func NewClient(rt http.RoundTripper) *http.Client {
	return &http.Client{Transport: rt}
}

type bearerAuthRoundTripper struct {
	bearerToken string
	rt          http.RoundTripper
}

func NewBearerAuthRoundTripper(token string, rt http.RoundTripper) http.RoundTripper {
	return &bearerAuthRoundTripper{token, rt}
}

func (rt *bearerAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) == 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rt.bearerToken))
	}
	return rt.rt.RoundTrip(req)
}

type basicAuthRoundTripper struct {
	username string
	password string
	rt       http.RoundTripper
}

func NewBasicAuthRoundTripper(username string, password string, rt http.RoundTripper) http.RoundTripper {
	return &basicAuthRoundTripper{username, password, rt}
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if len(req.Header.Get("Authorization")) != 0 {
		return rt.rt.RoundTrip(req)
	}
	req.SetBasicAuth(rt.username, strings.TrimSpace(rt.password))
	return rt.rt.RoundTrip(req)
}
