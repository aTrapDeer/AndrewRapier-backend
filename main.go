// backend/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/patrickmn/go-cache"
	"github.com/rs/cors"
)

var c = cache.New(5*time.Minute, 10*time.Minute)

func getCachedData(key string, fetchFunc func() (interface{}, error)) (interface{}, error) {
	if data, found := c.Get(key); found {
		return data, nil
	}

	data, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	c.Set(key, data, cache.DefaultExpiration)
	return data, nil
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := bearerToken[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte("your-secret-key"), nil
		})

		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if _, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// The token is valid, proceed with the request
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
		}
	}
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
}

func main() {
	initDB()

	mux := http.NewServeMux()

	mux.HandleFunc("/login", Login)
	mux.HandleFunc("/users", GetUsers)
	mux.HandleFunc("/websites", GetWebsites)
	mux.HandleFunc("/music", GetMusicWorks)
	mux.HandleFunc("/contributions", GetContributions)
	mux.HandleFunc("/skills", GetSkills)
	mux.HandleFunc("/education", GetEducation)

	// Protect create routes with JWT auth
	mux.HandleFunc("/users/create", authMiddleware(CreateUser))
	mux.HandleFunc("/websites/create", authMiddleware(CreateWebsite))
	mux.HandleFunc("/music/create", authMiddleware(CreateMusicWork))
	mux.HandleFunc("/contributions/create", authMiddleware(CreateContribution))
	mux.HandleFunc("/skills/create", authMiddleware(CreateSkill))
	mux.HandleFunc("/education/create", authMiddleware(CreateEducation))

	// Add delete route for skills
	mux.HandleFunc("/skills/delete", authMiddleware(DeleteSkill))

	// Update these lines to use PUT method
	mux.HandleFunc("/websites/", authMiddleware(handleResource))
	mux.HandleFunc("/music/", authMiddleware(handleResource))
	mux.HandleFunc("/contributions/", authMiddleware(handleResource))
	mux.HandleFunc("/skills/", authMiddleware(handleResource))
	mux.HandleFunc("/education/", authMiddleware(handleResource))

	frontendURL := os.Getenv("FRONTEND_URL")
	frontendURL2 := os.Getenv("FRONTEND_URL2")

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins: []string{frontendURL, frontendURL2},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
	})
	handler := c.Handler(mux)

	fmt.Println("Golang backend running on port 8081")
	log.Fatal(http.ListenAndServe(":8081", handler))

	// New routes for individual resource editing
	mux.HandleFunc("/websites/", authMiddleware(UpdateResource))
	mux.HandleFunc("/music/", authMiddleware(UpdateResource))
	mux.HandleFunc("/contributions/", authMiddleware(UpdateResource))
	mux.HandleFunc("/skills/", authMiddleware(UpdateResource))
	mux.HandleFunc("/education/", authMiddleware(UpdateResource))
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		GetResourceById(w, r)
	case "PUT":
		UpdateResource(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
