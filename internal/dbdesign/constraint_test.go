package dbdesign

import (
	"strings"
	"testing"
)

func TestNormalizeFKAction(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", FKActionRestrict, true},
		{"cascade", FKActionCascade, true},
		{"SET NULL", FKActionSetNull, true},
		{"no action", FKActionNoAction, true},
		{"INVALID", "", false},
	}
	for _, tc := range tests {
		got, err := NormalizeFKAction(tc.in)
		if tc.ok && err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q: expected error", tc.in)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseFKActions(t *testing.T) {
	sql := `FOREIGN KEY (x) REFERENCES y(z) ON UPDATE CASCADE ON DELETE RESTRICT`
	del, upd := ParseFKActions(sql)
	if del != FKActionRestrict {
		t.Fatalf("delete: %q", del)
	}
	if upd != FKActionCascade {
		t.Fatalf("update: %q", upd)
	}
}

func TestDefaultFKName(t *testing.T) {
	got := DefaultFKName("Order Details", "user-id")
	if got != "fk_order_details_user_id" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeFKName(t *testing.T) {
	got, err := NormalizeFKName("fk_orders_user_id")
	if err != nil || got != "fk_orders_user_id" {
		t.Fatalf("got %q err %v", got, err)
	}
	if _, err := NormalizeFKName("bad name"); err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestExportForeignKeyConstraints(t *testing.T) {
	schema := &Schema{
		Project: Project{Dialect: DialectMySQL},
		Tables: []TableDTO{
			{Table: Table{ID: 1, Name: "orders"}, Columns: []Column{{ID: 10, Name: "user_id"}}},
			{Table: Table{ID: 2, Name: "users"}, Columns: []Column{{ID: 20, Name: "id"}}},
		},
		Relations: []Relation{{
			Name:        "fk_orders_user",
			FromTableID: 1, FromColumnID: 10, ToTableID: 2, ToColumnID: 20,
			OnDelete: FKActionRestrict, OnUpdate: FKActionCascade,
		}},
	}
	line := buildForeignKey(schema, schema.Relations[0], DialectMySQL)
	if !strings.Contains(line, "CONSTRAINT `fk_orders_user`") {
		t.Fatalf("missing constraint name: %q", line)
	}
	if !strings.Contains(line, "ON DELETE RESTRICT") {
		t.Fatalf("missing ON DELETE: %q", line)
	}
	if !strings.Contains(line, "ON UPDATE CASCADE") {
		t.Fatalf("missing ON UPDATE: %q", line)
	}
}
