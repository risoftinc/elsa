package envmanager

import "testing"

func TestRenderTemplate(t *testing.T) {
	body := "DB_HOST={{.DB_HOST}}\nDB_USER={{.DB_USER}}"
	data := map[string]string{
		"DB_HOST": "localhost",
		"DB_USER": "admin",
	}
	out, err := RenderTemplate(body, data)
	if err != nil {
		t.Fatal(err)
	}
	want := "DB_HOST=localhost\nDB_USER=admin"
	if out != want {
		t.Fatalf("got %q, want %q", out, want)
	}
}

func TestDefaultFilename(t *testing.T) {
	if got := DefaultFilename("Staging"); got != ".staging.env" {
		t.Fatalf("got %q", got)
	}
}
