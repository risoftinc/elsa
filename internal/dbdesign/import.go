package dbdesign

import (
	"fmt"
	"regexp"
	"strings"
)

// ImportResult summarizes an SQL import operation
type ImportResult struct {
	TablesCreated    int `json:"tables_created"`
	ColumnsCreated   int `json:"columns_created"`
	RelationsCreated int `json:"relations_created"`
	IndexesCreated   int `json:"indexes_created"`
}

type parsedIndex struct {
	Name     string
	IsUnique bool
	Columns  []string
}

type parsedColumn struct {
	Name            string
	DataType        string
	IsPrimaryKey    bool
	IsAutoIncrement bool
	IsNullable      bool
	IsUnique        bool
	DefaultValue    string
}

type parsedForeignKey struct {
	Name       string
	FromTable  string
	FromColumn string
	ToTable    string
	ToColumn   string
	OnDelete   string
	OnUpdate   string
}

type parsedTable struct {
	Name    string
	Columns []parsedColumn
	FKs     []parsedForeignKey
	Indexes []parsedIndex
}

var (
	reCreateTable            = regexp.MustCompile("(?is)CREATE\\s+TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?[`\"]?(?P<name>[a-zA-Z0-9_]+)[`\"]?\\s*\\(")
	reAlterFK                = regexp.MustCompile("(?is)ALTER\\s+TABLE\\s+[`\"]?(?P<from>[a-zA-Z0-9_]+)[`\"]?\\s+ADD\\s+(?:CONSTRAINT\\s+[`\"]?(?P<cname>[a-zA-Z0-9_]+)[`\"]?\\s+)?FOREIGN\\s+KEY\\s*\\((?P<fc>[^)]+)\\)\\s*REFERENCES\\s+[`\"]?(?P<to>[a-zA-Z0-9_]+)[`\"]?\\s*\\((?P<tc>[^)]+)\\)")
	reFKInline               = regexp.MustCompile("(?is)FOREIGN\\s+KEY\\s*\\((?P<fc>[^)]+)\\)\\s*REFERENCES\\s+[`\"]?(?P<to>[a-zA-Z0-9_]+)[`\"]?\\s*\\((?P<tc>[^)]+)\\)")
	reFKConstraint           = regexp.MustCompile("(?is)CONSTRAINT\\s+[`\"]?(?P<name>[a-zA-Z0-9_]+)[`\"]?\\s+FOREIGN\\s+KEY")
	reColNameType            = regexp.MustCompile(`(?i)(?:^|\s)[` + "`\"" + `]?([a-zA-Z_][a-zA-Z0-9_]*)[` + "`\"" + `]?\s+(INT|INTEGER|BIGINT|SMALLINT|TINYINT|MEDIUMINT|SERIAL|BIGSERIAL|SMALLSERIAL|VARCHAR|CHAR|TEXT|LONGTEXT|MEDIUMTEXT|TINYTEXT|BOOLEAN|BOOL|DATE|DATETIME|TIMESTAMP|TIME|DECIMAL|NUMERIC|FLOAT|DOUBLE|REAL|JSON|JSONB|BLOB|BYTEA|UUID|BIT|ENUM|SET)\b`)
	reCreateTableKeyword     = regexp.MustCompile(`(?is)\bCREATE\s+TABLE\b`)
	reCreateTableMissingName = regexp.MustCompile(`(?is)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?\s*\(`)
	reCreatePgEnum           = regexp.MustCompile(`(?is)CREATE\s+TYPE\s+(?:IF\s+NOT\s+EXISTS\s+)?[` + "`\"" + `]?([a-zA-Z_][a-zA-Z0-9_]*)[` + "`\"" + `]?\s+AS\s+ENUM\s*\(([^)]*)\)`)
	reDesignerFKComment      = regexp.MustCompile(`(?i)--\s*FK:\s*(?P<name>\S+)\s*\((?P<actions>[^)]*)\)\s*(?P<fromtable>[a-zA-Z0-9_]+)\.(?P<fromcol>[a-zA-Z0-9_]+)\s*->\s*(?P<totable>[a-zA-Z0-9_]+)\.(?P<tocol>[a-zA-Z0-9_]+)`)
)

// SQLParseError describes a DDL parse failure with optional source location.
type SQLParseError struct {
	Line    int
	Column  int
	Message string
	Snippet string
}

func (e *SQLParseError) Error() string {
	if e == nil {
		return ""
	}
	if e.Line > 0 {
		if e.Column > 0 {
			return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
		}
		return fmt.Sprintf("line %d: %s", e.Line, e.Message)
	}
	return e.Message
}

func (e *SQLParseError) locateIn(original string) {
	if e == nil || e.Line > 0 || strings.TrimSpace(e.Snippet) == "" {
		return
	}
	snippet := strings.TrimSpace(e.Snippet)
	for _, needle := range []string{snippet, strings.TrimSpace(strings.Split(snippet, "\n")[0])} {
		if needle == "" {
			continue
		}
		idx := strings.Index(original, needle)
		if idx < 0 {
			continue
		}
		e.Line = 1 + strings.Count(original[:idx], "\n")
		lastNL := strings.LastIndex(original[:idx], "\n")
		e.Column = idx - lastNL
		if e.Column <= 0 {
			e.Column = 1
		}
		return
	}
}

// ParseSQL extracts tables, columns, and foreign keys from SQL DDL
func ParseSQL(sql string) ([]parsedTable, []parsedForeignKey, error) {
	commentFKs := parseDesignerFKComments(sql)
	sql = stripSQLComments(sql)
	if err := validateCreateTableStatements(sql); err != nil {
		return nil, nil, err
	}
	pgEnumTypes := parsePostgresEnumTypes(sql)
	var extraFKs []parsedForeignKey

	for _, m := range reAlterFK.FindAllStringSubmatch(sql, -1) {
		onDelete, onUpdate := ParseFKActions(m[0])
		fromTable := m[1]
		fromCol := cleanIdent(m[3])
		name := m[2]
		if name == "" {
			name = DefaultFKName(fromTable, fromCol)
		}
		extraFKs = append(extraFKs, parsedForeignKey{
			Name:       name,
			FromTable:  fromTable,
			FromColumn: fromCol,
			ToTable:    m[4],
			ToColumn:   cleanIdent(m[5]),
			OnDelete:   onDelete,
			OnUpdate:   onUpdate,
		})
	}

	var tables []parsedTable
	loc := reCreateTable.FindAllStringSubmatchIndex(sql, -1)
	for i, idx := range loc {
		name := sql[idx[2]:idx[3]]
		bodyEnd := len(sql)
		if i+1 < len(loc) {
			bodyEnd = loc[i+1][0]
		}
		// idx[1] is the byte after the table's opening "(" — use that paren, not the first
		// "(" inside a column type such as VARCHAR(150) or DECIMAL(15,2).
		openParen := idx[1] - 1
		if openParen < 0 || openParen >= len(sql) || sql[openParen] != '(' {
			return nil, extraFKs, &SQLParseError{
				Message: fmt.Sprintf("invalid CREATE TABLE syntax for %q", name),
				Snippet: sql[idx[0]:minInt(bodyEnd, idx[0]+120)],
			}
		}
		colBlock, ok := extractParenBlock(sql[openParen:bodyEnd])
		if !ok {
			return nil, extraFKs, &SQLParseError{
				Message: fmt.Sprintf("unclosed column list in CREATE TABLE %q", name),
				Snippet: sql[idx[0]:minInt(bodyEnd, idx[0]+120)],
			}
		}
		pt := parsedTable{Name: name}
		var err error
		pt.Columns, pt.FKs, pt.Indexes, err = parseColumnBlock(colBlock, name, pgEnumTypes)
		if err != nil {
			return nil, extraFKs, err
		}
		tables = append(tables, pt)
	}

	if len(tables) == 0 {
		return nil, extraFKs, fmt.Errorf("no CREATE TABLE statements found")
	}

	extraFKs = appendUniqueForeignKeys(extraFKs, commentFKs...)
	return tables, extraFKs, nil
}

func parseDesignerFKComments(sql string) []parsedForeignKey {
	var out []parsedForeignKey
	for _, m := range reDesignerFKComment.FindAllStringSubmatch(sql, -1) {
		if len(m) < 7 {
			continue
		}
		onDelete, onUpdate := ParseFKActions(m[2])
		out = append(out, parsedForeignKey{
			Name:       m[1],
			FromTable:  m[3],
			FromColumn: m[4],
			ToTable:    m[5],
			ToColumn:   m[6],
			OnDelete:   onDelete,
			OnUpdate:   onUpdate,
		})
	}
	return out
}

func foreignKeyIdentity(fk parsedForeignKey) string {
	return strings.ToLower(fk.FromTable + "." + fk.FromColumn + "->" + fk.ToTable + "." + fk.ToColumn)
}

func appendUniqueForeignKeys(list []parsedForeignKey, add ...parsedForeignKey) []parsedForeignKey {
	if len(add) == 0 {
		return list
	}
	seen := make(map[string]struct{}, len(list)+len(add))
	for _, fk := range list {
		seen[foreignKeyIdentity(fk)] = struct{}{}
	}
	for _, fk := range add {
		key := foreignKeyIdentity(fk)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		list = append(list, fk)
	}
	return list
}

// ValidateSQL checks whether DDL can be parsed without applying it.
func ValidateSQL(sql string) error {
	if strings.TrimSpace(sql) == "" {
		return fmt.Errorf("sql is required")
	}
	_, _, err := ParseSQL(sql)
	if err != nil {
		if pe, ok := err.(*SQLParseError); ok {
			pe.locateIn(sql)
		}
	}
	return err
}

func validateCreateTableStatements(sql string) *SQLParseError {
	for _, idx := range reCreateTableKeyword.FindAllStringIndex(sql, -1) {
		rest := sql[idx[0]:]
		firstLine := strings.TrimSpace(strings.Split(rest, "\n")[0])
		if reCreateTableMissingName.MatchString(rest) {
			return &SQLParseError{
				Message: "CREATE TABLE requires a table name",
				Snippet: firstLine,
			}
		}
		if loc := reCreateTable.FindStringSubmatchIndex(rest); loc == nil || loc[0] != 0 {
			return &SQLParseError{
				Message: "invalid CREATE TABLE statement",
				Snippet: firstLine,
			}
		}
	}
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func stripSQLComments(sql string) string {
	lines := strings.Split(sql, "\n")
	var out []string
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func extractParenBlock(s string) (string, bool) {
	if len(s) < 2 || s[0] != '(' {
		return "", false
	}
	depth := 0
	for i, ch := range s {
		switch ch {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return s[1:i], true
			}
		}
	}
	return "", false
}

func parsePostgresEnumTypes(sql string) map[string][]string {
	out := make(map[string][]string)
	for _, m := range reCreatePgEnum.FindAllStringSubmatch(sql, -1) {
		if len(m) < 3 {
			continue
		}
		typeName := strings.ToLower(cleanIdent(m[1]))
		values := parseEnumValuesFromDataType("ENUM(" + m[2] + ")")
		if typeName == "" || len(values) == 0 {
			continue
		}
		out[typeName] = values
	}
	return out
}

func resolveImportedColumnType(dataType, tableName, columnName string, pgEnums map[string][]string) string {
	dt := strings.TrimSpace(dataType)
	if len(parseEnumValuesFromDataType(dt)) > 0 {
		return normalizeImportedDataType(dt)
	}
	keys := []string{
		strings.ToLower(cleanIdent(dt)),
		strings.ToLower(postgresEnumTypeName(tableName, columnName)),
	}
	for _, key := range keys {
		if key == "" {
			continue
		}
		if values, ok := pgEnums[key]; ok {
			return normalizeImportedDataType(formatEnumDataType(values))
		}
	}
	return normalizeImportedDataType(dt)
}

// normalizeImportedDataType maps dialect-specific DDL types back to canvas canonical types.
func normalizeImportedDataType(dataType string) string {
	dt := strings.TrimSpace(dataType)
	if dt == "" {
		return "TEXT"
	}
	if len(parseEnumValuesFromDataType(dt)) > 0 {
		return dt
	}
	upper := strings.ToUpper(strings.Join(strings.Fields(dt), " "))
	if strings.HasPrefix(upper, "DOUBLE PRECISION") {
		_, suffix := splitTypeBaseSuffix(dt)
		return "DOUBLE" + suffix
	}
	base, suffix := splitTypeBaseSuffix(dt)
	switch strings.ToUpper(strings.TrimSpace(base)) {
	case "INTEGER":
		return "INT" + suffix
	case "BOOL":
		return "BOOLEAN" + suffix
	case "JSONB":
		return "JSON" + suffix
	case "BYTEA":
		return "BLOB" + suffix
	case "REAL":
		return "FLOAT" + suffix
	case "NUMERIC":
		return "DECIMAL" + suffix
	case "SERIAL":
		return "INT" + suffix
	case "BIGSERIAL":
		return "BIGINT" + suffix
	case "SMALLSERIAL":
		return "SMALLINT" + suffix
	case "MEDIUMINT":
		return "INT" + suffix
	case "LONGTEXT", "MEDIUMTEXT", "TINYTEXT":
		return strings.ToUpper(base) + suffix
	case "LONGBLOB", "MEDIUMBLOB", "TINYBLOB":
		return "BLOB" + suffix
	default:
		return dt
	}
}

func parseColumnBlock(block string, tableName string, pgEnums map[string][]string) ([]parsedColumn, []parsedForeignKey, []parsedIndex, error) {
	var cols []parsedColumn
	var fks []parsedForeignKey
	var indexes []parsedIndex
	seenCols := make(map[string]struct{})

	parts := splitColumnDefs(block)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		upper := strings.ToUpper(part)

		if strings.HasPrefix(upper, "PRIMARY KEY") {
			continue
		}
		if pi, ok := parseIndexLine(part); ok {
			indexes = append(indexes, pi)
			continue
		}

		if m := reFKInline.FindStringSubmatch(part); len(m) == 4 {
			onDelete, onUpdate := ParseFKActions(part)
			name := extractFKConstraintName(part)
			fks = append(fks, parsedForeignKey{
				Name:       name,
				FromColumn: cleanIdent(m[1]),
				ToTable:    m[2],
				ToColumn:   cleanIdent(m[3]),
				OnDelete:   onDelete,
				OnUpdate:   onUpdate,
			})
			continue
		}

		if countColumnDefinitions(part) > 1 {
			firstLine := strings.TrimSpace(strings.Split(part, "\n")[0])
			return nil, nil, nil, &SQLParseError{
				Message: fmt.Sprintf("missing comma between column definitions in table %q", tableName),
				Snippet: firstLine,
			}
		}

		col := parseColumnLine(part)
		if col.Name != "" {
			col.DataType = resolveImportedColumnType(col.DataType, tableName, col.Name, pgEnums)
			key := strings.ToLower(col.Name)
			if _, dup := seenCols[key]; dup {
				firstLine := strings.TrimSpace(strings.Split(part, "\n")[0])
				return nil, nil, nil, &SQLParseError{
					Message: fmt.Sprintf("duplicate column name %q in table %q", col.Name, tableName),
					Snippet: firstLine,
				}
			}
			seenCols[key] = struct{}{}
			cols = append(cols, col)
			continue
		}

		return nil, nil, nil, &SQLParseError{
			Message: fmt.Sprintf("unrecognized column definition in table %q", tableName),
			Snippet: part,
		}
	}
	return cols, fks, indexes, nil
}

func countColumnDefinitions(part string) int {
	flat := flattenParenContent(part)
	return len(reColNameType.FindAllStringIndex(flat, -1))
}

func flattenParenContent(s string) string {
	var b strings.Builder
	depth := 0
	for _, ch := range s {
		switch ch {
		case '(':
			depth++
			if depth == 1 {
				b.WriteByte(' ')
				continue
			}
		case ')':
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func extractFKConstraintName(part string) string {
	if m := reFKConstraint.FindStringSubmatch(part); len(m) > 1 {
		return m[1]
	}
	return ""
}

func parseIndexLine(part string) (parsedIndex, bool) {
	upper := strings.ToUpper(strings.TrimSpace(part))
	isUnique := strings.Contains(upper, "UNIQUE")

	switch {
	case strings.HasPrefix(upper, "UNIQUE KEY"), strings.HasPrefix(upper, "UNIQUE INDEX"):
		isUnique = true
	case strings.HasPrefix(upper, "KEY "), strings.HasPrefix(upper, "INDEX "):
		isUnique = false
	case strings.HasPrefix(upper, "CONSTRAINT "):
		if !strings.Contains(upper, "UNIQUE") || strings.Contains(upper, "FOREIGN KEY") {
			return parsedIndex{}, false
		}
		isUnique = true
	default:
		return parsedIndex{}, false
	}

	open := strings.LastIndex(part, "(")
	close := strings.LastIndex(part, ")")
	if open < 0 || close <= open {
		return parsedIndex{}, false
	}
	colList := part[open+1 : close]
	cols := splitIndexColumns(colList)
	if len(cols) == 0 {
		return parsedIndex{}, false
	}

	head := strings.TrimSpace(part[:open])
	name := extractIndexName(head)
	return parsedIndex{Name: name, IsUnique: isUnique, Columns: cols}, true
}

func extractIndexName(head string) string {
	head = strings.TrimSpace(head)
	upper := strings.ToUpper(head)
	for _, prefix := range []string{"CONSTRAINT", "UNIQUE", "KEY", "INDEX"} {
		if strings.HasPrefix(upper, prefix) {
			head = strings.TrimSpace(head[len(prefix):])
			upper = strings.ToUpper(head)
		}
	}
	if strings.HasPrefix(upper, "KEY ") || strings.HasPrefix(upper, "INDEX ") {
		head = strings.TrimSpace(head[strings.Index(head, " ")+1:])
	}
	parts := strings.Fields(head)
	if len(parts) == 0 {
		return ""
	}
	return cleanIdent(parts[0])
}

func splitIndexColumns(list string) []string {
	var cols []string
	for _, p := range strings.Split(list, ",") {
		name := cleanIdent(strings.TrimSpace(p))
		if name != "" {
			cols = append(cols, name)
		}
	}
	return cols
}

func splitColumnDefs(block string) []string {
	var parts []string
	var cur strings.Builder
	depth := 0
	for _, ch := range block {
		switch ch {
		case '(':
			depth++
			cur.WriteRune(ch)
		case ')':
			depth--
			cur.WriteRune(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, cur.String())
				cur.Reset()
				continue
			}
			cur.WriteRune(ch)
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

func parseColumnLine(line string) parsedColumn {
	upper := strings.ToUpper(line)
	col := parsedColumn{IsNullable: true}

	if strings.Contains(upper, "PRIMARY KEY") {
		col.IsPrimaryKey = true
		col.IsNullable = false
	}
	if strings.Contains(upper, "NOT NULL") {
		col.IsNullable = false
	}
	if strings.Contains(upper, " UNIQUE") || strings.HasPrefix(upper, "UNIQUE ") {
		col.IsUnique = true
	}
	if strings.Contains(upper, "AUTO_INCREMENT") || strings.Contains(upper, "AUTOINCREMENT") {
		col.IsAutoIncrement = true
	}

	// name first token(s) before type
	tokens := strings.Fields(line)
	if len(tokens) == 0 {
		return col
	}
	col.Name = cleanIdent(tokens[0])
	if len(tokens) < 2 {
		col.DataType = "TEXT"
		return col
	}

	typeParts := []string{}
	for i := 1; i < len(tokens); i++ {
		t := strings.ToUpper(tokens[i])
		if t == "PRIMARY" || t == "KEY" || t == "NOT" || t == "NULL" ||
			t == "UNIQUE" || t == "DEFAULT" || t == "AUTO_INCREMENT" ||
			t == "AUTOINCREMENT" || strings.HasPrefix(t, "REFERENCES") {
			if t == "DEFAULT" && i+1 < len(tokens) {
				col.DefaultValue = tokens[i+1]
			}
			break
		}
		typeParts = append(typeParts, tokens[i])
	}
	col.DataType = strings.Join(typeParts, " ")
	if col.DataType == "" {
		col.DataType = "TEXT"
	}
	if isSerialDataType(col.DataType) {
		col.IsAutoIncrement = true
	}
	return col
}

func isSerialDataType(dt string) bool {
	u := strings.ToUpper(strings.TrimSpace(dt))
	switch u {
	case "SERIAL", "BIGSERIAL", "SMALLSERIAL":
		return true
	default:
		return false
	}
}

func cleanIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"[]")
	return s
}
