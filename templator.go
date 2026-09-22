package mmmigrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const migrationTemplate = `package migrations

import (
	"database/sql"

	"lib.tekkno.co.za/mmmigrate"
)

func init() {
	mmmigrate.Register(
		//Up
		func(tx *sql.Tx) error {
			//TODO: up	
			_, err := tx.Exec(` + "`" + `` + "`" + `)
			return err
		},
		//Down
		func(tx *sql.Tx) error {
			// TODO: down
			_, err := tx.Exec(` + "`" + `` + "`" + `)
			return err
		},
	)
}
`

var invalidChars = regexp.MustCompile(`[^a-z0-9_]+`)

func CreateMigration(rawName string) error {
	if err := os.MkdirAll(DefaultMigrationsDir, 0o755); err != nil {
		return fmt.Errorf("scaffold: creating migrations dir %q: %w", DefaultMigrationsDir, err)
	}

	name := strings.ToLower(strings.TrimSpace(rawName))
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")
	//Strip out invalid characters
	name = invalidChars.ReplaceAllString(name, "")

	if name == "" {
		return fmt.Errorf("invalid name - use alphanumeric/underscores for name")
	}

	timestamp := time.Now().UTC().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.go", timestamp, name)
	path := filepath.Join(DefaultMigrationsDir, filename)

	//Check if already exists
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("file already exists: %s", path)
	}

	//Load template
	writer, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return err
	}

	//Create file
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	//Ensure file is closed once written
	defer f.Close()

	if err := writer.Execute(f, nil); err != nil {
		return err
	}

	fmt.Printf("created %s\n", path)
	return nil
}
