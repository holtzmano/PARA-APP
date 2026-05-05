package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"para/internal/store"
)

func main() {
	const dbPath = "./para.db"
	s, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer s.Close()
	log.Printf("database opened at %s", dbPath)

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello para"))
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
