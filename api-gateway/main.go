package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	userServiceURL string
	postServiceURL string
	jwtSecret      []byte
	client         = &http.Client{Timeout: 10 * time.Second}
)

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func main() {
	userServiceURL = os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8081"
	}
	postServiceURL = os.Getenv("POST_SERVICE_URL")
	if postServiceURL == "" {
		postServiceURL = "http://localhost:8082"
	}
	jwtSecret = []byte(os.Getenv("JWT_SECRET"))
	if len(jwtSecret) == 0 {
		jwtSecret = []byte("secret")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", corsHandler(proxyHandler(userServiceURL+"/register", false)))
	mux.HandleFunc("/api/login", corsHandler(proxyHandler(userServiceURL+"/login", false)))
	mux.HandleFunc("/api/users", corsHandler(proxyHandler(userServiceURL+"/users", false)))
	mux.HandleFunc("/api/users/", corsHandler(handleUserDetail))
	mux.HandleFunc("/api/follow", corsHandler(authProxyHandler(userServiceURL+"/follow", true)))
	mux.HandleFunc("/api/unfollow", corsHandler(authProxyHandler(userServiceURL+"/unfollow", true)))
	mux.HandleFunc("/api/posts", corsHandler(handlePosts))
	mux.HandleFunc("/api/feed", corsHandler(authWrapper(handleFeed)))

	log.Printf("API Gateway starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func corsHandler(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func proxyHandler(target string, withAuth bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if withAuth {
			authProxy(target, w, r)
			return
		}
		proxy(target, w, r)
	}
}

func authProxyHandler(target string, setUserID bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authProxy(target, w, r, setUserID)
	}
}

func proxy(target string, w http.ResponseWriter, r *http.Request) {
	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}
	req, err := http.NewRequest(r.Method, target, body)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	req.Header = r.Header.Clone()
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func authProxy(target string, w http.ResponseWriter, r *http.Request, setUserID ...bool) {
	userID, ok := validateToken(r)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var body io.Reader
	if r.Body != nil {
		body = r.Body
	}
	req, err := http.NewRequest(r.Method, target, body)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	req.Header = r.Header.Clone()
	if len(setUserID) > 0 && setUserID[0] {
		req.Header.Set("X-User-ID", strconv.Itoa(userID))
	}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "bad gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func validateToken(r *http.Request) (int, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return 0, false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return 0, false
	}
	tokenStr := parts[1]
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, false
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return 0, false
	}
	return claims.UserID, true
}

func authWrapper(next func(http.ResponseWriter, *http.Request, int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := validateToken(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		next(w, r, userID)
	}
}

func handleUserDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/users")
	target := userServiceURL + "/users" + path
	proxy(target, w, r)
}

func fetchUsername(userID int) string {
	url := fmt.Sprintf("%s/users/%d", userServiceURL, userID)
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var data struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return ""
	}
	return data.User.Username
}

func handlePosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		target := postServiceURL + "/posts"
		if r.URL.RawQuery != "" {
			target = target + "?" + r.URL.RawQuery
		}
		proxy(target, w, r)
	case http.MethodPost:
		userID, ok := validateToken(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}
		username := fetchUsername(userID)

		var body io.Reader
		if r.Body != nil {
			body = r.Body
		}
		req, err := http.NewRequest(r.Method, postServiceURL+"/posts", body)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		req.Header.Set("X-User-ID", strconv.Itoa(userID))
		req.Header.Set("X-Username", username)
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "bad gateway", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleFeed(w http.ResponseWriter, r *http.Request, userID int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Get following IDs from user-service
	url := fmt.Sprintf("%s/internal/following/%d", userServiceURL, userID)
	resp, err := client.Get(url)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch following"})
		return
	}
	defer resp.Body.Close()

	var followingIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&followingIDs); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to decode following"})
		return
	}

	// 2. Build user_ids query string
	idStrs := make([]string, len(followingIDs))
	for i, id := range followingIDs {
		idStrs[i] = strconv.Itoa(id)
	}
	userIDs := strings.Join(idStrs, ",")

	// 3. Fetch feed from post-service
	feedURL := fmt.Sprintf("%s/feed?user_ids=%s", postServiceURL, userIDs)
	resp2, err := client.Get(feedURL)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to fetch feed"})
		return
	}
	defer resp2.Body.Close()

	for k, vv := range resp2.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp2.StatusCode)
	io.Copy(w, resp2.Body)
}
