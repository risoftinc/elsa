package dbdesign

import (
	"regexp"
	"strings"
)

var reLineIdent = regexp.MustCompile("^[`\"]?([a-zA-Z_][a-zA-Z0-9_]*)[`\"]?")

type sqlCommentLayout struct {
	leading         []string
	beforeStatement map[string][]string
	lineTrailing    map[string]string
	remaining       []string
}

func collectSQLCommentLayout(sql string) sqlCommentLayout {
	layout := sqlCommentLayout{
		beforeStatement: make(map[string][]string),
		lineTrailing:    make(map[string]string),
	}
	lines := strings.Split(sql, "\n")
	var pending []string

	flushPendingToLeading := func() {
		if len(pending) == 0 {
			return
		}
		layout.leading = append(layout.leading, pending...)
		pending = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(pending) > 0 {
				pending = append(pending, "")
			}
			continue
		}
		if strings.HasPrefix(trimmed, "--") {
			pending = append(pending, line)
			continue
		}

		code, trail := splitTrailingLineComment(line)
		anchor := statementCommentAnchor(code)
		if anchor != "" && len(pending) > 0 {
			key := normalizeStatementAnchor(anchor)
			layout.beforeStatement[key] = append(layout.beforeStatement[key], pending...)
			pending = nil
		} else if len(pending) > 0 {
			flushPendingToLeading()
		}

		if trail != "" {
			layout.lineTrailing[lineCommentKey(code)] = trail
		}
	}

	if len(pending) > 0 {
		layout.remaining = append(layout.remaining, pending...)
	}
	return layout
}

func applySQLCommentLayout(formatted string, layout sqlCommentLayout) string {
	var out []string
	out = append(out, layout.leading...)

	usedTrailing := make(map[string]bool)

	for _, line := range strings.Split(formatted, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			if anchor := statementCommentAnchor(trimmed); anchor != "" {
				key := normalizeStatementAnchor(anchor)
				if blocks, ok := layout.beforeStatement[key]; ok {
					out = append(out, blocks...)
					delete(layout.beforeStatement, key)
				}
			}

			key := lineCommentKey(trimmed)
			if trail, ok := layout.lineTrailing[key]; ok && !usedTrailing[key] {
				line = appendTrailingLineComment(line, trail)
				usedTrailing[key] = true
			}
		}
		out = append(out, line)
	}

	for _, blocks := range layout.beforeStatement {
		out = append(out, blocks...)
	}
	out = append(out, layout.remaining...)
	return strings.Join(out, "\n")
}

func splitTrailingLineComment(line string) (code, comment string) {
	idx := strings.Index(line, "--")
	if idx < 0 {
		return line, ""
	}
	return line[:idx], strings.TrimSpace(line[idx:])
}

func statementCommentAnchor(code string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return ""
	}
	upper := strings.ToUpper(trimmed)
	switch {
	case strings.HasPrefix(upper, "CREATE TABLE"):
		if m := reCreateTable.FindStringSubmatch(trimmed); len(m) > 1 {
			return "create table " + strings.ToLower(m[1])
		}
	case strings.HasPrefix(upper, "ALTER TABLE"):
		re := regexp.MustCompile(`(?is)^ALTER\s+TABLE\s+[` + "`\"" + `]?([a-zA-Z0-9_]+)`)
		if m := re.FindStringSubmatch(trimmed); len(m) > 1 {
			return "alter table " + strings.ToLower(m[1])
		}
	case strings.HasPrefix(upper, "CREATE UNIQUE INDEX"), strings.HasPrefix(upper, "CREATE INDEX"):
		return normalizeCodeLine(trimmed)
	}
	return ""
}

func normalizeStatementAnchor(anchor string) string {
	return strings.ToLower(strings.Join(strings.Fields(anchor), " "))
}

func lineCommentKey(code string) string {
	trimmed := strings.TrimSpace(code)
	trimmed = strings.TrimSuffix(trimmed, ",")
	trimmed = strings.TrimSpace(trimmed)
	if m := reLineIdent.FindStringSubmatch(trimmed); len(m) > 1 && strings.HasPrefix(trimmed, m[0]) {
		return "col:" + strings.ToLower(m[1])
	}
	return "line:" + normalizeCodeLine(trimmed)
}

func normalizeCodeLine(code string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(code)), " "))
}

func appendTrailingLineComment(line, comment string) string {
	if strings.TrimSpace(comment) == "" {
		return line
	}
	trimmed := strings.TrimRight(line, " \t")
	hadComma := strings.HasSuffix(trimmed, ",")
	if hadComma {
		trimmed = strings.TrimSuffix(trimmed, ",")
		trimmed = strings.TrimRight(trimmed, " \t")
	}
	result := trimmed + " " + comment
	if hadComma {
		result += ","
	}
	return result
}
