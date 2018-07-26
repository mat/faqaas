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
	resp, err := doRequest("GET", "/admin", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 302)
	expectHeader(t, resp, "Location", "/admin/faqs")
}

func TestGetAdminLogin(t *testing.T) {
	resp, err := doRequest("GET", "/admin/login", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / Login</title>`)
	expectBodyContains(t, resp, `<form action="/admin/login" method="post"`)

}

func TestGetAdminFAQs(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/admin/faqs", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / FAQs</title>`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/123"`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/456"`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/789"`)
}

func TestGetAdminFAQsNew(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/admin/faqs/new", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / New FAQ</title>`)
	expectBodyContains(t, resp, `<form action="/admin/faqs/create" method="post">`)
}

func TestGetAdminFAQsEdit(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/admin/faqs/edit/123", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / Edit FAQ</title>`)
	expectBodyContains(t, resp, `<form action="/admin/faqs/delete" method="post">`)
}

func TestGetAdminLocales(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/admin/locales", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / Languages</title>`)

	expectBodyContains(t, resp, `<td>de</td>`)
	expectBodyContains(t, resp, `German (Deutsch)`)

	expectBodyContains(t, resp, `<td>es</td>`)
	expectBodyContains(t, resp, `Spanish (español)`)
}

func doRequest(method, uri string, body *bytes.Buffer) (*httptest.ResponseRecorder, error) {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return nil, err
	}

	router := httprouter.New()
	router.GET("/admin", getAdmin)
	router.GET("/admin/faqs", getAdminFAQs)
	router.GET("/admin/login", getAdminLogin)
	router.GET("/admin/locales", getAdminLocales)
	router.GET("/admin/faqs/new", getAdminFAQsNew)
	router.GET("/admin/faqs/edit/:id", getAdminFAQsEdit)
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

func emptyBody() *bytes.Buffer {
	return bytes.NewBufferString("hello")
}
