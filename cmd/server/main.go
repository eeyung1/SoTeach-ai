// Command server runs the SoTeach tutoring engine and its thin HTTP API with
// the Stage 2 web client served from the same origin — one process, no CORS
// (workingReadme §3.2, §3.5, §8 Stage 2).
//
// Session state persists durably (Agent.md §20): a file-backed store by
// default, or PostgreSQL when -dsn is set. When an AI provider is configured
// via SOTEACH_AI_PROVIDER (+ SOTEACH_AI_API_KEY, optional SOTEACH_AI_MODEL),
// diagnosis is evaluated by that provider behind the ai.AIProvider boundary
// (Agent.md §38); otherwise the tutor records the learner's explanation
// verbatim.
//
// Run with: go run ./cmd/server — then open http://localhost:8080.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"soteach/ai/provider"
	"soteach/api"
	"soteach/session"
	"soteach/tutor"
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

	var tut *tutor.Tutor
	if name := os.Getenv("SOTEACH_AI_PROVIDER"); name != "" {
		p, err := provider.New(name, provider.Config{
			APIKey: os.Getenv("SOTEACH_AI_API_KEY"),
			Model:  os.Getenv("SOTEACH_AI_MODEL"),
		})
		if err != nil {
			log.Fatalf("configure AI provider %q: %v", name, err)
		}
		tut = tutor.NewAITutor(store, p)
		log.Printf("SoTeach using AI provider %q for diagnosis", name)
	} else {
		tut = tutor.NewTutor(store)
		log.Printf("SoTeach running without an AI provider (verbatim diagnosis)")
	}

	mux := http.NewServeMux()
	api.AddRoutesWithTutor(mux, store, tut)
	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	log.Printf("SoTeach listening on http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
