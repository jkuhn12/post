package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB
var jwtSecret []byte

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func main() {
	var err error
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/userdb?sslmode=disable"
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("secret")
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
		port = "8081"
	}

	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/users", usersHandler)
	http.HandleFunc("/users/", userDetailHandler)
	http.HandleFunc("/follow", followHandler)
	http.HandleFunc("/unfollow", unfollowHandler)
	http.HandleFunc("/internal/following/", internalFollowingHandler)

	log.Printf("User service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(100) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS follows (
			follower_id INT REFERENCES users(id) ON DELETE CASCADE,
			following_id INT REFERENCES users(id) ON DELETE CASCADE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (follower_id, following_id)
		)`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			log.Fatal(err)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing fields"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	var userID int
	err = db.QueryRow(
		"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id",
		req.Username, req.Email, string(hash),
	).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username or email taken"})
		return
	}

	// Auto-follow yourself so your own posts appear in your feed
	db.Exec("INSERT INTO follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, userID)

	writeJSON(w, http.StatusCreated, map[string]any{"id": userID, "username": req.Username})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	var id int
	var hash string
	var username string
	err := db.QueryRow("SELECT id, username, password_hash FROM users WHERE username = $1", req.Username).Scan(&id, &username, &hash)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	})
	tokenStr, err := token.SignedString(jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": tokenStr, "user_id": id, "username": username})
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rows, err := db.Query("SELECT id, username, email, created_at FROM users ORDER BY created_at DESC")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	writeJSON(w, http.StatusOK, users)
}

func userDetailHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	idStr := parts[1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var u User
		err := db.QueryRow("SELECT id, username, email, created_at FROM users WHERE id = $1", id).
			Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "user not found"})
			return
		}

		var followerCount, followingCount int
		db.QueryRow("SELECT COUNT(*) FROM follows WHERE following_id = $1", id).Scan(&followerCount)
		db.QueryRow("SELECT COUNT(*) FROM follows WHERE follower_id = $1", id).Scan(&followingCount)

		writeJSON(w, http.StatusOK, map[string]any{
			"user":             u,
			"followers_count":  followerCount,
			"following_count": followingCount,
		})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func followHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FollowerID  int `json:"follower_id"`
		FollowingID int `json:"following_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	_, err := db.Exec("INSERT INTO follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", req.FollowerID, req.FollowingID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "followed"})
}

func unfollowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FollowerID  int `json:"follower_id"`
		FollowingID int `json:"following_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	_, err := db.Exec("DELETE FROM follows WHERE follower_id = $1 AND following_id = $2", req.FollowerID, req.FollowingID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "unfollowed"})
}

func internalFollowingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rows, err := db.Query("SELECT following_id FROM follows WHERE follower_id = $1", id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var fid int
		if err := rows.Scan(&fid); err == nil {
			ids = append(ids, fid)
		}
	}
	writeJSON(w, http.StatusOK, ids)
}
