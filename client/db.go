package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
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
	SoroushUserID int64     `json:"soroushUserId"`
	AccessHash    int64     `json:"accessHash"`
	DisplayName   string    `json:"displayName"`
	AuthKey       []byte    `json:"-"`
	AuthKeyID     []byte    `json:"-"`
	ServerSalt    []byte    `json:"-"`
	SessionData   string    `json:"-"`
	DcID          int       `json:"dcId"`
	Status        string    `gorm:"default:'idle'" json:"status"`
	LastActive    string    `json:"lastActive"`
	CreatedAt     time.Time `json:"createdAt"`
}

// DBTunnelConfig stores the tunnel configuration (group bus + legacy dispatcher)
type DBTunnelConfig struct {
	ID                   uint   `gorm:"primaryKey" json:"id"`
	GroupChatID          int64  `json:"groupChatId"`          // "My lovely family" group chat ID
	GroupAccessHash      int64  `json:"groupAccessHash"`      // Access hash for channel/supergroup
	PSK                  string `json:"psk"`                  // Pre-shared key for stealth encoding
	DispatcherUserID     int64  `json:"dispatcherUserId"`     // Legacy: direct dispatcher
	DispatcherAccessHash int64  `json:"dispatcherAccessHash"` // Legacy: direct dispatcher
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
	err = db.AutoMigrate(&DBAdmin{}, &DBSoroushAccount{}, &DBTunnelConfig{}, &DBLogEntry{})
	if err != nil {
		log.Fatalf("[DB] Database migration failed: %v", err)
	}
	fmt.Println("[DB] Tables migrated successfully.")

	// Seed Admin user
	seedAdmin()
}

// Seed the default admin credential (salman / ADMIN_PASSWORD env or random/fallback)
func seedAdmin() {
	var count int64
	db.Model(&DBAdmin{}).Count(&count)
	if count == 0 {
		adminUser := os.Getenv("ADMIN_USERNAME")
		if adminUser == "" {
			adminUser = "salman"
		}
		adminPass := os.Getenv("ADMIN_PASSWORD")
		if adminPass == "" {
			bytes := make([]byte, 8)
			if _, err := rand.Read(bytes); err == nil {
				adminPass = fmt.Sprintf("%x", bytes)
				log.Printf("[DB] WARNING: ADMIN_PASSWORD env variable not set. Generated random admin password: %s\n", adminPass)
			} else {
				adminPass = "136517"
				log.Println("[DB] WARNING: Failed to generate random password, using fallback '136517'")
			}
		} else {
			log.Printf("[DB] Seeding admin user '%s' using ADMIN_PASSWORD from environment\n", adminUser)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("[DB] Failed to hash password: %v", err)
		}

		admin := DBAdmin{
			Username:     adminUser,
			PasswordHash: string(hashedPassword),
			CreatedAt:    time.Now(),
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Fatalf("[DB] Failed to seed admin user: %v", err)
		}
		fmt.Printf("[DB] Successfully seeded admin user (%s / %s)\n", adminUser, adminPass)
	} else {
		fmt.Println("[DB] Admin credentials already seeded.")
	}
}
