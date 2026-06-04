package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// GORM Database handle (points to MySQL)
var db *gorm.DB

// Admin Model
type DBAdmin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:191;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// SoroushAccount Model (exit node matched signaling credentials)
type DBSoroushAccount struct {
	ID            string    `gorm:"primaryKey;size:191" json:"id"`
	PhoneNumber   string    `gorm:"uniqueIndex;size:191;not null" json:"phoneNumber"`
	Name          string    `gorm:"not null" json:"name"`
	SoroushUserID int64     `json:"soroushUserId"`
	AccessHash    int64     `json:"accessHash"`
	DisplayName   string    `json:"displayName"`
	AuthKey       []byte    `gorm:"type:blob" json:"-"`
	AuthKeyID     []byte    `gorm:"type:blob" json:"-"`
	ServerSalt    []byte    `gorm:"type:blob" json:"-"`
	SessionData   string    `gorm:"type:text" json:"-"`
	DcID          int       `json:"dcId"`
	Role          string    `gorm:"default:''" json:"role"` // "dispatcher", "worker", or ""
	Status        string    `gorm:"default:'idle'" json:"status"`
	LastActive    string    `json:"lastActive"`
	CreatedAt     time.Time `json:"createdAt"`
}

// DBGroupConfig stores the group bus configuration for the server
type DBGroupConfig struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	GroupChatID     int64  `json:"groupChatId"`     // "My lovely family" group chat ID
	GroupAccessHash int64  `json:"groupAccessHash"` // Access hash for channel/supergroup
	PSK             string `gorm:"size:191" json:"psk"` // Pre-shared key for stealth encoding
}

// Initialize MySQL database for server exit node (no fallbacks for security)
func initDB() {
	var err error

	host := os.Getenv("MYSQL_ADDON_HOST")
	port := os.Getenv("MYSQL_ADDON_PORT")
	user := os.Getenv("MYSQL_ADDON_USER")
	password := os.Getenv("MYSQL_ADDON_PASSWORD")
	dbname := os.Getenv("MYSQL_ADDON_DB")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		log.Fatal("[DB] Missing required MySQL environment variables (MYSQL_ADDON_HOST, MYSQL_ADDON_PORT, MYSQL_ADDON_USER, MYSQL_ADDON_PASSWORD, MYSQL_ADDON_DB)")
	}

	// Construct standardized MySQL DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", 
		user, password, host, port, dbname)

	fmt.Printf("[DB] Connecting to MySQL database at %s:%s...\n", host, port)

	// Open connection to MySQL using GORM driver
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("[DB] Failed to connect to MySQL database: %v", err)
	}

	fmt.Println("[DB] MySQL database connection established successfully.")

	// Auto migrate tables (supports GORM migrations inside MySQL)
	err = db.AutoMigrate(&DBAdmin{}, &DBSoroushAccount{}, &DBGroupConfig{}, &DBLogEntry{})
	if err != nil {
		log.Fatalf("[DB] Database migration failed: %v", err)
	}
	fmt.Println("[DB] Tables migrated successfully inside MySQL.")

	// Seed Admin user
	seedAdmin()
}

// Seed default admin user (salman / ADMIN_PASSWORD env or random/fallback)
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
