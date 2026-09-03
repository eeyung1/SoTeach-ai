// Command server runs the SoTeach tutoring engine and its thin HTTP API with
// the Stage 2 web client served from the same origin — one process, no CORS
// (workingReadme §3.2, §3.5, §8 Stage 2).
//
// Session state is in-memory (Agent.md §20: proven in memory first). Run it
// with: go run ./cmd/server  — then open http://localhost:8080.
package main

import (
	"log"
	"net/http"

	"soteach/api"
	"soteach/session"
	"soteach/web"
)

func main() {
	store := session.NewMemoryStore()

	mux := http.NewServeMux()
	api.AddRoutes(mux, store)
	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	addr := ":8080"
	log.Printf("SoTeach listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
