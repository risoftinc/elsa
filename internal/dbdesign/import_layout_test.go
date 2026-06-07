package dbdesign

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Project{}, &Table{}, &Column{}, &Relation{}, &Index{}, &IndexColumn{}); err != nil {
		t.Fatal(err)
	}
	return NewStore(db)
}

func TestImportSQL_preservesTableLayoutOnReplace(t *testing.T) {
	store := openTestStore(t)
	proj, err := store.CreateProject("demo", "", DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}

	tbl, err := store.CreateTable(proj.ID, "users", 180, 420)
	if err != nil {
		t.Fatal(err)
	}
	width := 300.0
	updated, err := store.UpdateTable(tbl.ID, "", ptrFloat(180), ptrFloat(420), &width)
	if err != nil {
		t.Fatal(err)
	}

	sql := `
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL
);
CREATE TABLE roles (
  id BIGINT PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);
`
	if _, err := store.ImportSQL(proj.ID, sql, true); err != nil {
		t.Fatal(err)
	}

	schema, err := store.GetSchema(proj.ID)
	if err != nil {
		t.Fatal(err)
	}

	var users, roles *TableDTO
	for i := range schema.Tables {
		switch schema.Tables[i].Name {
		case "users":
			users = &schema.Tables[i]
		case "roles":
			roles = &schema.Tables[i]
		}
	}
	if users == nil || roles == nil {
		t.Fatalf("expected users and roles tables, got %+v", schema.Tables)
	}
	if users.PosX != updated.PosX || users.PosY != updated.PosY || users.Width != updated.Width {
		t.Fatalf("users layout not preserved: got (%v,%v,%v) want (%v,%v,%v)",
			users.PosX, users.PosY, users.Width, updated.PosX, updated.PosY, updated.Width)
	}
	if roles.PosX == users.PosX && roles.PosY == users.PosY {
		t.Fatalf("new roles table should not reuse users coordinates: %+v", roles.Table)
	}
}

func ptrFloat(v float64) *float64 { return &v }
