package dbdesign

import (
	"strings"
)

// exportDataType maps a canvas/SQL type to the target dialect for DDL export.
func exportDataType(dataType, dialect string) string {
	dialect = normalizeDialect(dialect)
	dt := strings.TrimSpace(dataType)
	if dt == "" || dialect == DialectMySQL {
		return dt
	}
	if dialect == DialectPostgres {
		return mapPostgresDataType(dt)
	}
	if dialect == DialectSQLite {
		return mapSQLiteDataType(dt)
	}
	return dt
}

func mapPostgresDataType(dt string) string {
	s := strings.TrimSpace(dt)
	unsigned := typeHasUnsigned(s)
	s = stripUnsignedModifier(s)

	upper := strings.ToUpper(s)
	if strings.HasPrefix(upper, "SET(") {
		return "TEXT"
	}

	base, suffix := splitTypeBaseSuffix(s)
	switch strings.ToUpper(base) {
	case "DATETIME":
		return "TIMESTAMP" + suffix
	case "TINYINT":
		if suffix == "(1)" {
			return "BOOLEAN"
		}
		return "SMALLINT" + suffix
	case "MEDIUMINT":
		return "INTEGER" + suffix
	case "INT", "INTEGER":
		if unsigned {
			return "BIGINT" + suffix
		}
		return "INTEGER" + suffix
	case "BIGINT":
		if unsigned {
			return "NUMERIC(20)" + suffix
		}
		return "BIGINT" + suffix
	case "SMALLINT", "SERIAL", "BIGSERIAL", "SMALLSERIAL":
		return strings.ToUpper(base) + suffix
	case "LONGTEXT", "MEDIUMTEXT", "TINYTEXT":
		return "TEXT"
	case "LONGBLOB", "MEDIUMBLOB", "TINYBLOB", "BLOB":
		return "BYTEA"
	case "DOUBLE":
		return "DOUBLE PRECISION"
	case "FLOAT":
		return "REAL"
	case "BOOL":
		return "BOOLEAN"
	case "JSON":
		return "JSONB"
	default:
		return s
	}
}

func mapSQLiteDataType(dt string) string {
	s := strings.TrimSpace(dt)
	unsigned := typeHasUnsigned(s)
	s = stripUnsignedModifier(s)

	base, suffix := splitTypeBaseSuffix(s)
	switch strings.ToUpper(base) {
	case "DATETIME":
		return "TEXT"
	case "TINYINT":
		if suffix == "(1)" {
			return "INTEGER"
		}
		return "INTEGER" + suffix
	case "MEDIUMINT", "INT", "INTEGER":
		if unsigned {
			return "INTEGER" + suffix
		}
		return "INTEGER" + suffix
	case "BIGINT":
		return "INTEGER" + suffix
	case "LONGTEXT", "MEDIUMTEXT", "TINYTEXT":
		return "TEXT"
	case "LONGBLOB", "MEDIUMBLOB", "TINYBLOB", "BLOB":
		return "BLOB"
	case "BOOL", "BOOLEAN":
		return "INTEGER"
	case "JSON":
		return "TEXT"
	case "UUID":
		return "TEXT"
	default:
		if strings.HasPrefix(strings.ToUpper(s), "ENUM(") || strings.HasPrefix(strings.ToUpper(s), "SET(") {
			return "TEXT"
		}
		return s
	}
}

func exportDefaultValue(defaultValue, dataType, dialect string) string {
	dv := strings.TrimSpace(defaultValue)
	if dv == "" {
		return ""
	}
	dialect = normalizeDialect(dialect)
	if dialect != DialectPostgres {
		return dv
	}
	if len(parseEnumValuesFromDataType(dataType)) > 0 &&
		strings.HasPrefix(strings.ToUpper(strings.TrimSpace(dataType)), "ENUM(") {
		if strings.HasPrefix(dv, "'") || strings.HasPrefix(dv, `"`) {
			return dv
		}
		return quoteSQLStringLiteral(dv)
	}
	mapped := exportDataType(dataType, dialect)
	base, _ := splitTypeBaseSuffix(mapped)
	if strings.EqualFold(base, "BOOLEAN") {
		switch strings.ToUpper(dv) {
		case "0", "FALSE":
			return "FALSE"
		case "1", "TRUE":
			return "TRUE"
		}
	}
	return dv
}

func typeHasUnsigned(dt string) bool {
	return strings.Contains(strings.ToUpper(dt), " UNSIGNED")
}

func stripUnsignedModifier(dt string) string {
	s := dt
	for {
		upper := strings.ToUpper(s)
		idx := strings.Index(upper, " UNSIGNED")
		if idx < 0 {
			return strings.TrimSpace(s)
		}
		s = strings.TrimSpace(s[:idx] + s[idx+len(" UNSIGNED"):])
	}
}

func splitTypeBaseSuffix(dt string) (base, suffix string) {
	s := strings.TrimSpace(dt)
	i := strings.Index(s, "(")
	if i < 0 {
		return s, ""
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i:])
}
