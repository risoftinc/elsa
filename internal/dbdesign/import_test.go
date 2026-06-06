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
