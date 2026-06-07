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

func TestValidateSQL(t *testing.T) {
	if err := ValidateSQL(`CREATE TABLE t (id INT PRIMARY KEY);`); err != nil {
		t.Fatalf("valid sql: %v", err)
	}
	if err := ValidateSQL(`SELECT 1;`); err == nil {
		t.Fatal("expected error when no CREATE TABLE statements found")
	}
	if err := ValidateSQL(""); err == nil {
		t.Fatal("expected error for empty sql")
	}
}

func TestValidateSQL_missingComma(t *testing.T) {
	sql := "CREATE TABLE `users` (\n" +
		"  `id` INT PRIMARY KEY,\n" +
		"  `username` VARCHAR(100) NOT NULL UNIQUE\n" +
		"  `password` VARCHAR(255) NOT NULL\n" +
		");"
	err := ValidateSQL(sql)
	if err == nil {
		t.Fatal("expected error for missing comma")
	}
	pe, ok := err.(*SQLParseError)
	if !ok {
		t.Fatalf("expected SQLParseError, got %T: %v", err, err)
	}
	if pe.Line != 3 {
		t.Fatalf("expected error on line 3, got line %d: %v", pe.Line, err)
	}
	if !strings.Contains(pe.Message, "missing comma") {
		t.Fatalf("unexpected message: %v", pe.Message)
	}
}

func TestValidateSQL_missingTableName(t *testing.T) {
	sql := "CREATE TABLE (\n" +
		"  `id` INT PRIMARY KEY AUTO_INCREMENT,\n" +
		"  `product_id` INT NOT NULL\n" +
		");"
	err := ValidateSQL(sql)
	if err == nil {
		t.Fatal("expected error for CREATE TABLE without name")
	}
	pe, ok := err.(*SQLParseError)
	if !ok {
		t.Fatalf("expected SQLParseError, got %T: %v", err, err)
	}
	if pe.Line != 1 {
		t.Fatalf("expected error on line 1, got line %d: %v", pe.Line, err)
	}
	if !strings.Contains(pe.Message, "table name") {
		t.Fatalf("unexpected message: %v", pe.Message)
	}
}

func TestFormatSQL(t *testing.T) {
	messy := "create table users(id int primary key,email varchar(255) not null);"
	formatted, err := FormatSQL(messy, DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(formatted, "CREATE TABLE `users` (") {
		t.Fatalf("expected formatted create table, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "  `id` int PRIMARY KEY") {
		t.Fatalf("expected indented columns, got:\n%s", formatted)
	}
}

func TestFormatSQL_preservesComments(t *testing.T) {
	sql := `-- inventory tables
CREATE TABLE roles (
  id int primary key
);

CREATE TABLE users (
  id int primary key, -- surrogate key
  email varchar(255) not null
);

-- relations
ALTER TABLE users ADD CONSTRAINT fk_demo FOREIGN KEY (id) REFERENCES roles(id);`
	formatted, err := FormatSQL(sql, DialectMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(formatted, "-- inventory tables") {
		t.Fatalf("expected section comment before users table, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "-- surrogate key") {
		t.Fatalf("expected trailing column comment, got:\n%s", formatted)
	}
	if !strings.Contains(formatted, "-- relations") {
		t.Fatalf("expected comment before alter table, got:\n%s", formatted)
	}
}

func TestParseSQL_autoIncrementFlag(t *testing.T) {
	sql := `
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  slug VARCHAR(50) NOT NULL UNIQUE
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

func TestExportSQL_postgresMapsDatetime(t *testing.T) {
	schema := &Schema{
		Project: Project{Dialect: DialectPostgres},
		Tables: []TableDTO{
			{
				Table: Table{Name: "events"},
				Columns: []Column{
					{Name: "created_at", DataType: "DATETIME", IsNullable: true, DefaultValue: "CURRENT_TIMESTAMP"},
					{Name: "note", DataType: "LONGTEXT", IsNullable: true},
					{Name: "active", DataType: "TINYINT(1)", IsNullable: true, DefaultValue: "1"},
				},
			},
		},
	}

	sql, err := ExportSQL(schema, DialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(sql)
	if strings.Contains(upper, "DATETIME") {
		t.Fatalf("expected DATETIME mapped away:\n%s", sql)
	}
	if !strings.Contains(sql, `"created_at" TIMESTAMP DEFAULT CURRENT_TIMESTAMP`) {
		t.Fatalf("expected TIMESTAMP column:\n%s", sql)
	}
	if !strings.Contains(sql, `"note" TEXT`) {
		t.Fatalf("expected LONGTEXT -> TEXT:\n%s", sql)
	}
	if !strings.Contains(sql, `"active" BOOLEAN DEFAULT TRUE`) {
		t.Fatalf("expected TINYINT(1) -> BOOLEAN:\n%s", sql)
	}
}

func TestMapPostgresDataType(t *testing.T) {
	cases := map[string]string{
		"DATETIME":      "TIMESTAMP",
		"DATETIME(6)":   "TIMESTAMP(6)",
		"INT UNSIGNED":  "BIGINT",
		"ENUM('a','b')": "ENUM('a','b')",
		"DOUBLE":        "DOUBLE PRECISION",
		"BLOB":          "BYTEA",
	}
	for in, want := range cases {
		got := mapPostgresDataType(in)
		if got != want {
			t.Fatalf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestExportSQL_postgresEnumCreateType(t *testing.T) {
	schema := &Schema{
		Project: Project{Dialect: DialectPostgres},
		Tables: []TableDTO{
			{
				Table: Table{Name: "payments"},
				Columns: []Column{
					{Name: "method", DataType: "ENUM('cash','transfer','ewallet')", IsNullable: true, DefaultValue: "cash"},
				},
			},
		},
	}

	sql, err := ExportSQL(schema, DialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, `CREATE TYPE "payments_method" AS ENUM ('cash', 'transfer', 'ewallet');`) {
		t.Fatalf("missing CREATE TYPE:\n%s", sql)
	}
	if !strings.Contains(sql, `"method" "payments_method" DEFAULT 'cash'`) {
		t.Fatalf("expected enum column type:\n%s", sql)
	}
	if strings.Contains(sql, "VARCHAR(255)") {
		t.Fatalf("unexpected VARCHAR fallback:\n%s", sql)
	}
}
