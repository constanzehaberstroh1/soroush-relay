package main

import (
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// GORM SQLite Database handle
var db *gorm.DB

// Admin Model
type DBAdmin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SoroushAccount Model
type DBSoroushAccount struct {
	ID            string    `gorm:"primaryKey" json:"id"`
	PhoneNumber   string    `gorm:"uniqueIndex;not null" json:"phoneNumber"`
	Name          string    `gorm:"not null" json:"name"`
	SoroushUserID string    `gorm:"not null" json:"soroushUserId"`
	SessionToken  string    `gorm:"not null" json:"sessionToken"`
	Status        string    `gorm:"default:'idle'" json:"status"`
	LastActive    string    `json:"lastActive"`
	CreatedAt     time.Time `json:"createdAt"`
}

// Initialize SQLite database
func initDB() {
	var err error
	// Use CGO-free modernc sqlite via glebarez
	db, err = gorm.Open(sqlite.Open("client_config.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("[DB] Failed to connect to SQLite database: %v", err)
	}

	fmt.Println("[DB] SQLite database initialized successfully: client_config.db")

	// Auto migrate tables
	err = db.AutoMigrate(&DBAdmin{}, &DBSoroushAccount{})
	if err != nil {
		log.Fatalf("[DB] Database migration failed: %v", err)
	}
	fmt.Println("[DB] Tables migrated successfully.")

	// Seed Admin user
	seedAdmin()
}

// Seed the default admin credential (salman / 136517)
func seedAdmin() {
	var count int64
	db.Model(&DBAdmin{}).Count(&count)
	if count == 0 {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("136517"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("[DB] Failed to hash password: %v", err)
		}

		admin := DBAdmin{
			Username:     "salman",
			PasswordHash: string(hashedPassword),
			CreatedAt:    time.Now(),
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Fatalf("[DB] Failed to seed default admin user: %v", err)
		}
		fmt.Println("[DB] Successfully seeded default admin user (salman / 136517)")
	} else {
		fmt.Println("[DB] Admin credentials already seeded.")
	}
}
