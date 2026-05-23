package envmanager

import (
	"os"
	"regexp"
	"testing"
)

func TestGenerateConfirmCode(t *testing.T) {
	code, err := GenerateConfirmCode(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 8 {
		t.Fatalf("got length %d", len(code))
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9]{8}$`).MatchString(code) {
		t.Fatalf("invalid charset: %q", code)
	}
}

func TestDeleteDatabase(t *testing.T) {
	path := t.TempDir() + "/elsaenvmanager.db"
	db, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()

	if err := DeleteDatabase(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected db removed")
	}
}
