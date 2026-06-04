package main

import (
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type DBLogEntry struct {
	ID        uint
	Timestamp string
	Type      string
	Message   string
}

func main() {
	db, err := gorm.Open(sqlite.Open("client_config.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	var entries []DBLogEntry
	db.Where("message LIKE ? OR message LIKE ? OR message LIKE ? OR type != ?", "%parseUpdates%", "%Processing update%", "%error%", "info").Order("id desc").Limit(100).Find(&entries)
	fmt.Println("Filtered logs in DB:")
	for _, e := range entries {
		fmt.Printf("[%d] [%s] [%s] %s\n", e.ID, e.Timestamp, e.Type, e.Message)
	}
}
