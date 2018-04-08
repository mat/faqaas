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
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

var db *sql.DB

type Locale struct {
	Code string `json:"code"`
	Name string `json:"name"`
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

func IndexNoLocale(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

func putLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var l Locale
	err := decoder.Decode(&l)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err = saveLocale(db, &l)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	} else {
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(l)
	}
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

	err = saveFAQ(db, &faq)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	} else {
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(faq)
	}
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

	err = deleteFAQ(db, &faq)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(Error{Error: err.Error()})
	}
}

func getLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := db.Ping(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	locales := supportedLocales

	// Write JSON result
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(locales)
}

func deleteLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	var l Locale
	err := decoder.Decode(&l)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	err = deleteLocale(db, &l)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func saveLocale(db *sql.DB, loc *Locale) error {
	_, err := language.Parse(loc.Code)
	if err != nil {
		return err
	}

	sqlStatement := `
		INSERT INTO locales (code,name) VALUES ($1, $2)
		 ON CONFLICT (code)
		 DO UPDATE SET name = EXCLUDED.name`
	_, err = db.Exec(sqlStatement, loc.Code, loc.Name)
	if err != nil {
		fmt.Print("DB ERR:", err)
	}
	return err
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

func saveFAQ(db *sql.DB, faq *FAQ) error {
	if faq.ID > 0 {
		return updateFAQ(db, faq)
	}
	return createFAQ(db, faq)
}

func createFAQ(db *sql.DB, faq *FAQ) error {
	// sqlStatement := `
	// 	INSERT INTO faqs (question,answer) VALUES ($1, $2)
	// 	RETURNING id`
	// err := db.QueryRow(sqlStatement, faq.Question, faq.Answer).Scan(&faq.ID)
	// if err != nil {
	// 	panic(err)
	// }
	// return err
	return nil
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

func deleteFAQ(db *sql.DB, faq *FAQ) error {
	sqlStatement := `
		DELETE FROM faqs WHERE id=$1`
	_, err := db.Exec(sqlStatement, faq.ID)
	return err
}

func deleteLocale(db *sql.DB, loc *Locale) error {
	sqlStatement := `DELETE FROM locales WHERE code = $1`
	_, err := db.Exec(sqlStatement, loc.Code)
	return err
}

func getAllLocales(db *sql.DB) ([]Locale, error) {
	rows, err := db.Query("SELECT id, code, name FROM locales;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	locales := []Locale{}
	for rows.Next() {
		var id int
		var code string
		var name string
		err = rows.Scan(&id, &code, &name)
		if err != nil {
			return nil, err
		}
		locales = append(locales, Locale{Code: code, Name: name})
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return locales, nil
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

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang/22892986#22892986
var letters = []rune("01234567890ABCDEFGHIJKLMNPQRSTUVWXYZ")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func createCategory(db *sql.DB) (*Category, error) {
	category := Category{Code: randomString(5)}

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
var tmplAdminLocales *template.Template
var tmplAdminFAQEdit *template.Template

func init() {
	tmplAdminFAQs = template.Must(template.ParseFiles("admin/templates/faqs.html"))
	tmplAdminLocales = template.Must(template.ParseFiles("admin/templates/locales.html"))
	tmplAdminFAQEdit = template.Must(template.ParseFiles("admin/templates/faqs_edit.html"))
}

func mustExecuteTemplate(tmpl *template.Template, wr io.Writer, data interface{}) {
	err := tmpl.Execute(wr, data)
	if err != nil {
		panic(err)
	}
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
		// Write JSON result
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(text)
	}
}

func getAdminLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	data := LocalesPageData{
		PageTitle: "Admin / Locales",
		Locales:   supportedLocales,
	}
	mustExecuteTemplate(tmplAdminLocales, w, data)
}

func main() {
	// connStr := "postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full"
	// connStr := fmt.Sprintf("postgres://mat:@localhost/faqaas?sslmode=disable",
	// host, port, user, dbname)
	// db, err := sql.Open("postgres", connStr)

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
	router.PUT("/api/locales", putLocales)
	router.DELETE("/api/locales", deleteLocales)

	router.GET("/api/categories", getCategories)
	router.POST("/api/categories", postCategories)

	router.GET("/api/faqs", getFAQs)
	router.POST("/api/faqs", postFAQs)
	router.DELETE("/api/faqs", deleteFAQs)

	router.GET("/admin/faqs", getAdminFAQs)
	router.GET("/admin/locales", getAdminLocales)
	router.GET("/admin/faqs/edit/:id", getAdminFAQsEdit)
	router.POST("/admin/faqs/update", postAdminFAQsUpdate)

	router.ServeFiles("/static/*filepath", http.Dir("public/static/"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port

	loggedRouter := handlers.CombinedLoggingHandler(os.Stdout, router)
	log.Fatal(http.ListenAndServe(addr, loggedRouter))
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
		fmt.Println(en.Name(tag))
		fmt.Println(display.Self.Name(tag))
		supportedLocales = append(supportedLocales, Locale{Code: code, Name: display.Self.Name(tag) + " (" + en.Name(tag) + ")"})
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
