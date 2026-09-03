// Command server runs the SoTeach tutoring engine and its thin HTTP API with
// the Stage 2 web client served from the same origin — one process, no CORS
// (workingReadme §3.2, §3.5, §8 Stage 2).
//
// Session state is persisted durably to a directory as JSON (Agent.md §20), so
// a session survives a server restart. Run it with: go run ./cmd/server — then
// open http://localhost:8080.
package main

import (
	"flag"
	"log"
	"net/http"

	"soteach/api"
	"soteach/session"
	"soteach/web"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dataDir := flag.String("data", "soteach-data", "directory for the durable session store")
	flag.Parse()

	store, err := session.NewFileStore(*dataDir)
	if err != nil {
		log.Fatalf("open session store: %v", err)
	}

	mux := http.NewServeMux()
	api.AddRoutes(mux, store)
	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	log.Printf("SoTeach listening on http://localhost%s (sessions in %s)", *addr, *dataDir)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
