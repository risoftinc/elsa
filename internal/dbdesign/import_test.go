package dbdesign

import (
	"os"
	"strings"
	"testing"
)

func TestParseSQL(t *testing.T) {
	sql := `
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL,
  role_id BIGINT
);

CREATE TABLE roles (
  id BIGINT PRIMARY KEY,
  name VARCHAR(100) NOT NULL
);

ALTER TABLE users ADD CONSTRAINT fk_users_role FOREIGN KEY (role_id) REFERENCES roles(id);
`
	tables, fks, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("tables: %d", len(tables))
	}
	if len(fks) < 1 {
		t.Fatal("expected alter fk")
	}
}

func TestParseSQL_inlineConstraintFK(t *testing.T) {
	sql := `
CREATE TABLE categories (
    id INT AUTO_INCREMENT PRIMARY KEY,
    category_name VARCHAR(100) NOT NULL
);

CREATE TABLE products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    category_id INT NOT NULL,
    product_name VARCHAR(150) NOT NULL,
    price DECIMAL(15,2) NOT NULL DEFAULT 0,

    CONSTRAINT fk_product_category
        FOREIGN KEY (category_id)
        REFERENCES categories(id)
        ON UPDATE CASCADE
        ON DELETE RESTRICT
);
`
	tables, _, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 2 {
		t.Fatalf("tables: %d", len(tables))
	}
	var products *parsedTable
	for i := range tables {
		if tables[i].Name == "products" {
			products = &tables[i]
			break
		}
	}
	if products == nil {
		t.Fatal("products table not found")
	}
	if len(products.Columns) < 3 {
		t.Fatalf("products columns: %d", len(products.Columns))
	}
	if products.Columns[2].Name != "product_name" || products.Columns[2].DataType != "VARCHAR(150)" {
		t.Fatalf("product_name column: %+v", products.Columns[2])
	}
	if len(products.FKs) != 1 {
		t.Fatalf("expected 1 inline FK, got %d", len(products.FKs))
	}
	fk := products.FKs[0]
	if fk.FromColumn != "category_id" || fk.ToTable != "categories" || fk.ToColumn != "id" {
		t.Fatalf("unexpected fk: %+v", fk)
	}
	if fk.OnDelete != FKActionRestrict {
		t.Fatalf("on delete: %q", fk.OnDelete)
	}
	if fk.OnUpdate != FKActionCascade {
		t.Fatalf("on update: %q", fk.OnUpdate)
	}
	if fk.Name != "fk_product_category" {
		t.Fatalf("fk name: %q", fk.Name)
	}
}

func TestParseSQL_tokoSchema(t *testing.T) {
	b, err := os.ReadFile("../../a.sql")
	if err != nil {
		t.Skip("a.sql not in repo root")
	}
	tables, alterFKs, err := ParseSQL(string(b))
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 10 {
		t.Fatalf("tables: %d", len(tables))
	}
	if len(alterFKs) != 0 {
		t.Fatalf("alter fks: %d", len(alterFKs))
	}

	totalFKs := 0
	for _, pt := range tables {
		totalFKs += len(pt.FKs)
	}
	if totalFKs != 9 {
		t.Fatalf("expected 9 inline FKs, got %d", totalFKs)
	}

	names := make([]string, len(tables))
	for i, pt := range tables {
		names[i] = pt.Name
	}
	if !strings.Contains(strings.Join(names, ","), "order_details") {
		t.Fatalf("missing tables: %v", names)
	}
}

func TestParseSQL_postgresEnumType(t *testing.T) {
	sql := `
CREATE TYPE "payments_method" AS ENUM ('cash', 'transfer', 'ewallet');

CREATE TABLE payments (
  id SERIAL PRIMARY KEY,
  method "payments_method" NOT NULL DEFAULT 'cash'
);
`
	tables, _, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || len(tables[0].Columns) != 2 {
		t.Fatalf("unexpected tables: %+v", tables)
	}
	method := tables[0].Columns[1]
	if method.Name != "method" {
		t.Fatalf("column name: %q", method.Name)
	}
	want := "ENUM('cash','transfer','ewallet')"
	if method.DataType != want {
		t.Fatalf("datatype: %q want %q", method.DataType, want)
	}
}

func TestFormatSQL_preservesPostgresEnum(t *testing.T) {
	sql := `
CREATE TYPE "payments_method" AS ENUM ('cash', 'transfer');

CREATE TABLE payments (
  id SERIAL PRIMARY KEY,
  method "payments_method" DEFAULT 'cash'
);
`
	formatted, err := FormatSQL(sql, DialectPostgres)
	if err != nil {
		t.Fatal(err)
	}
	tables, _, err := ParseSQL(formatted)
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || len(tables[0].Columns) != 2 {
		t.Fatalf("unexpected tables: %+v", tables)
	}
	if tables[0].Columns[1].DataType != "ENUM('cash','transfer')" {
		t.Fatalf("enum lost after format: %q", tables[0].Columns[1].DataType)
	}
	if !strings.Contains(formatted, `CREATE TYPE "payments_method" AS ENUM`) {
		t.Fatalf("missing CREATE TYPE in formatted output:\n%s", formatted)
	}
}

func TestParseSQL_duplicateColumnName(t *testing.T) {
	sql := `CREATE TABLE users (
  id INT PRIMARY KEY,
  email VARCHAR(255),
  email VARCHAR(100)
);`
	_, _, err := ParseSQL(sql)
	if err == nil {
		t.Fatal("expected error for duplicate column name")
	}
	pe, ok := err.(*SQLParseError)
	if !ok {
		t.Fatalf("expected SQLParseError, got %T: %v", err, err)
	}
	if !strings.Contains(pe.Message, "duplicate column name") {
		t.Fatalf("unexpected message: %v", pe.Message)
	}
}

func TestParseSQL_normalizesPostgresInteger(t *testing.T) {
	sql := `CREATE TABLE items (
  id INTEGER PRIMARY KEY,
  qty INTEGER NOT NULL
);`
	tables, _, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if tables[0].Columns[0].DataType != "INT" {
		t.Fatalf("id type: %q want INT", tables[0].Columns[0].DataType)
	}
	if tables[0].Columns[1].DataType != "INT" {
		t.Fatalf("qty type: %q want INT", tables[0].Columns[1].DataType)
	}
}

func TestValidateSQL_duplicateColumnName(t *testing.T) {
	sql := `CREATE TABLE t (
  a INT,
  a TEXT
);`
	err := ValidateSQL(sql)
	if err == nil {
		t.Fatal("expected validation error for duplicate column")
	}
	if !strings.Contains(err.Error(), "duplicate column name") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSQL_invalidVarcharLength(t *testing.T) {
	sql := "CREATE TABLE certificates (\n  `certificate_no` VARCHAR(10ddd0) NOT NULL\n);"
	err := ValidateSQL(sql)
	if err == nil {
		t.Fatal("expected error for invalid VARCHAR length")
	}
	if !strings.Contains(err.Error(), "invalid column type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateSQL_validVarcharLength(t *testing.T) {
	sql := "CREATE TABLE certificates (\n  `certificate_no` VARCHAR(100) NOT NULL\n);"
	if err := ValidateSQL(sql); err != nil {
		t.Fatalf("expected valid sql: %v", err)
	}
}

func TestValidateSQL_invalidDecimalPrecision(t *testing.T) {
	sql := "CREATE TABLE items (price DECIMAL(10,xx) NOT NULL);"
	err := ValidateSQL(sql)
	if err == nil {
		t.Fatal("expected error for invalid DECIMAL precision")
	}
}

func TestParseSQL_sqliteDesignerFKComment(t *testing.T) {
	sql := `CREATE TABLE roles (
  id INTEGER PRIMARY KEY
);

CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  role_id INTEGER
);

-- FK: fk_users_role_id ( ON DELETE RESTRICT ON UPDATE RESTRICT) users.role_id -> roles.id
-- ALTER TABLE "users" ADD CONSTRAINT "fk_users_role_id" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON DELETE RESTRICT ON UPDATE RESTRICT;`

	_, fks, err := ParseSQL(sql)
	if err != nil {
		t.Fatal(err)
	}
	if len(fks) != 1 {
		t.Fatalf("expected 1 FK from designer comment, got %d", len(fks))
	}
	if fks[0].FromTable != "users" || fks[0].FromColumn != "role_id" {
		t.Fatalf("unexpected fk: %+v", fks[0])
	}
	if fks[0].ToTable != "roles" || fks[0].ToColumn != "id" {
		t.Fatalf("unexpected fk target: %+v", fks[0])
	}
}

func TestFormatSQL_sqlitePreservesDesignerFKComments(t *testing.T) {
	sql := `CREATE TABLE roles (id INTEGER PRIMARY KEY);
CREATE TABLE users (id INTEGER PRIMARY KEY, role_id INTEGER);

-- FK: fk_users_role_id ( ON DELETE RESTRICT ON UPDATE RESTRICT) users.role_id -> roles.id
-- ALTER TABLE "users" ADD CONSTRAINT "fk_users_role_id" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON DELETE RESTRICT ON UPDATE RESTRICT;`

	formatted, err := FormatSQL(sql, DialectSQLite)
	if err != nil {
		t.Fatal(err)
	}
	_, fks, err := ParseSQL(formatted)
	if err != nil {
		t.Fatal(err)
	}
	if len(fks) != 1 {
		t.Fatalf("expected FK preserved after sqlite format, got %d\n%s", len(fks), formatted)
	}
	if !strings.Contains(formatted, "-- FK: fk_users_role_id") {
		t.Fatalf("expected designer FK comment in output:\n%s", formatted)
	}
}
