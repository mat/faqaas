package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
)

// TODO https://stackoverflow.com/questions/25337126/testing-http-routes-in-golang#25585458

func TestGetAdminIndex(t *testing.T) {
	body := bytes.NewBufferString("hello")
	handle := getAdmin
	resp, err := doRequest("GET", "/admin", body, handle)
	if err != nil {
		panic(err)
	}

	expectBodyContains(t, resp, `<a href="/admin/faqs">Found</a>`)
	expectStatus(t, resp, 302)
	expectHeader(t, resp, "Location", "/admin/faqs")
}

func TestGetAdminLogin(t *testing.T) {
	body := bytes.NewBufferString("")
	handle := getAdminLogin
	resp, err := doRequest("GET", "/admin/login", body, handle)
	if err != nil {
		panic(err)
	}

	expectBodyContains(t, resp, `<title>Admin / Login</title>`)
	expectBodyContains(t, resp, `<form action="/admin/login" method="post"`)
	expectStatus(t, resp, 200)
}

func TestGetAdminFAQs(t *testing.T) {
	faqRepository = &mockDB{}

	body := bytes.NewBufferString("")
	handle := getAdminFAQs
	resp, err := doRequest("GET", "/admin/faqs", body, handle)
	if err != nil {
		panic(err)
	}

	expectBodyContains(t, resp, `<title>Admin / FAQs</title>`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/123"`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/456"`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/789"`)
	expectStatus(t, resp, 200)
}

func doRequest(method, uri string, body *bytes.Buffer, handle httprouter.Handle) (*httptest.ResponseRecorder, error) {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	router := httprouter.New()
	router.GET("/admin", getAdmin)
	router.GET("/admin/faqs", getAdminFAQs)
	router.GET("/admin/login", getAdminLogin)
	// router.Handle(method, uri, handle)
	router.ServeHTTP(resp, req)
	return resp, nil
}

func expectStatus(t *testing.T, resp *httptest.ResponseRecorder, expected int) {
	if status := resp.Code; status != expected {
		t.Errorf("wrong status code: is %v but wanted %v", status, expected)
	}
}

func expectBodyContains(t *testing.T, resp *httptest.ResponseRecorder, expected string) {
	if !strings.Contains(resp.Body.String(), expected) {
		t.Errorf("wrong body: '%v' not contained in '%v'",
			expected, resp.Body.String())
	}
}

func expectHeader(t *testing.T, resp *httptest.ResponseRecorder, headerName string, expected string) {
	if resp.Header().Get(headerName) != expected {
		t.Errorf("wrong header %v: is '%v' but wanted '%v'",
			headerName, resp.Header().Get(headerName), expected)
	}
}
