package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TODO https://stackoverflow.com/questions/25337126/testing-http-routes-in-golang#25585458

func TestGetAdminIndex(t *testing.T) {
	resp := doRequest("GET", "/admin", emptyBody())

	expectStatus(t, resp, 302)
	expectHeader(t, resp, "Location", "/admin/faqs")
}

func TestGetAdminLogin(t *testing.T) {
	resp := doRequest("GET", "/admin/login", emptyBody())

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / Login</title>`)
	expectBodyContains(t, resp, `<form action="/admin/login" method="post"`)
}

func TestPostAdminLogin(t *testing.T) {
	body := body("email=admin&password=secret")
	isAdminFunc = alwaysAdminFunc
	header := http.Header{}
	header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp := doRequestWithHeader("POST", "/admin/login", body, &header)

	expectStatus(t, resp, 302)
	expectHeader(t, resp, "Location", "/admin/faqs")
	expectHeaderMatches(t, resp, "Set-Cookie", "^Authorization.*Path=/admin.*HttpOnly$")
}

func TestGetAdminFAQs(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/admin/faqs", emptyBody())

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / FAQs</title>`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/123"`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/456"`)
	expectBodyContains(t, resp, `href="/admin/faqs/edit/789"`)
}

func TestGetAdminFAQsNew(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/admin/faqs/new", emptyBody())

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / New FAQ</title>`)
	expectBodyContains(t, resp, `<form action="/admin/faqs/create" method="post">`)
}
func TestGetAdminFAQsEdit(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/admin/faqs/edit/123", emptyBody())

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / Edit FAQ</title>`)
	expectBodyContains(t, resp, `<form action="/admin/faqs/delete" method="post">`)
}

func TestGetAdminLocales(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/admin/locales", emptyBody())

	expectStatus(t, resp, 200)
	expectBodyContains(t, resp, `<title>Admin / Languages</title>`)

	expectBodyContains(t, resp, `<td>de</td>`)
	expectBodyContains(t, resp, `German (Deutsch)`)

	expectBodyContains(t, resp, `<td>es</td>`)
	expectBodyContains(t, resp, `Spanish (español)`)
}

func TestPostAdminFAQsCreate(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("POST", "/admin/faqs/create", emptyBody())

	expectStatus(t, resp, 302)
	expectHeader(t, resp, "Location", "/admin/faqs/edit/123")
}

func TestGetAPILanguages(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/api/languages", emptyBody())

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `[{"code":"en","name_en":"English","name_local":"English"},{"code":"de","name_en":"German","name_local":"Deutsch"},{"code":"fr","name_en":"French","name_local":"français"},{"code":"es","name_en":"Spanish","name_local":"español"},{"code":"it","name_en":"Italian","name_local":"italiano"},{"code":"nl","name_en":"Dutch","name_local":"Nederlands"},{"code":"pt","name_en":"Portuguese","name_local":"português"},{"code":"pt-BR","name_en":"Brazilian Portuguese","name_local":"português"},{"code":"da","name_en":"Danish","name_local":"dansk"},{"code":"sv","name_en":"Swedish","name_local":"svenska"},{"code":"no","name_en":"Norwegian Bokmål","name_local":"norsk bokmål"},{"code":"ru","name_en":"Russian","name_local":"русский"},{"code":"ar","name_en":"Arabic","name_local":"العربية"},{"code":"zh","name_en":"Chinese","name_local":"中文"}]`)
}

func TestGetAPIFAQs(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/api/faqs", emptyBody())

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `[{"id":123,"texts":null},{"id":456,"texts":null},{"id":789,"texts":null}]`)
}

func TestGetAPISingleFAQ(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/api/faqs/123", emptyBody())

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `{"id":123,"texts":[{"locale":{"code":"de"},"question":"Welcher Tag ist heute?","answer":"Freitag"}]}`)
}

func TestGetAPISearchFAQ(t *testing.T) {
	faqRepository = &mockDB{}
	resp := doRequest("GET", "/api/search-faqs?lang=en&query=bar", emptyBody())

	expectStatus(t, resp, 200)
	expectHeader(t, resp, "Content-Type", "application/json")
	expectBodyContains(t, resp, `[{"id":123,"texts":null},{"id":456,"texts":null},{"id":789,"texts":null}]`)
}

func TestConnectAndGetAll(t *testing.T) {
	repo := prepareDB()

	faqs, err := repo.AllFAQs()
	expectNoError(t, err)
	expectNoFAQs(t, faqs)
}

func TestSaveAndGet(t *testing.T) {
	repo := prepareDB()

	faqs, err := repo.AllFAQs()
	expectNoError(t, err)
	expectNoFAQs(t, faqs)

	f, err := repo.CreateFAQ()
	expectNoError(t, err)
	expectHasID(t, f.ID)
	expectNoTexts(t, f.Texts)

	txt := FAQText{Question: "question", Answer: "answer", Locale: Locale{Code: "en"}}
	err = repo.SaveFAQText(f.ID, &txt)
	expectNoError(t, err)

	f2, err := repo.FAQById(f.ID)
	expectNoError(t, err)
	expectSameID(t, f.ID, f2.ID)
	txt2 := f2.Texts[0]
	expectSameString(t, "en", txt2.Locale.Code)
	expectSameString(t, "question", txt2.Question)
	expectSameString(t, "answer", txt2.Answer)
}

func TestSaveAndDelete(t *testing.T) {
	repo := prepareDB()

	f, err := repo.CreateFAQ()
	expectNoError(t, err)
	expectHasID(t, f.ID)
	expectNoTexts(t, f.Texts)

	txt := FAQText{Question: "question", Answer: "answer", Locale: Locale{Code: "en"}}
	err = repo.SaveFAQText(f.ID, &txt)
	expectNoError(t, err)

	err = repo.DeleteFAQ(f.ID)
	expectNoError(t, err)

	f2, err := repo.FAQById(f.ID)
	expectNoError(t, err)
	expectSameID(t, f.ID, f2.ID)
	expectNoTexts(t, f.Texts)
}

func TestSimpleSearch(t *testing.T) {
	repo := prepareDB()

	f, err := repo.CreateFAQ()
	expectNoError(t, err)
	expectHasID(t, f.ID)
	expectNoTexts(t, f.Texts)

	txt := FAQText{Question: "question", Answer: "answer", Locale: Locale{Code: "en"}}
	err = repo.SaveFAQText(f.ID, &txt)
	expectNoError(t, err)

	// Failed search
	faqs, err := repo.SearchFAQs("de", "foobar")
	expectNoError(t, err)
	expectNoFAQs(t, faqs)

	// Successful search
	faqs, err = repo.SearchFAQs("en", "answer")
	expectNoError(t, err)

	t2 := faqs[0].Texts[0]
	expectSameString(t, "en", t2.Locale.Code)
	expectSameString(t, "question", t2.Question)
	expectSameString(t, "answer", t2.Answer)
}

func prepareDB() *DB {
	repo, err := NewDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	repo.ClearDB()
	repo.UpdateSearchIndex()
	return repo
}

func doRequest(method, uri string, body *bytes.Buffer) *httptest.ResponseRecorder {
	return doRequestWithHeader(method, uri, body, nil)
}

func doRequestWithHeader(method, uri string, body *bytes.Buffer, header *http.Header) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		panic(err)
	}
	if header != nil {
		req.Header = *header
	}

	router := buildRouter()
	router.ServeHTTP(resp, req)
	return resp
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

func expectNoError(t *testing.T, e error) {
	if e != nil {
		t.Errorf("expected no error but got: %v", e)
	}
}

func expectHasID(t *testing.T, id int) {
	if id <= 0 {
		t.Errorf("expected id > 0 but got: %v", id)
	}
}

func expectSameID(t *testing.T, id1 int, id2 int) {
	if id1 != id2 {
		t.Errorf("expected same ids, but got: id1=%v and id2=%v", id1, id2)
	}
}

func expectSameString(t *testing.T, str1 string, str2 string) {
	if str1 != str2 {
		t.Errorf("expected same strings, but got: str1=%v and str2=%v", str1, str2)
	}
}

func expectNoFAQs(t *testing.T, faqs []FAQ) {
	if len(faqs) != 0 {
		t.Errorf("expected empty slice but got: %v", faqs)
	}
}

func expectNoTexts(t *testing.T, texts []FAQText) {
	if len(texts) != 0 {
		t.Errorf("expected empty slice but got: %v", texts)
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

func alwaysAdminFunc(string, string) bool { return true }
