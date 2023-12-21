package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func NewTemplates() *Templates {
	return &Templates{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
}

func DB() *sql.DB {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbUrl := os.Getenv("LOCAL_DB")
	db, err := sql.Open("sqlite3", dbUrl)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open db %s: %s", dbUrl, err)
		os.Exit(1)
	}
	createTable(db)
	return db
}

func createTable(db *sql.DB) {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			cpi FLOAT NOT NULL
		)
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("failed to create table: %s", err)
	}
}

func main() {
	db := DB()

	// Insert sample entry into the table
	insertSampleEntry(db)

	e := echo.New()
	e.Renderer = NewTemplates()
	e.Use(middleware.Logger())
	e.GET("/crow", func(c echo.Context) error {
		return c.String(http.StatusOK, "crow")
	})
	e.GET("/", func(c echo.Context) error {
		data := 7
		return c.Render(http.StatusOK, "index", data)
	})
	e.Logger.Fatal(e.Start(":7878"))
}

func insertSampleEntry(db *sql.DB) {
	query := `
		INSERT INTO users (email, cpi)
		VALUES ('student@study.com', 7.8)
		ON CONFLICT DO NOTHING
	`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("failed to insert sample entry: %s", err)
	}
}
