package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestPostAdminLogin(t *testing.T) {
	body := body("email=admin&password=secret")
	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := doRequestWithHeader("POST", "/admin/login", body, &header)
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 302)
	expectHeader(t, resp, "Location", "/admin/faqs")
	expectHeaderMatches(t, resp, "Set-Cookie", "^Authorization.*Path=/admin.*HttpOnly$")
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

func TestGetAPILanguages(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/api/languages", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `[{"code":"en","name_en":"English","name_local":"English"},{"code":"de","name_en":"German","name_local":"Deutsch"},{"code":"fr","name_en":"French","name_local":"français"},{"code":"es","name_en":"Spanish","name_local":"español"},{"code":"it","name_en":"Italian","name_local":"italiano"},{"code":"nl","name_en":"Dutch","name_local":"Nederlands"},{"code":"pt","name_en":"Portuguese","name_local":"português"},{"code":"pt-BR","name_en":"Brazilian Portuguese","name_local":"português"},{"code":"da","name_en":"Danish","name_local":"dansk"},{"code":"sv","name_en":"Swedish","name_local":"svenska"},{"code":"no","name_en":"Norwegian Bokmål","name_local":"norsk bokmål"},{"code":"ru","name_en":"Russian","name_local":"русский"},{"code":"ar","name_en":"Arabic","name_local":"العربية"},{"code":"zh","name_en":"Chinese","name_local":"中文"}]`)
}

func TestGetAPIFAQs(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/api/faqs", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `[{"id":123,"texts":null},{"id":456,"texts":null},{"id":789,"texts":null}]`)
}

func TestGetAPISingleFAQ(t *testing.T) {
	faqRepository = &mockDB{}
	resp, err := doRequest("GET", "/api/faqs/123", emptyBody())
	if err != nil {
		panic(err)
	}

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `{"id":123,"texts":[{"locale":{"code":"de"},"question":"Welcher Tag ist heute?","answer":"Freitag"}]}`)
}

func doRequest(method, uri string, body *bytes.Buffer) (*httptest.ResponseRecorder, error) {
	return doRequestWithHeader(method, uri, body, nil)
}

func doRequestWithHeader(method, uri string, body *bytes.Buffer, header *http.Header) (*httptest.ResponseRecorder, error) {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, uri, body)
	if header != nil {
		req.Header = *header
	}
	if err != nil {
		return nil, err
	}

	router := httprouter.New()
	router.GET("/admin", getAdmin)
	router.GET("/admin/faqs", getAdminFAQs)
	router.GET("/admin/login", getAdminLogin)
	router.POST("/admin/login", postAdminLogin)
	router.GET("/admin/locales", getAdminLocales)
	router.GET("/admin/faqs/new", getAdminFAQsNew)
	router.GET("/admin/faqs/edit/:id", getAdminFAQsEdit)

	router.GET("/api/languages", getLanguages)
	router.GET("/api/faqs", getFAQs)
	router.GET("/api/faqs/:id", getSingleFAQ)
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

func expectHeaderMatches(t *testing.T, resp *httptest.ResponseRecorder, headerName string, expectedRegexp string) {
	var regex = regexp.MustCompile(expectedRegexp)
	actualHeaderValue := resp.Header().Get(headerName)
	if !regex.MatchString(actualHeaderValue) {
		t.Errorf("wrong header %v: '%v' did not match '%v'",
			headerName, actualHeaderValue, expectedRegexp)
	}
}

func body(str string) *bytes.Buffer {
	return bytes.NewBufferString(str)
}
func emptyBody() *bytes.Buffer {
	return bytes.NewBufferString("hello")
}
