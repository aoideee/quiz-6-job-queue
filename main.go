package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/aoideee/quiz-job-queue/internal/db"
	"github.com/aoideee/quiz-job-queue/internal/handler"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://jobs:jobs@localhost:5432/jobs?sslmode=disable"
	}

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	log.Println("connected to database")

	store := db.NewStore(database)
	h := handler.New(store)

	mux := http.NewServeMux()

	// Job lifecycle endpoints
	mux.HandleFunc("POST /jobs", h.CreateJob)
	mux.HandleFunc("GET /jobs/{id}", h.GetJob)
	mux.HandleFunc("PATCH /jobs/{id}", h.UpdateJob)

	// Worker endpoint — claims the next pending job atomically
	mux.HandleFunc("GET /jobs/next", h.ClaimNextJob)

	// SSE stream — pushes updates whenever a job's updated_at changes
	mux.HandleFunc("GET /jobs/stream", h.StreamUpdates)

	addr := ":3000"
	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
