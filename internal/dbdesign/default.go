package dbdesign

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reIntDefault      = regexp.MustCompile(`^-?\d+$`)
	reNumberDefault   = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	reDateDefault     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reDateTimeDefault = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	reTimeDefault     = regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)
	reUUIDDefault     = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reQuotedDefault   = regexp.MustCompile(`^('([^']|'')*'|"([^"]|"")*")$`)
)

func ValidateColumnDefault(dataType, defaultValue string, isPrimaryKey bool) error {
	dv := strings.TrimSpace(defaultValue)
	if dv == "" {
		return nil
	}
	if isPrimaryKey {
		return fmt.Errorf("primary key columns cannot have a default value")
	}

	dt := strings.ToUpper(strings.TrimSpace(dataType))
	base := strings.Fields(dt)[0]
	if idx := strings.Index(base, "("); idx > 0 {
		base = base[:idx]
	}

	if strings.Contains(dv, ";") {
		return fmt.Errorf("invalid default value")
	}

	upper := strings.ToUpper(dv)
	switch upper {
	case "NULL", "TRUE", "FALSE", "CURRENT_TIMESTAMP", "CURRENT_DATE", "CURRENT_TIME", "NOW()":
		return validateKeywordDefault(base, upper)
	}
	if strings.HasPrefix(upper, "CURRENT_") {
		return validateKeywordDefault(base, upper)
	}

	if strings.HasPrefix(dt, "ENUM(") {
		return validateEnumDefault(dt, dv, false)
	}
	if strings.HasPrefix(dt, "SET(") {
		return validateEnumDefault(dt, dv, true)
	}

	switch {
	case base == "BLOB" || strings.HasPrefix(base, "TINYBLOB") || strings.HasPrefix(base, "MEDIUMBLOB") || strings.HasPrefix(base, "LONGBLOB"):
		return fmt.Errorf("BLOB columns cannot have a default value")
	case base == "INT" || base == "INTEGER" || base == "BIGINT" || base == "SMALLINT" || base == "TINYINT" || base == "SERIAL":
		if !reIntDefault.MatchString(dv) {
			return fmt.Errorf("default must be an integer for %s", base)
		}
	case base == "DECIMAL" || base == "NUMERIC" || base == "FLOAT" || base == "DOUBLE" || base == "REAL":
		if !reNumberDefault.MatchString(dv) {
			return fmt.Errorf("default must be a number for %s", base)
		}
	case base == "BOOLEAN" || base == "BOOL":
		if !reIntDefault.MatchString(dv) && upper != "TRUE" && upper != "FALSE" {
			return fmt.Errorf("default must be 0, 1, TRUE, or FALSE for BOOLEAN")
		}
	case base == "DATE":
		if !reDateDefault.MatchString(dv) && !reQuotedDefault.MatchString(dv) {
			return fmt.Errorf("default must be YYYY-MM-DD or a quoted date literal")
		}
	case base == "DATETIME" || base == "TIMESTAMP":
		if !reDateTimeDefault.MatchString(dv) && !reQuotedDefault.MatchString(dv) {
			return fmt.Errorf("default must be YYYY-MM-DD HH:MM:SS, a quoted datetime, or CURRENT_TIMESTAMP")
		}
	case base == "TIME":
		if !reTimeDefault.MatchString(dv) && !reQuotedDefault.MatchString(dv) {
			return fmt.Errorf("default must be HH:MM:SS or a quoted time literal")
		}
	case base == "JSON":
		if !reQuotedDefault.MatchString(dv) {
			return fmt.Errorf("JSON default must be a quoted string literal")
		}
	case strings.Contains(base, "CHAR") || base == "TEXT" || strings.HasPrefix(base, "TINYTEXT") || strings.HasPrefix(base, "MEDIUMTEXT") || strings.HasPrefix(base, "LONGTEXT"):
		if !reQuotedDefault.MatchString(dv) {
			return fmt.Errorf("text default must be a quoted string literal")
		}
	case base == "UUID":
		inner := strings.Trim(dv, "'\"")
		if !reUUIDDefault.MatchString(inner) {
			return fmt.Errorf("default must be a valid UUID")
		}
	}

	return nil
}

func validateKeywordDefault(base, keyword string) error {
	switch keyword {
	case "NULL":
		return nil
	case "TRUE", "FALSE":
		if base != "BOOLEAN" && base != "BOOL" {
			return fmt.Errorf("TRUE/FALSE default is only valid for BOOLEAN columns")
		}
	case "CURRENT_TIMESTAMP", "NOW()":
		if base != "DATETIME" && base != "TIMESTAMP" {
			return fmt.Errorf("CURRENT_TIMESTAMP is only valid for DATETIME/TIMESTAMP columns")
		}
	case "CURRENT_DATE":
		if base != "DATE" {
			return fmt.Errorf("CURRENT_DATE is only valid for DATE columns")
		}
	case "CURRENT_TIME":
		if base != "TIME" {
			return fmt.Errorf("CURRENT_TIME is only valid for TIME columns")
		}
	default:
		if strings.HasPrefix(keyword, "CURRENT_") {
			return fmt.Errorf("unsupported default keyword %s", keyword)
		}
	}
	return nil
}

func validateEnumDefault(dataType, defaultValue string, isSet bool) error {
	values := parseEnumValuesFromDataType(dataType)
	if len(values) == 0 {
		return fmt.Errorf("define ENUM/SET values before setting a default")
	}

	lit := defaultValue
	if reQuotedDefault.MatchString(lit) {
		lit = strings.Trim(lit, "'\"")
	}

	if isSet {
		parts := strings.Split(lit, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !enumValueAllowed(values, p) {
				return fmt.Errorf("default SET value %q is not allowed", p)
			}
		}
		return nil
	}

	if !enumValueAllowed(values, lit) {
		return fmt.Errorf("default ENUM value must be one of the defined values")
	}
	return nil
}

func formatEnumDataType(values []string) string {
	if len(values) == 0 {
		return "ENUM()"
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = quoteSQLStringLiteral(v)
	}
	return "ENUM(" + strings.Join(parts, ",") + ")"
}

func parseEnumValuesFromDataType(dataType string) []string {
	upper := strings.ToUpper(strings.TrimSpace(dataType))
	if !strings.HasPrefix(upper, "ENUM(") && !strings.HasPrefix(upper, "SET(") {
		return nil
	}
	start := strings.Index(dataType, "(")
	end := strings.LastIndex(dataType, ")")
	if start < 0 || end <= start {
		return nil
	}
	body := dataType[start+1 : end]
	var values []string
	for _, part := range splitEnumBody(body) {
		v := strings.TrimSpace(part)
		v = strings.Trim(v, "'\"")
		if v != "" {
			values = append(values, v)
		}
	}
	return values
}

func splitEnumBody(body string) []string {
	var parts []string
	var b strings.Builder
	inQuote := rune(0)
	for _, r := range body {
		switch {
		case inQuote != 0:
			b.WriteRune(r)
			if r == inQuote {
				inQuote = 0
			}
		case r == '\'' || r == '"':
			inQuote = r
			b.WriteRune(r)
		case r == ',':
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}

func enumValueAllowed(values []string, candidate string) bool {
	for _, v := range values {
		if strings.EqualFold(v, candidate) {
			return true
		}
	}
	return false
}
