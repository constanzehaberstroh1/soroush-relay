package main

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type DBAdmin struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string `gorm:"not null"`
}

type DBSoroushAccount struct {
	ID            string `gorm:"primaryKey"`
	PhoneNumber   string `gorm:"uniqueIndex"`
	Name          string `gorm:"not null"`
	SoroushUserID int64  `json:"soroushUserId"`
	AccessHash    int64  `json:"accessHash"`
	Status        string `gorm:"default:'idle'"`
}

type DBTunnelConfig struct {
	ID                   uint  `gorm:"primaryKey"`
	GroupChatID          int64 `json:"groupChatId"`
	GroupAccessHash      int64 `json:"groupAccessHash"`
	PSK                  string
	DispatcherUserID     int64
	DispatcherAccessHash int64
}

func main() {
	db, err := gorm.Open(sqlite.Open("client_config.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open: %v", err)
	}

	fmt.Println("=== CLIENT ACCOUNTS ===")
	var accs []DBSoroushAccount
	db.Find(&accs)
	for _, a := range accs {
		fmt.Printf("ID: %s, Phone: %s, UID: %d, AccessHash: %d, Status: %s\n", a.ID, a.PhoneNumber, a.SoroushUserID, a.AccessHash, a.Status)
	}

	fmt.Println("\n=== CLIENT TUNNEL CONFIG ===")
	var cfgs []DBTunnelConfig
	db.Find(&cfgs)
	for _, c := range cfgs {
		fmt.Printf("ID: %d, GroupChatID: %d, GroupAccessHash: %d, PSK: %s\n", c.ID, c.GroupChatID, c.GroupAccessHash, c.PSK)
	}
}
