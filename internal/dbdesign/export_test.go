package dbdesign

import (
	"strings"
	"testing"
)

func TestExportSQL_autoIncrement(t *testing.T) {
	schema := &Schema{
		Project: Project{Dialect: DialectMySQL},
		Tables: []TableDTO{
			{
				Table: Table{Name: "users"},
				Columns: []Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true, IsAutoIncrement: true},
					{Name: "code", DataType: "VARCHAR(50)", IsPrimaryKey: true},
				},
			},
		},
	}

	sql, err := ExportSQL(schema, DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, "`id` BIGINT PRIMARY KEY AUTO_INCREMENT") {
		t.Fatalf("missing AI on id:\n%s", sql)
	}
	if !strings.Contains(sql, "`code` VARCHAR(50) PRIMARY KEY") {
		t.Fatalf("missing non-AI PK:\n%s", sql)
	}
	if strings.Contains(sql, "`code` VARCHAR(50) PRIMARY KEY AUTO_INCREMENT") {
		t.Fatalf("unexpected AI on code PK:\n%s", sql)
	}
}

func TestExportSQL_noAutoIncrementOnPK(t *testing.T) {
	schema := &Schema{
		Project: Project{Dialect: DialectMySQL},
		Tables: []TableDTO{
			{
				Table: Table{Name: "roles"},
				Columns: []Column{
					{Name: "id", DataType: "BIGINT", IsPrimaryKey: true},
				},
			},
		},
	}

	sql, err := ExportSQL(schema, DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sql, "AUTO_INCREMENT") {
		t.Fatalf("expected no AUTO_INCREMENT when flag unset:\n%s", sql)
	}
}

func TestParseSQL_autoIncrementFlag(t *testing.T) {
	sql := `
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  slug VARCHAR(50) PRIMARY KEY
);
`
	tables, _, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || len(tables[0].Columns) != 2 {
		t.Fatalf("unexpected parse: %+v", tables)
	}
	if !tables[0].Columns[0].IsAutoIncrement {
		t.Fatal("expected id to be auto increment")
	}
	if tables[0].Columns[1].IsAutoIncrement {
		t.Fatal("slug PK should not be auto increment")
	}
}
