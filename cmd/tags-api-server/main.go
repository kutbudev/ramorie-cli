// main.go - Tags API Server
package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"strings"

	"github.com/terzigolu/josepshbrain-go/handlers"
	"github.com/terzigolu/josepshbrain-go/repository"

	_ "github.com/lib/pq"
)

func main() {
	// Database connection string from environment variable or default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	tagRepo := repository.NewTagRepository(db)
	tagHandler := handlers.NewTagHandler(tagRepo)

	http.HandleFunc("/tags", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			tagHandler.GetTags(w, r)
		case http.MethodPost:
			tagHandler.CreateTag(w, r)
		case http.MethodPut:
			tagHandler.UpdateTag(w, r)
		case http.MethodDelete:
			tagHandler.DeleteTag(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// --- Task Notes Endpoints ---
	http.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		// /tasks/{taskId}/notes and /tasks/{taskId}/notes/{noteId}
		if strings.HasSuffix(path, "/notes") {
			switch r.Method {
			case http.MethodGet:
				handlers.ListNotes(w, r)
			case http.MethodPost:
				handlers.AddNote(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		if strings.Contains(path, "/notes/") {
			switch r.Method {
			case http.MethodPut:
				handlers.UpdateNote(w, r)
			case http.MethodDelete:
				handlers.DeleteNote(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		// /tasks/{taskId}/ai/{feature}
		if strings.Contains(path, "/ai/") {
			switch {
			case strings.HasSuffix(path, "/ai/analysis"):
				handlers.AIAnalysis(w, r)
			case strings.HasSuffix(path, "/ai/decompose"):
				handlers.AIDecompose(w, r)
			case strings.HasSuffix(path, "/ai/priority"):
				handlers.AIPriority(w, r)
			case strings.HasSuffix(path, "/ai/elaborate"):
				handlers.AIElaborate(w, r)
			case strings.HasSuffix(path, "/ai/suggestions"):
				handlers.AISuggestions(w, r)
			default:
				http.Error(w, "Not found", http.StatusNotFound)
			}
			return
		}
		http.NotFound(w, r)
	})

	// Add bulk update endpoint
	http.HandleFunc("/tasks/bulk-update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handlers.BulkUpdateTasks(w, r)
	})

	log.Println("Tags API server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
