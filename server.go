package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
)

func _() {
	indexPath := os.Getenv("BLEVE_INDEX_PATH")

	index, err := bleve.Open(indexPath)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		t, _ := template.ParseFiles("templates/index.html")
		t.Execute(w, nil)
	})
	mux.HandleFunc("POST /search", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		t, _ := template.ParseFiles("templates/results.html")

		r.ParseForm()

		search, ok := r.PostForm["search"]
		if !ok || len(search) != 1 {
			http.Error(w, "search form parameter is required", http.StatusBadRequest)
		} else {
			// query := bleve.NewMatchQuery(search[0])
			query := bleve.NewFuzzyQuery(search[0])
			searchRequest := bleve.NewSearchRequest(query)
			searchRequest.Fields = []string{"key", "summary"}
			searchRequest.IncludeLocations = true
			searchRequest.Explain = true
			searchResult, err := index.Search(searchRequest)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			log.Println("Total: ", searchResult.Total)
			for _, hit := range searchResult.Hits {
				log.Println(hit.ID, hit.Expl)
				// for k, _ := range hit.Locations {
				// 	log.Println(k)
				// }
				// for term, fieldMap := range hit.Locations {
				// 	for field, locations := range fieldMap {
				// 		log.Printf("Term '%s' found in field '%s':\n", term, field)
				// 		for _, loc := range locations {
				// 			log.Printf("→ Starts at byte %d, ends at byte %d (position %d)\n",
				// 				loc.Start, loc.End, loc.Pos)
				// 		}
				// 	}
				// }
			}
			log.Println(t.Execute(w, searchResult))
		}
		// params, _ := url.ParseQuery(r.URL.RawQuery)
		// s, ok := params["s"]
		// if ok {
		// 	// http.Error(w, "", http.StatusInternalServerError)
		// 	query := bleve.NewMatchQuery(s[0])
		// 	search := bleve.NewSearchRequest(query)
		// 	searchResults, err := index.Search(search)
		// 	if err != nil {
		// 		http.Error(w, err.Error(), http.StatusInternalServerError)
		// 	}
		// 	t.Execute(w, searchResults)
		// } else {
		// }

		// data := struct{ SearchResuls []Issue }{SearchResuls}
		// t.Execute(w, data)
	})

	assetsFS := os.DirFS("assets")
	mux.HandleFunc("GET /assets/{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/assets/")
		if path == "" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, assetsFS, path)
	})

	s := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Println("Server is running at http://localhost:8080")
	log.Fatal(s.ListenAndServe())
}
