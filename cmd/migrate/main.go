package main

import (
	"context"
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"lib.tekkno.co.za/mmmigrate"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: migrate <create|up|down> [args]")
	}

	switch os.Args[1] {
	case "create":
		runCreate()
	case "up":
		runUp(context.Background())
	case "down":
		runDown(context.Background())
	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}

func runCreate() {
	if len(os.Args) < 3 {
		log.Fatal("usage: migrate create <name>")
	}
	if err := mmmigrate.CreateMigration(os.Args[2]); err != nil {
		log.Fatalf("create migration: %v", err)
	}
}

func runUp(ctx context.Context) {
	db := connectToDb()
	m, err := mmmigrate.New(db)
	if err != nil {
		log.Fatalf("creating migrator: %w", err)
	}
	if err := m.Up(ctx, mmmigrate.All()); err != nil {
		log.Fatalf("up: %v", err)
	}
}

func runDown(ctx context.Context) {
	db := connectToDb()
	m, err := mmmigrate.New(db)
	if err != nil {
		log.Fatalf("creating migrator: %w", err)
	}
	if err := m.Down(ctx, mmmigrate.All()); err != nil {
		log.Fatalf("down: %v", err)
	}
}

func connectToDb() *sql.DB {
	db, err := sql.Open("pgx", os.Getenv("DB_DSN"))
	if err != nil {
		panic("Failed to connect to database")
	}
	return db
}
