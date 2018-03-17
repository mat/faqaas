package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"golang.org/x/text/language"
)

var db *sql.DB

const (
	host     = "localhost"
	port     = 5432
	user     = "mat"
	dbname   = "faqaas"
	password = ""
)

type Locale struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Category struct {
	Code string `json:"code"`
	// Name string `json:"name"`
}

type FAQ struct {
	Code     string `json:"code"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Error struct {
	Error string `json:"error"`
}

func redirectToFAQs(w http.ResponseWriter, r *http.Request, p httprouter.Params) {

	locales, err := getAllLocales(db)
	if err != nil {
		panic(err)
	}
	supported := []language.Tag{}
	for _, loc := range locales {
		tag, err := language.Parse(loc.Code)
		if err != nil {
			panic(nil)
		}
		supported = append(supported, tag)
	}

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
	var matcher = language.NewMatcher(supported)

	lang, _ := r.Cookie("lang")
	accept := r.Header.Get("Accept-Language")
	tag, _ := language.MatchStrings(matcher, lang.String(), accept)

	redirectURL := fmt.Sprintf("/faqs/%s", tag)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func getFAQsHTML(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
	fmt.Fprint(w, p.ByName("locale"))
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
	log.Println(l)

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
	log.Println(faq)

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

func getLocales(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if err := db.Ping(); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	locales, err := getAllLocales(db)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

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
	log.Println(l)

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

func saveFAQ(db *sql.DB, faq *FAQ) error {
	if len(faq.Code) == 0 {
		faq.Code = randomString(5)
	}

	sqlStatement := `
		INSERT INTO faqs (code,question,answer) VALUES ($1, $2, $3)
		 ON CONFLICT (code)
		 DO UPDATE SET question = EXCLUDED.question,
		 answer = EXCLUDED.answer`
	_, err := db.Exec(sqlStatement, faq.Code, faq.Question, faq.Answer)
	if err != nil {
		fmt.Print("DB ERR:", err)
	}
	return err
}

func deleteLocale(db *sql.DB, loc *Locale) error {
	sqlStatement := `DELETE FROM locales WHERE code = $1`
	_, err := db.Exec(sqlStatement, loc.Code)
	if err != nil {
		fmt.Print("DB ERR:", err)
	}
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
	rows, err := db.Query("SELECT id, code, question, answer FROM faqs;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	faqs := []FAQ{}
	for rows.Next() {
		var id int
		var code string
		var question string
		var answer string
		err = rows.Scan(&id, &code, &question, &answer)
		if err != nil {
			return nil, err
		}
		faqs = append(faqs, FAQ{Code: code, Question: question, Answer: answer})
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return faqs, nil
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

func main() {
	// connStr := "postgres://pqgotest:password@localhost/pqgotest?sslmode=verify-full"
	// connStr := fmt.Sprintf("postgres://mat:@localhost/faqaas?sslmode=disable",
	// host, port, user, dbname)
	// db, err := sql.Open("postgres", connStr)

	databaseURL := os.Getenv("DATABASE_URL")
	var err error
	if databaseURL != "" {
		db, err = sql.Open("postgres", databaseURL)
	} else {
		psqlInfo := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=disable",
			host, port, user, dbname)
		db, err = sql.Open("postgres", psqlInfo)
	}
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected!")

	locales, _ := getAllLocales(db)

	enc := json.NewEncoder(os.Stdout)
	enc.Encode(locales)

	router := httprouter.New()
	router.GET("/", redirectToFAQs)
	router.GET("/faqs/", redirectToFAQs)
	router.GET("/faqs/:locale", getFAQsHTML)

	router.GET("/api/locales", getLocales)
	router.PUT("/api/locales", putLocales)
	router.DELETE("/api/locales", deleteLocales)

	router.GET("/api/categories", getCategories)
	router.POST("/api/categories", postCategories)

	router.GET("/api/faqs", getFAQs)
	router.POST("/api/faqs", postFAQs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	log.Fatal(http.ListenAndServe(addr, router))
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
