package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `json:"username,omitempty"`
}

func main() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/postdb?sslmode=disable"
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	initDB()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	http.HandleFunc("/posts", postsHandler)
	http.HandleFunc("/feed", feedHandler)

	log.Printf("Post service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS posts (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL,
		username VARCHAR(50) DEFAULT '',
		content VARCHAR(320) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		log.Fatal(err)
	}
	// Migrate existing tables
	db.Exec(`ALTER TABLE posts ADD COLUMN IF NOT EXISTS username VARCHAR(50) DEFAULT ''`)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func postsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		userIDStr := r.URL.Query().Get("user_id")
		var rows *sql.Rows
		var err error
		if userIDStr != "" {
			userID, _ := strconv.Atoi(userIDStr)
			rows, err = db.Query("SELECT id, user_id, username, content, created_at FROM posts WHERE user_id = $1 ORDER BY created_at DESC", userID)
		} else {
			rows, err = db.Query("SELECT id, user_id, username, content, created_at FROM posts ORDER BY created_at DESC LIMIT 100")
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
			return
		}
		defer rows.Close()

		posts := []Post{}
		for rows.Next() {
			var p Post
			if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &p.CreatedAt); err == nil {
				posts = append(posts, p)
			}
		}
		writeJSON(w, http.StatusOK, posts)

	case http.MethodPost:
		userIDStr := r.Header.Get("X-User-ID")
		if userIDStr == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing user id"})
			return
		}
		userID, _ := strconv.Atoi(userIDStr)
		var req struct {
			Content  string `json:"content"`
			Username string `json:"username"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if len(req.Content) == 0 || len(req.Content) > 320 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content must be 1-320 characters"})
			return
		}

		// Prefer X-Username header from gateway; fallback to body
		username := r.Header.Get("X-Username")
		if username == "" {
			username = req.Username
		}

		var id int
		var createdAt time.Time
		err := db.QueryRow(
			"INSERT INTO posts (user_id, username, content) VALUES ($1, $2, $3) RETURNING id, created_at",
			userID, username, req.Content,
		).Scan(&id, &createdAt)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "user_id": userID, "username": username, "content": req.Content})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func feedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	idsStr := r.URL.Query().Get("user_ids")
	if idsStr == "" {
		writeJSON(w, http.StatusOK, []Post{})
		return
	}
	parts := strings.Split(idsStr, ",")
	ids := []int{}
	for _, p := range parts {
		if v, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			ids = append(ids, v)
		}
	}
	if len(ids) == 0 {
		writeJSON(w, http.StatusOK, []Post{})
		return
	}

	// Build parameterized query
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(
		"SELECT id, user_id, username, content, created_at FROM posts WHERE user_id IN (%s) ORDER BY created_at DESC LIMIT 200",
		strings.Join(placeholders, ","),
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	defer rows.Close()

	posts := []Post{}
	for rows.Next() {
		var p Post
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Content, &p.CreatedAt); err == nil {
			posts = append(posts, p)
		}
	}
	writeJSON(w, http.StatusOK, posts)
}
