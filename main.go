package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var db *sql.DB

type Locale struct {
	Code           string `json:"code"`
	Name           string `json:"name"`        // Name in English
	NameInLanguage string `json:"locale_name"` // Name in local language
}

func (l *Locale) IsDefaultLocale() bool {
	return l.Code == getDefaultLocale().Code
}

type Category struct {
	Code string `json:"code"`
	// Name string `json:"name"`
}

type FAQ struct {
	ID    int       `json:"id"`
	Texts []FAQText `json:"texts"`
}

func (f *FAQ) TextInDefaultLocale() FAQText {
	for _, t := range f.Texts {
		if t.Locale.IsDefaultLocale() {
			return t
		}
	}
	return FAQText{Locale: getDefaultLocale()}
}

type FAQText struct {
	iD       int
	Locale   Locale `json:"locale"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Error struct {
	Error string `json:"error"`
}

func redirectToFAQs(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	lang, _ := r.Cookie("lang")
	accept := r.Header.Get("Accept-Language")
	tag, _ := language.MatchStrings(languageMatcher, lang.String(), accept)

	redirectURL := fmt.Sprintf("/faqs/%s", tag)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func getFAQsHTML(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
	fmt.Fprint(w, p.ByName("locale"))
}

func getSingleFAQHTML(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
	fmt.Fprint(w, "locale=", p.ByName("locale"), "\n")
	fmt.Fprint(w, "id=", p.ByName("id"), "\n")

	idPart := p.ByName("id")
	parts := strings.Split(idPart, "-")
	lastPart := parts[len(parts)-1]
	id, err := strconv.Atoi(lastPart)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(w, "id=", id, "\n")
}

func postFAQs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var faq FAQ
	err := decoder.Decode(&faq)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
}

func deleteFAQs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var faq FAQ
	err := decoder.Decode(&faq)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err = deleteFAQ(db, faq.ID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	}
}

func getLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	locales := supportedLocales

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(locales)
}

func saveFAQText(db *sql.DB, faqID int, text *FAQText) error {
	sqlStatement := `
		INSERT INTO faq_texts (faq_id,locale,question,answer)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT ON CONSTRAINT texts_faq_id_locale
		  DO UPDATE SET
		   question = EXCLUDED.question,
		   answer = EXCLUDED.answer;
		`
	_, err := db.Exec(sqlStatement, faqID, text.Locale.Code, text.Question, text.Answer)
	if err != nil {
		fmt.Print("DB ERR:", err)
	}
	return err
}

func createFAQ(db *sql.DB) (*FAQ, error) {
	sqlStatement := `INSERT INTO faqs (question) VALUES (NULL) RETURNING id;`
	faq := FAQ{}
	err := db.QueryRow(sqlStatement).Scan(&faq.ID)
	if err != nil {
		panic(err)
	}
	return &faq, nil
}

func deleteFAQ(db *sql.DB, faqID int) error {
	sqlStatement := `DELETE FROM faq_texts WHERE faq_id = $1;`
	_, err := db.Exec(sqlStatement, faqID)
	if err != nil {
		return err
	}

	sqlStatement = `DELETE FROM faqs WHERE id = $1;`
	_, err = db.Exec(sqlStatement, faqID)
	return err
}

func updateFAQ(db *sql.DB, faq *FAQ) error {
	// sqlStatement := `
	// 	UPDATE faqs SET question=$1,answer=$2 WHERE id = $3`
	// _, err := db.Exec(sqlStatement, faq.Question, faq.Answer, faq.ID)
	// if err != nil {
	// 	panic(err)
	// }
	// return err
	return nil
}

func getAllCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT id, code FROM categories;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := []Category{}
	for rows.Next() {
		var id int
		var code string
		// var name string
		err = rows.Scan(&id, &code)
		if err != nil {
			return nil, err
		}
		categories = append(categories, Category{Code: code})
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return categories, nil
}

func getAllFAQs(db *sql.DB) ([]FAQ, error) {
	rows, err := db.Query("SELECT id FROM faqs ORDER BY id;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	faqs := []FAQ{}
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}

		faq := FAQ{ID: id}
		texts, err := getTextForFAQ(db, id)
		if err != nil {
			panic(err)
		}

		faq.Texts = texts
		faqs = append(faqs, faq)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return faqs, nil
}

func getTextForFAQ(db *sql.DB, faqID int) ([]FAQText, error) {
	rows, err := db.Query("SELECT id, locale, question, answer FROM faq_texts WHERE faq_id = $1;", faqID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	texts := []FAQText{}
	for rows.Next() {
		var id int
		var localeCode string
		var question string
		var answer string
		err = rows.Scan(&id, &localeCode, &question, &answer)
		if err != nil {
			return nil, err
		}
		l := Locale{Code: localeCode}
		t := FAQText{Locale: l, Question: question, Answer: answer}
		texts = append(texts, t)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return texts, nil
}

func getFAQ(db *sql.DB, id int) (*FAQ, error) {
	rows, err := db.Query("SELECT faq_texts.locale, faq_texts.question, faq_texts.answer FROM faqs, faq_texts WHERE faqs.id = $1 AND faq_texts.faq_id = $1;", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	faq := FAQ{ID: id}
	faq.Texts = []FAQText{}
	for rows.Next() {
		var locale string
		var question string
		var answer string
		err = rows.Scan(&locale, &question, &answer)
		if err != nil {
			return nil, err
		}
		loc := Locale{Code: locale}
		faq.Texts = append(faq.Texts, FAQText{Locale: loc, Question: question, Answer: answer})
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return &faq, nil
}

func getCategories(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	categories, err := getAllCategories(db)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Write JSON result
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(categories)
}

func getFAQs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	faqs, err := getAllFAQs(db)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Write JSON result
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(faqs)
}

func postCategories(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	category, err := createCategory(db)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Write JSON result
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(category)
}

func createCategory(db *sql.DB) (*Category, error) {
	category := Category{Code: "deadbeef"}

	sqlStatement := `
		INSERT INTO categories (code) VALUES ($1)`
	_, err := db.Exec(sqlStatement, category.Code)
	if err != nil {
		fmt.Print("DB ERR:", err)
	}

	return &category, err
}

type FAQsPageData struct {
	PageTitle string
	Locales   []Locale
	FAQs      []FAQ
}

type FAQsNewPageData struct {
	PageTitle     string
	DefaultLocale Locale
}

type FAQEditPageData struct {
	PageTitle string
	Locales   []Locale
	FAQ       FAQ
}

type LocalesPageData struct {
	PageTitle string
	Locales   []Locale
}

var tmplAdminFAQs *template.Template
var tmplAdminFAQsNew *template.Template
var tmplAdminFAQEdit *template.Template
var tmplAdminLocales *template.Template
var tmplAdminLogin *template.Template

func init() {
	tmplAdminFAQs = template.Must(template.ParseFiles("admin/templates/faqs.html"))
	tmplAdminFAQsNew = template.Must(template.ParseFiles("admin/templates/faqs_new.html"))
	tmplAdminFAQEdit = template.Must(template.ParseFiles("admin/templates/faqs_edit.html"))
	tmplAdminLocales = template.Must(template.ParseFiles("admin/templates/locales.html"))
	tmplAdminLogin = template.Must(template.ParseFiles("admin/templates/login.html"))
}

func mustExecuteTemplate(tmpl *template.Template, wr io.Writer, data interface{}) {
	err := tmpl.Execute(wr, data)
	if err != nil {
		panic(err)
	}
}

func createJWT(expiry time.Time) string {
	key := []byte(jwtKey)
	sig, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT"))
	if err != nil {
		panic(err)
	}

	cl := jwt.Claims{
		Subject: "admin",
		// Issuer:    "issuer",
		// NotBefore: jwt.NewNumericDate(time.Date(2016, 1, 1, 0, 0, 0, 0, time.UTC)),
		Expiry: jwt.NewNumericDate(expiry),
		// Audience:  jwt.Audience{"leela", "fry"},
	}
	raw, err := jwt.Signed(sig).Claims(cl).CompactSerialize()
	if err != nil {
		panic(err)
	}

	return raw
}

var jwtKey string
var adminPasswordHash string

func init() {
	jwtKey = os.Getenv("JWT_KEY")
	if len(jwtKey) == 0 {
		panic("JWT_KEY not set")
	}

	adminPasswordHash = os.Getenv("ADMIN_PASSWORD")
	if len(adminPasswordHash) == 0 {
		panic("ADMIN_PASSWORD not set")
	}
}

const (
	// leeway for matching NotBefore/Expiry claims.
	leeway = 1.0 * time.Minute
)

func validateJWT(rawToken string) bool {
	tok, err := jwt.ParseSigned(rawToken)
	if err != nil {
		return false
	}

	key := []byte(jwtKey)

	cl := jwt.Claims{}
	if err := tok.Claims(key, &cl); err != nil {
		return false
	}

	err = cl.ValidateWithLeeway(jwt.Expected{
		Subject: "admin",
		Time:    time.Now(),
		// Issuer:  "issuer",
	}, leeway)
	if err != nil {
		return false
	}

	return true
}

func getAdmin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	http.Redirect(w, r, "/admin/faqs", http.StatusFound)
}

func loggedInAsAdmin(r *http.Request) bool {
	authCookie, err := r.Cookie("Auth")
	if err != nil {
		return false
	}
	fmt.Println("AUTH cookie:")
	val := authCookie.Value
	fmt.Println(val)

	isValid := validateJWT(val)
	if isValid {
		fmt.Printf("valid!")
		return true
	} else {
		fmt.Printf("NOT valid!!")
		return false
	}
}

func redirectToAdminLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func getAdminFAQs(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	faqs, err := getAllFAQs(db)
	if err != nil {
		panic(err)
	}
	data := FAQsPageData{
		PageTitle: "Admin / FAQs",
		FAQs:      faqs,
	}
	mustExecuteTemplate(tmplAdminFAQs, w, data)
}

func getAdminLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data := FAQsPageData{
		PageTitle: "Admin / Login",
		// FAQs:      faqs,
	}
	mustExecuteTemplate(tmplAdminLogin, w, data)
}

func getAdminFAQsNew(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data := FAQsNewPageData{
		PageTitle:     "Admin / New FAQ",
		DefaultLocale: getDefaultLocale(),
	}
	mustExecuteTemplate(tmplAdminFAQsNew, w, data)
}

func getAdminFAQsEdit(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	idStr := ps.ByName("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		panic(err)
	}

	faq, err := getFAQ(db, id)
	if err != nil {
		panic(err)
	}

	m := make(map[string]FAQText)
	for _, text := range faq.Texts {
		m[text.Locale.Code] = text
	}

	faq.Texts = []FAQText{}
	for _, loc := range supportedLocales {
		t := FAQText{Locale: loc}
		t2, ok := m[loc.Code]
		if ok {
			t2.Locale = t.Locale
			t = t2
		}
		faq.Texts = append(faq.Texts, t)
	}

	data := FAQEditPageData{
		PageTitle: "Admin / Edit FAQ",
		FAQ:       *faq,
	}
	mustExecuteTemplate(tmplAdminFAQEdit, w, data)
}

type faqForm struct {
	faqID      string
	localeCode string
	question   string
	answer     string
}

func postAdminFAQsUpdate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	form := faqForm{
		faqID:      r.FormValue("faqID"),
		localeCode: r.FormValue("localeCode"),
		question:   r.FormValue("question"),
		answer:     r.FormValue("answer"),
	}

	loc := Locale{Code: form.localeCode}
	text := FAQText{Question: form.question, Answer: form.answer, Locale: loc}

	faqID, err := strconv.Atoi(form.faqID)
	if err != nil {
		panic(err)
	}

	err = saveFAQText(db, faqID, &text)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	} else {
		redirectURL := fmt.Sprintf("/admin/faqs/edit/%d", faqID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

func postAdminFAQsDelete(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	form := faqForm{
		faqID: r.FormValue("faqID"),
	}

	faqID, err := strconv.Atoi(form.faqID)
	if err != nil {
		panic(err)
	}

	err = deleteFAQ(db, faqID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	} else {
		http.Redirect(w, r, "/admin/faqs", http.StatusFound)
	}
}

func postAdminFAQsCreate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	form := faqForm{
		localeCode: r.FormValue("localeCode"),
		question:   r.FormValue("question"),
		answer:     r.FormValue("answer"),
	}

	loc := Locale{Code: form.localeCode}
	text := FAQText{Question: form.question, Answer: form.answer, Locale: loc}

	faq, err := createFAQ(db)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
		return
	}

	err = saveFAQText(db, faq.ID, &text)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	} else {
		redirectURL := fmt.Sprintf("/admin/faqs/edit/%d", faq.ID)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

func getAdminLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data := LocalesPageData{
		PageTitle: "Admin / Locales",
		Locales:   supportedLocales,
	}
	mustExecuteTemplate(tmplAdminLocales, w, data)
}

func postAdminLogin(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	email := r.FormValue("email")
	password := r.FormValue("password")

	if email == "admin" && isAdminPassword(password) {
		fmt.Println("Logged in as admin!")
		setAuthCookie(w)
		http.Redirect(w, r, "/admin/faqs", http.StatusFound)
	} else {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
}

func isAdminPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(adminPasswordHash), []byte(password))
	return err == nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

const (
	adminSessionDuration = 1.0 * time.Hour
)

func setAuthCookie(w http.ResponseWriter) {
	expires := time.Now().Add(adminSessionDuration)

	ck := http.Cookie{
		Name:  "Auth",
		Value: createJWT(expires),
		// Domain:  "foo.com",
		// Path:    "/",
		Expires: expires,
	}

	http.SetCookie(w, &ck)
}

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		var err error
		db, err = sql.Open("postgres", databaseURL)
		if err != nil {
			panic(err)
		}
	} else {
		panic("DATABASE_URL not set")
	}
	defer db.Close()

	err := db.Ping()
	if err != nil {
		panic(err)
	}

	router := httprouter.New()
	router.GET("/", redirectToFAQs)
	router.GET("/faqs/", redirectToFAQs)
	router.GET("/faqs/:locale", getFAQsHTML)
	router.GET("/faq/:locale/:id", getSingleFAQHTML)

	router.GET("/api/locales", getLocales)

	router.GET("/api/categories", getCategories)
	router.POST("/api/categories", postCategories)

	router.GET("/api/faqs", getFAQs)
	router.POST("/api/faqs", postFAQs)
	router.DELETE("/api/faqs", deleteFAQs)

	router.GET("/admin", httpsOnly(adminPassword(getAdmin)))
	router.GET("/admin/faqs", httpsOnly(adminPassword(getAdminFAQs)))
	router.GET("/admin/locales", httpsOnly(adminPassword(getAdminLocales)))
	router.GET("/admin/faqs/edit/:id", httpsOnly(adminPassword(getAdminFAQsEdit)))
	router.GET("/admin/faqs/new", httpsOnly(adminPassword(getAdminFAQsNew)))
	router.POST("/admin/faqs/update", httpsOnly(adminPassword(postAdminFAQsUpdate)))
	router.POST("/admin/faqs/create", httpsOnly(adminPassword(postAdminFAQsCreate)))
	router.POST("/admin/faqs/delete", httpsOnly(adminPassword(postAdminFAQsDelete)))
	router.GET("/admin/login", httpsOnly(getAdminLogin))
	router.POST("/admin/login", httpsOnly(postAdminLogin))

	router.ServeFiles("/static/*filepath", http.Dir("public/static/"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port

	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, router)
	log.Fatal(http.ListenAndServe(addr, loggedRouter))
}

func adminPassword(h httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if !loggedInAsAdmin(r) {
			redirectToAdminLogin(w, r)
			return
		} else {
			h(w, r, ps)
		}
	}
}

func httpsOnly(h httprouter.Handle) httprouter.Handle {
	if os.Getenv("HTTP_ALLOWED") == "true" {
		return h
	}

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		if r.Header.Get("X-Forwarded-Proto") == "http" {
			targetURL := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
			http.Redirect(w, r, targetURL.String(), http.StatusFound)
		} else {
			h(w, r, ps)
		}
	}
}

var languageMatcher language.Matcher
var supportedLocales []Locale

func init() {
	locales := strings.Split(os.Getenv("SUPPORTED_LOCALES"), ",")

	supportedLocales = []Locale{}
	en := display.English.Tags()
	for _, code := range locales {
		tag, err := language.Parse(code)
		if err != nil {
			panic(nil)
		}
		locale := Locale{Code: code, Name: en.Name(tag), NameInLanguage: display.Self.Name(tag)}
		supportedLocales = append(supportedLocales, locale)
	}
	if len(supportedLocales) == 0 {
		panic("SUPPORTED_LOCALES missing or wrong")
	}

	languageMatcher = buildLanguageMatcher()
	rand.Seed(time.Now().UnixNano())
}

func getDefaultLocale() Locale {
	return supportedLocales[0]
}

func buildLanguageMatcher() language.Matcher {
	supportedLocales := parseLocales(os.Getenv("SUPPORTED_LOCALES"))
	return language.NewMatcher(supportedLocales)
}

func parseLocales(supportedLocales string) []language.Tag {
	locales := strings.Split(supportedLocales, ",")

	supported := []language.Tag{}
	for _, loc := range locales {
		tag, err := language.Parse(loc)
		if err != nil {
			panic(nil)
		}
		supported = append(supported, tag)
	}

	return supported

	// var supported = []language.Tag{
	// 	language.AmericanEnglish,    // en-US: first language is fallback
	// 	language.German,             // de
	// 	language.Dutch,              // nl
	// 	language.Portuguese,         // pt (defaults to Brazilian)
	// 	language.EuropeanPortuguese, // pt-pT
	// 	language.Romanian,           // ro
	// 	language.Serbian,            // sr (defaults to Cyrillic script)
	// 	language.SerbianLatin,       // sr-Latn
	// 	language.SimplifiedChinese,  // zh-Hans
	// 	language.TraditionalChinese, // zh-Hant
	// }
}
