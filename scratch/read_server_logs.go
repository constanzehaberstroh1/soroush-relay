package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type DBLogEntry struct {
	ID        uint   `gorm:"primaryKey"`
	Timestamp string `gorm:"column:timestamp"`
	Type      string `gorm:"size:20"`
	Message   string `gorm:"type:text"`
}

func main() {
	dsn := "ubbjvpmkfqpwo1ku:gJ1RsKBEuzuh0rm5qIl6@tcp(bqgalqe1hnsoyltraetp-mysql.services.clever-cloud.com:3306)/bqgalqe1hnsoyltraetp?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}

	var logs []DBLogEntry
	if err := db.Order("id desc").Limit(200).Find(&logs).Error; err != nil {
		log.Fatalf("failed to query: %v", err)
	}

	fmt.Println("=== SERVER LOGS FROM MYSQL ===")
	for _, l := range logs {
		fmt.Printf("[%d] [%s] [%s] %s\n", l.ID, l.Timestamp, l.Type, l.Message)
	}
}
