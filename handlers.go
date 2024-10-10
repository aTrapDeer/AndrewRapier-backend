package main

// handlers.go this is our CRUD operations for the database

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"bytes"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

const ANDREW_USER_ID = 1 // We'll use this to associate content with your account

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("../database/portfolio.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Auto Migrate the schema
	db.AutoMigrate(&User{}, &Website{}, &MusicWork{}, &Contribution{}, &Skill{}, &Education{})

	// Check if the admin user exists
	var userCount int64
	db.Model(&User{}).Where("id = ?", ANDREW_USER_ID).Count(&userCount)

	if userCount == 0 {
		log.Println("Admin user does not exist. Please create it manually or through a secure setup process.")
	}
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	var users []User
	db.Find(&users)
	json.NewEncoder(w).Encode(users)
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	json.NewDecoder(r.Body).Decode(&user)
	db.Create(&user)
	json.NewEncoder(w).Encode(user)
}

func GetContributions(w http.ResponseWriter, r *http.Request) {
	var contributions []Contribution
	db.Where("user_id = ?", ANDREW_USER_ID).Find(&contributions)
	json.NewEncoder(w).Encode(contributions)
}

func CreateContribution(w http.ResponseWriter, r *http.Request) {
	var contribution Contribution
	json.NewDecoder(r.Body).Decode(&contribution)
	contribution.UserID = ANDREW_USER_ID
	db.Create(&contribution)
	json.NewEncoder(w).Encode(contribution)
}

func GetMusicWorks(w http.ResponseWriter, r *http.Request) {
	var musicWorks []MusicWork
	db.Where("user_id = ?", ANDREW_USER_ID).Find(&musicWorks)
	json.NewEncoder(w).Encode(musicWorks)
}

func CreateMusicWork(w http.ResponseWriter, r *http.Request) {
	var musicWork MusicWork
	json.NewDecoder(r.Body).Decode(&musicWork)
	db.Create(&musicWork)
	musicWork.UserID = ANDREW_USER_ID
	json.NewEncoder(w).Encode(musicWork)
}

func GetWebsites(w http.ResponseWriter, r *http.Request) {
	data, err := getCachedData("websites", func() (interface{}, error) {
		log.Println("Fetching websites from database")
		var websites []Website
		result := db.Where("user_id = ?", ANDREW_USER_ID).Find(&websites)
		if result.Error != nil {
			return nil, result.Error
		}
		return websites, nil
	})

	if err != nil {
		log.Printf("Error fetching websites: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	websites, ok := data.([]Website)
	if !ok {
		http.Error(w, "Invalid data format", http.StatusInternalServerError)
		return
	}

	log.Printf("Returning %d websites", len(websites))
	json.NewEncoder(w).Encode(websites)
}

func CreateWebsite(w http.ResponseWriter, r *http.Request) {
	var website Website
	json.NewDecoder(r.Body).Decode(&website)
	website.UserID = ANDREW_USER_ID
	db.Create(&website)
	json.NewEncoder(w).Encode(website)

	// Trigger revalidation after successful creation
	go triggerRevalidation()
}

func GetSkills(w http.ResponseWriter, r *http.Request) {
	var skills []Skill
	db.Where("user_id = ?", ANDREW_USER_ID).Find(&skills)
	json.NewEncoder(w).Encode(skills)
}

func CreateSkill(w http.ResponseWriter, r *http.Request) {
	var skill Skill
	json.NewDecoder(r.Body).Decode(&skill)
	skill.UserID = ANDREW_USER_ID
	db.Create(&skill)
	json.NewEncoder(w).Encode(skill)
}

func DeleteSkill(w http.ResponseWriter, r *http.Request) {
	// Extract the skill ID from the URL
	skillID := r.URL.Query().Get("id")
	if skillID == "" {
		http.Error(w, "Skill ID is required", http.StatusBadRequest)
		return
	}

	log.Printf("Attempting to delete skill with ID: %s", skillID) // Add this log

	// Delete the skill from the database
	result := db.Delete(&Skill{}, skillID)
	if result.Error != nil {
		log.Printf("Error deleting skill: %v", result.Error) // Add this log
		http.Error(w, "Failed to delete skill", http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected (i.e., if the skill was actually deleted)
	if result.RowsAffected == 0 {
		http.Error(w, "Skill not found", http.StatusNotFound)
		return
	}

	// Return a success message
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Skill deleted successfully"})
}

func GetEducation(w http.ResponseWriter, r *http.Request) {
	var education []Education
	db.Where("user_id = ?", ANDREW_USER_ID).Find(&education)
	json.NewEncoder(w).Encode(education)
}

func CreateEducation(w http.ResponseWriter, r *http.Request) {
	var education Education
	json.NewDecoder(r.Body).Decode(&education)
	education.UserID = ANDREW_USER_ID
	db.Create(&education)
	json.NewEncoder(w).Encode(education)
}

func Login(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	frontendURL := os.Getenv("FRONTEND_URL")
	w.Header().Set("Access-Control-Allow-Origin", frontendURL)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var loginData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&loginData)

	var user User
	if err := db.Where("email = ?", loginData.Email).First(&user).Error; err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

func GetResourceById(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	resourceType := parts[1]
	id := parts[2]

	var result interface{}
	var err error

	switch resourceType {
	case "websites":
		var website Website
		err = db.First(&website, id).Error
		result = website
	case "music":
		var music MusicWork
		err = db.First(&music, id).Error
		result = music
	case "contributions":
		var contribution Contribution
		err = db.First(&contribution, id).Error
		result = contribution
	case "skills":
		var skill Skill
		err = db.First(&skill, id).Error
		result = skill
	case "education":
		var education Education
		err = db.First(&education, id).Error
		result = education
	default:
		http.Error(w, "Invalid resource type", http.StatusBadRequest)
		return
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Resource not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func UpdateResource(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	resourceType := parts[1]
	id := parts[2]

	var result interface{}
	var err error

	switch resourceType {
	case "websites":
		var website Website
		err = db.First(&website, id).Error
		if err == nil {
			json.NewDecoder(r.Body).Decode(&website)
			err = db.Save(&website).Error
			result = website
		}
	case "music":
		var musicWork MusicWork
		err = db.First(&musicWork, id).Error
		if err == nil {
			json.NewDecoder(r.Body).Decode(&musicWork)
			err = db.Save(&musicWork).Error
			result = musicWork
		}
	case "contributions":
		var contribution Contribution
		err = db.First(&contribution, id).Error
		if err == nil {
			json.NewDecoder(r.Body).Decode(&contribution)
			err = db.Save(&contribution).Error
			result = contribution
		}
	case "skills":
		var skill Skill
		err = db.First(&skill, id).Error
		if err == nil {
			json.NewDecoder(r.Body).Decode(&skill)
			err = db.Save(&skill).Error
			result = skill
		}
	case "education":
		var education Education
		err = db.First(&education, id).Error
		if err == nil {
			json.NewDecoder(r.Body).Decode(&education)
			err = db.Save(&education).Error
			result = education
		}
	default:
		http.Error(w, "Invalid resource type", http.StatusBadRequest)
		return
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Resource not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	// Trigger revalidation after successful update
	go triggerRevalidation()
}

func triggerRevalidation() {
	revalidationURL := os.Getenv("NEXT_REVALIDATION_URL")
	revalidationSecret := os.Getenv("REVALIDATION_SECRET")

	log.Printf("NEXT_REVALIDATION_URL: %s", revalidationURL)
	//log.Printf("REVALIDATION_SECRET: %s", revalidationSecret)

	// Add this check
	if revalidationURL == "" {
		log.Println("NEXT_REVALIDATION_URL is not set")
		return
	}

	payload := map[string]string{
		"secret": revalidationSecret,
	}

	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(revalidationURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error triggering revalidation: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Revalidation failed with status code: %d", resp.StatusCode)
	} else {
		log.Println("Revalidation triggered successfully")
	}
}
