package main

import (
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DBAdmin struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;size:191;not null" json:"username"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type DBGroupConfig struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	GroupChatID     int64  `json:"groupChatId"`
	GroupAccessHash int64  `json:"groupAccessHash"`
	PSK             string `gorm:"size:191" json:"psk"`
}

type DBSoroushAccount struct {
	ID            string    `gorm:"primaryKey;size:191" json:"id"`
	PhoneNumber   string    `gorm:"uniqueIndex;size:191;not null" json:"phoneNumber"`
	Name          string    `gorm:"not null" json:"name"`
	SoroushUserID int64     `json:"soroushUserId"`
	AccessHash    int64     `json:"accessHash"`
	DisplayName   string    `json:"displayName"`
	Role          string    `gorm:"default:''" json:"role"`
	Status        string    `gorm:"default:'idle'" json:"status"`
	CreatedAt     time.Time `json:"createdAt"`
}

func main() {
	dsn := "ubbjvpmkfqpwo1ku:gJ1RsKBEuzuh0rm5qIl6@tcp(bqgalqe1hnsoyltraetp-mysql.services.clever-cloud.com:3306)/bqgalqe1hnsoyltraetp?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("=== SERVER ADMINS ===")
	var admins []DBAdmin
	db.Find(&admins)
	for _, admin := range admins {
		fmt.Printf("ID: %d, Username: %s, Hash: %s\n", admin.ID, admin.Username, admin.PasswordHash)
	}

	fmt.Println("\n=== SERVER GROUP CONFIGS ===")
	var configs []DBGroupConfig
	db.Find(&configs)
	for _, c := range configs {
		fmt.Printf("ID: %d, GroupChatID: %d, GroupAccessHash: %d, PSK: %s\n", c.ID, c.GroupChatID, c.GroupAccessHash, c.PSK)
	}

	fmt.Println("\n=== SERVER ACCOUNTS ===")
	var accounts []DBSoroushAccount
	db.Find(&accounts)
	for _, acc := range accounts {
		fmt.Printf("ID: %s, Phone: %s, UID: %d, AccessHash: %d, Status: %s\n", acc.ID, acc.PhoneNumber, acc.SoroushUserID, acc.AccessHash, acc.Status)
	}

	// Reset server password for 'salman' to 'admin123'
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("bcrypt failed: %v", err)
	}
	err = db.Model(&DBAdmin{}).Where("username = ?", "salman").Update("password_hash", string(hashedPassword)).Error
	if err != nil {
		log.Fatalf("Update failed: %v", err)
	}
	fmt.Println("\nServer password for 'salman' successfully reset to 'admin123'!")
}
