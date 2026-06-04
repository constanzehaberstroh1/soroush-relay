package main

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"soroush-relay/soroushlib"
)

type DBSoroushAccount struct {
	ID            string `gorm:"primaryKey"`
	PhoneNumber   string
	SoroushUserID int64
	AccessHash    int64
	AuthKey       []byte
	AuthKeyID     []byte
	ServerSalt    []byte
	Status        string
}

type DBTunnelConfig struct {
	ID              uint  `gorm:"primaryKey"`
	GroupChatID     int64
	GroupAccessHash int64
}

func main() {
	db, err := gorm.Open(sqlite.Open("client_config.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	var account DBSoroushAccount
	if err := db.Where("length(auth_key) > 0").First(&account).Error; err != nil {
		log.Fatalf("no account: %v", err)
	}
	fmt.Printf("Using account: %s (UID: %d, AccessHash: %d)\n", account.PhoneNumber, account.SoroushUserID, account.AccessHash)

	var config DBTunnelConfig
	db.First(&config)
	fmt.Printf("Group: ChatID=%d, AccessHash=%d\n", config.GroupChatID, config.GroupAccessHash)

	// Worker UID to search for
	workerUID := int64(64698297)
	workerBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(workerBytes, uint64(workerUID))
	fmt.Printf("Searching for worker UID %d (bytes: %s)\n\n", workerUID, hex.EncodeToString(workerBytes))

	session, transport := soroushlib.RestoreSession(account.AuthKey, account.AuthKeyID, account.ServerSalt)
	session.Logger = func(msg string, level string) {
		if level == "error" || level == "warn" {
			fmt.Printf("[%s] %s\n", level, msg)
		}
	}

	ctx := context.Background()
	connCtx, connCancel := context.WithTimeout(ctx, 15*time.Second)
	if err := transport.Connect(connCtx); err != nil {
		connCancel()
		log.Fatalf("connect failed: %v", err)
	}
	connCancel()
	defer transport.Disconnect()

	session.StartReader(ctx)
	fmt.Println("Connected to Soroush ✅")

	// 1. Call getDialogs wrapped in initConnection
	fmt.Println("\n=== Step 1: getDialogs ===")
	initBody := soroushlib.BuildGetDialogsRequest()
	wrappedInit := soroushlib.WrapInitConnection(soroushlib.SoroushAppID, initBody)
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	_, dialogsResp, err := session.SendAndWait(initCtx, wrappedInit, true)
	initCancel()
	if err != nil {
		fmt.Printf("getDialogs failed: %v\n", err)
	} else {
		raw := dialogsResp.GetData()
		fmt.Printf("getDialogs response: %d bytes\n", len(raw))
		testScanner(raw)
	}

	fmt.Println("\nDone!")
}

func testScanner(raw []byte) {
	cids := []uint32{0x274DB727, 0x6A2179DD}
	for _, targetCID := range cids {
		for i := 0; i+32 <= len(raw); i++ {
			cid := binary.LittleEndian.Uint32(raw[i : i+4])
			if cid == targetCID {
				flags := binary.LittleEndian.Uint32(raw[i+4 : i+8])
				flags2 := binary.LittleEndian.Uint32(raw[i+8 : i+12])
				id := int64(binary.LittleEndian.Uint64(raw[i+12 : i+20]))
				var accessHash int64
				hasAH := flags&(1<<0) != 0
				if hasAH {
					accessHash = int64(binary.LittleEndian.Uint64(raw[i+20 : i+28]))
				}
				fmt.Printf("Parsed user CID=0x%08X at offset 0x%04X: flags=0x%08X, flags2=0x%08X, id=%d, hasAccessHash=%v, accessHash=%d\n",
					cid, i, flags, flags2, id, hasAH, accessHash)
			}
		}
	}
}
