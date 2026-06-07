package dbdesign

import (
	"fmt"
	"sort"
	"strings"
)

type postgresEnumRegistry struct {
	types map[string][]string
	order []string
}

func collectPostgresEnums(schema *Schema) *postgresEnumRegistry {
	r := &postgresEnumRegistry{types: make(map[string][]string)}
	if schema == nil {
		return r
	}
	for _, tbl := range schema.Tables {
		for _, col := range tbl.Columns {
			r.register(tbl.Name, col.Name, col.DataType)
		}
	}
	return r
}

func (r *postgresEnumRegistry) register(table, column, dataType string) (typeName string, ok bool) {
	if r == nil {
		return "", false
	}
	values := parseEnumValuesFromDataType(dataType)
	if len(values) == 0 || !strings.HasPrefix(strings.ToUpper(strings.TrimSpace(dataType)), "ENUM(") {
		return "", false
	}
	typeName = postgresEnumTypeName(table, column)
	if _, exists := r.types[typeName]; !exists {
		r.order = append(r.order, typeName)
	}
	r.types[typeName] = values
	return typeName, true
}

func (r *postgresEnumRegistry) createTypeStatements(dialect string) string {
	if r == nil || len(r.order) == 0 {
		return ""
	}
	names := append([]string(nil), r.order...)
	sort.Strings(names)

	var b strings.Builder
	for _, name := range names {
		values := r.types[name]
		if len(values) == 0 {
			continue
		}
		lits := make([]string, len(values))
		for i, v := range values {
			lits[i] = quoteSQLStringLiteral(v)
		}
		b.WriteString(fmt.Sprintf("CREATE TYPE %s AS ENUM (%s);\n",
			quoteIdent(name, dialect), strings.Join(lits, ", ")))
	}
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	return b.String()
}

func postgresEnumTypeName(table, column string) string {
	name := sanitizePostgresIdent(table) + "_" + sanitizePostgresIdent(column)
	if len(name) > 63 {
		name = name[:63]
		name = strings.TrimRight(name, "_")
	}
	if name == "" {
		return "enum_col"
	}
	return name
}

func sanitizePostgresIdent(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastUnderscore := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func quoteSQLStringLiteral(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func resolveExportColumnType(col Column, tableName, dialect string, pgEnums *postgresEnumRegistry) string {
	if dialect == DialectPostgres && pgEnums != nil {
		if typeName, ok := pgEnums.register(tableName, col.Name, col.DataType); ok {
			return quoteIdent(typeName, dialect)
		}
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(col.DataType)), "ENUM(") {
			return "VARCHAR(255)"
		}
	}
	return exportDataType(col.DataType, dialect)
}
