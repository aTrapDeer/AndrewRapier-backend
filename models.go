// models.go this is our database models
package main

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Name     string
	Email    string `gorm:"uniqueIndex"`
	Password string
}

type Website struct {
	gorm.Model
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Content     string `json:"content"` // For markdown content
}

type MusicWork struct {
	gorm.Model
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`     // For embedded videos or audio
	Content     string `json:"content"` // For markdown content
}

type Contribution struct {
	gorm.Model
	UserID      uint   `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Content     string `json:"content"` // For markdown content
}

type Skill struct {
	gorm.Model
	UserID      uint   `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Education struct {
	gorm.Model
	UserID       uint   `json:"user_id"`
	Institution  string `json:"institution"`
	Degree       string `json:"degree"`
	FieldOfStudy string `json:"field_of_study"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
	Description  string `json:"description"`
}
