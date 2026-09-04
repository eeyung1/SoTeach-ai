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
	dataDir := flag.String("data", "soteach-data", "directory for the file-backed session store")
	dsn := flag.String("dsn", "", "PostgreSQL DSN; when set the durable store is PostgreSQL instead of the file store")
	flag.Parse()

	var store session.Store
	if *dsn != "" {
		pg, err := session.NewPostgresStore(*dsn)
		if err != nil {
			log.Fatalf("open postgres store: %v", err)
		}
		defer pg.Close()
		store = pg
		log.Printf("SoTeach using PostgreSQL session store")
	} else {
		f, err := session.NewFileStore(*dataDir)
		if err != nil {
			log.Fatalf("open session store: %v", err)
		}
		store = f
		log.Printf("SoTeach using file session store (%s)", *dataDir)
	}

	mux := http.NewServeMux()
	api.AddRoutes(mux, store)
	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	log.Printf("SoTeach listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
