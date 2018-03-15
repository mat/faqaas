package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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

type Error struct {
	Error string `json:"error"`
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)

	router.GET("/api/locales", getLocales)
	router.PUT("/api/locales", putLocales)
	router.DELETE("/api/locales", deleteLocales)

	router.GET("/api/categories", getCategories)
	// router.POST("/api/categories", postCategories)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := "0.0.0.0:" + port
	log.Fatal(http.ListenAndServe(addr, router))
}

func init() {

}
