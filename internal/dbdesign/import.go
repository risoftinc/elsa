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
	Name         string
	DataType     string
	IsPrimaryKey bool
	IsNullable   bool
	IsUnique     bool
	DefaultValue string
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
	reCreateTable  = regexp.MustCompile("(?is)CREATE\\s+TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?[`\"]?(?P<name>[a-zA-Z0-9_]+)[`\"]?\\s*\\(")
	reAlterFK      = regexp.MustCompile("(?is)ALTER\\s+TABLE\\s+[`\"]?(?P<from>[a-zA-Z0-9_]+)[`\"]?\\s+ADD\\s+(?:CONSTRAINT\\s+[`\"]?(?P<cname>[a-zA-Z0-9_]+)[`\"]?\\s+)?FOREIGN\\s+KEY\\s*\\((?P<fc>[^)]+)\\)\\s*REFERENCES\\s+[`\"]?(?P<to>[a-zA-Z0-9_]+)[`\"]?\\s*\\((?P<tc>[^)]+)\\)")
	reFKInline     = regexp.MustCompile("(?is)FOREIGN\\s+KEY\\s*\\((?P<fc>[^)]+)\\)\\s*REFERENCES\\s+[`\"]?(?P<to>[a-zA-Z0-9_]+)[`\"]?\\s*\\((?P<tc>[^)]+)\\)")
	reFKConstraint = regexp.MustCompile("(?is)CONSTRAINT\\s+[`\"]?(?P<name>[a-zA-Z0-9_]+)[`\"]?\\s+FOREIGN\\s+KEY")
)

// ParseSQL extracts tables, columns, and foreign keys from SQL DDL
func ParseSQL(sql string) ([]parsedTable, []parsedForeignKey, error) {
	sql = stripSQLComments(sql)
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
			continue
		}
		colBlock, ok := extractParenBlock(sql[openParen:bodyEnd])
		if !ok {
			continue
		}
		pt := parsedTable{Name: name}
		pt.Columns, pt.FKs, pt.Indexes = parseColumnBlock(colBlock)
		tables = append(tables, pt)
	}

	if len(tables) == 0 {
		return nil, extraFKs, fmt.Errorf("no CREATE TABLE statements found")
	}

	return tables, extraFKs, nil
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

func parseColumnBlock(block string) ([]parsedColumn, []parsedForeignKey, []parsedIndex) {
	var cols []parsedColumn
	var fks []parsedForeignKey
	var indexes []parsedIndex

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

		col := parseColumnLine(part)
		if col.Name != "" {
			cols = append(cols, col)
		}
	}
	return cols, fks, indexes
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
	return col
}

func cleanIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"[]")
	return s
}
