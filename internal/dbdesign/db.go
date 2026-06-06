package dbdesign

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DefaultDBPath returns the default SQLite path for db designer
func DefaultDBPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	dir := filepath.Join(configDir, "elsa")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return filepath.Join(dir, "elsadbdesign.db"), nil
}

// OpenDB opens SQLite and migrates schema
func OpenDB(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.AutoMigrate(&Project{}, &Table{}, &Column{}, &Relation{}, &Index{}, &IndexColumn{}); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}
	return db, nil
}
