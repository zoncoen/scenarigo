package http

import "net/http"

type result struct {
	status string // http.Response.Status format e.g. "200 OK"
	body   interface{}
}

func newResult(resp *http.Response, body interface{}) *result {
	return &result{
		status: resp.Status,
		body:   body,
	}
}
