package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DBLogEntry struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Timestamp string    `json:"timestamp"`
	Type      string    `json:"type"` // "info", "warn", "error", "success"
	Message   string    `gorm:"type:text" json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

func main() {
	dsn := "ubbjvpmkfqpwo1ku:gJ1RsKBEuzuh0rm5qIl6@tcp(bqgalqe1hnsoyltraetp-mysql.services.clever-cloud.com:3306)/bqgalqe1hnsoyltraetp?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	fmt.Println("=== SERVER LOG ENTRIES (Last 100) ===")
	var logs []DBLogEntry
	db.Order("id desc").Limit(100).Find(&logs)
	for i := len(logs) - 1; i >= 0; i-- {
		l := logs[i]
		fmt.Printf("[%s] [%s] %s\n", l.CreatedAt.Format("15:04:05"), l.Type, l.Message)
	}
}
