package dbdesign

import "testing"

func TestValidateColumnDefault(t *testing.T) {
	tests := []struct {
		dataType string
		def      string
		pk       bool
		wantErr  bool
	}{
		{"INT", "0", false, false},
		{"INT", "'1'", false, true},
		{"VARCHAR(255)", "'hello'", false, false},
		{"VARCHAR(255)", "hello", false, true},
		{"ENUM('a','b')", "'a'", false, false},
		{"ENUM('a','b')", "'c'", false, true},
		{"DATETIME", "CURRENT_TIMESTAMP", false, false},
		{"INT", "CURRENT_TIMESTAMP", false, true},
		{"BLOB", "'x'", false, true},
		{"INT", "1", true, true},
	}
	for _, tc := range tests {
		err := ValidateColumnDefault(tc.dataType, tc.def, tc.pk)
		if (err != nil) != tc.wantErr {
			t.Fatalf("ValidateColumnDefault(%q, %q, pk=%v) err=%v wantErr=%v", tc.dataType, tc.def, tc.pk, err, tc.wantErr)
		}
	}
}
