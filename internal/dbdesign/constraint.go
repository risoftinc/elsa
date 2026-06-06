package dbdesign

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	FKActionRestrict   = "RESTRICT"
	FKActionCascade    = "CASCADE"
	FKActionSetNull    = "SET NULL"
	FKActionNoAction   = "NO ACTION"
	FKActionSetDefault = "SET DEFAULT"
)

var (
	reOnDelete = regexp.MustCompile(`(?i)ON\s+DELETE\s+(RESTRICT|CASCADE|SET\s+NULL|NO\s+ACTION|SET\s+DEFAULT)`)
	reOnUpdate = regexp.MustCompile(`(?i)ON\s+UPDATE\s+(RESTRICT|CASCADE|SET\s+NULL|NO\s+ACTION|SET\s+DEFAULT)`)
	reFKName   = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func DefaultFKAction() string {
	return FKActionRestrict
}

func NormalizeFKAction(raw string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(raw))
	s = strings.Join(strings.Fields(s), " ")
	switch s {
	case "", FKActionRestrict, FKActionCascade, FKActionSetNull, FKActionNoAction, FKActionSetDefault:
		if s == "" {
			return DefaultFKAction(), nil
		}
		return s, nil
	default:
		return "", fmt.Errorf("invalid FK action %q (use RESTRICT, CASCADE, SET NULL, NO ACTION, or SET DEFAULT)", raw)
	}
}

func NormalizeFKActionDefault(raw string) string {
	v, err := NormalizeFKAction(raw)
	if err != nil {
		return DefaultFKAction()
	}
	return v
}

func ParseFKActions(sql string) (onDelete, onUpdate string) {
	onDelete = DefaultFKAction()
	onUpdate = DefaultFKAction()
	if m := reOnDelete.FindStringSubmatch(sql); len(m) > 1 {
		onDelete = NormalizeFKActionDefault(strings.ReplaceAll(m[1], "  ", " "))
	}
	if m := reOnUpdate.FindStringSubmatch(sql); len(m) > 1 {
		onUpdate = NormalizeFKActionDefault(strings.ReplaceAll(m[1], "  ", " "))
	}
	return onDelete, onUpdate
}

func formatFKActionClause(onDelete, onUpdate string) string {
	d := NormalizeFKActionDefault(onDelete)
	u := NormalizeFKActionDefault(onUpdate)
	return fmt.Sprintf(" ON DELETE %s ON UPDATE %s", d, u)
}

func sanitizeIdentPart(raw, fallback string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else if r == '-' || r == ' ' {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return fallback
	}
	return out
}

func DefaultFKName(fromTable, fromColumn string) string {
	return fmt.Sprintf("fk_%s_%s", sanitizeIdentPart(fromTable, "table"), sanitizeIdentPart(fromColumn, "col"))
}

func NormalizeFKName(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", fmt.Errorf("FK constraint name is required")
	}
	if !reFKName.MatchString(s) {
		return "", fmt.Errorf("FK name must contain only letters, numbers, and underscores")
	}
	return s, nil
}

func NormalizeFKNameDefault(raw string) string {
	v, err := NormalizeFKName(raw)
	if err != nil {
		return ""
	}
	return v
}

func relationConstraintName(name, fromTable, fromColumn string) string {
	if n := NormalizeFKNameDefault(name); n != "" {
		return n
	}
	return DefaultFKName(fromTable, fromColumn)
}
