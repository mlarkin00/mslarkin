package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/donseba/go-htmx"
)

var (
	htmx_app     *htmx.HTMX
	htmx_handler *htmx.Handler
	// htmx_index    *htmx.Component

	html_template *template.Template
)

//go:embed templates/*
var templatesFS embed.FS

func main() {
	htmx_app = htmx.New()
	html_template = template.Must(template.New("").ParseFS(templatesFS, "templates/*"))
	log.Printf("Templates: %v", html_template.DefinedTemplates())

	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/clicked", handleClicked)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// htmx_handler = htmx_app.NewHandler(w, r)
	// htmx_handler.Render(r.Context(), htmx_index)
	html_template.ExecuteTemplate(w, "index", nil)
}

func handleClicked(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	htmx_handler = htmx_app.NewHandler(w, r)
	htmx_handler.Render(r.Context(), htmx.NewComponent("<div>I've been clicked!</div>"))
	fmt.Println("Clicked!")
}
