package envmanager

import (
	"testing"
)

func TestTemplateNameUnique(t *testing.T) {
	path := t.TempDir() + "/test.db"
	db, err := OpenDB(path)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	store := NewStore(db)

	if _, err := store.CreateTemplate("My Template", "x={{.X}}"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateTemplate("my template", "y={{.Y}}"); err == nil {
		t.Fatal("expected duplicate name error")
	}
	if _, err := store.CreateTemplate("other", ""); err == nil {
		t.Fatal("expected empty body error")
	}
}
