package dbdesign

import (
	"fmt"
	"strings"
)

// FormatSQL normalizes valid DDL using the same layout as export.
func FormatSQL(sql, dialect string) (string, error) {
	dialect = normalizeDialect(dialect)
	if dialect == "" {
		dialect = DialectMySQL
	}
	comments := collectSQLCommentLayout(sql)
	tables, alterFKs, err := ParseSQL(sql)
	if err != nil {
		return "", err
	}
	schema := schemaFromParsed(tables, alterFKs, dialect)
	formatted := strings.TrimRight(exportSQLBody(schema, dialect), "\n")
	return applySQLCommentLayout(formatted, comments), nil
}

func schemaFromParsed(tables []parsedTable, alterFKs []parsedForeignKey, dialect string) *Schema {
	schema := &Schema{Project: Project{Dialect: dialect}}
	tableMap := make(map[string]uint)
	columnMap := make(map[string]uint)
	var nextTableID, nextColID, nextRelID, nextIdxID uint = 1, 1, 1, 1

	for _, pt := range tables {
		tbl := TableDTO{Table: Table{ID: nextTableID, Name: pt.Name}}
		nextTableID++
		tableMap[strings.ToLower(pt.Name)] = tbl.ID

		for si, pc := range pt.Columns {
			col := Column{
				ID:              nextColID,
				TableID:         tbl.ID,
				Name:            pc.Name,
				DataType:        pc.DataType,
				IsPrimaryKey:    pc.IsPrimaryKey,
				IsAutoIncrement: pc.IsAutoIncrement,
				IsNullable:      pc.IsNullable,
				IsUnique:        pc.IsUnique,
				DefaultValue:    pc.DefaultValue,
				SortOrder:       si,
			}
			nextColID++
			tbl.Columns = append(tbl.Columns, col)
			columnMap[strings.ToLower(pt.Name+"."+pc.Name)] = col.ID
		}

		for ii, pi := range pt.Indexes {
			idx := IndexDTO{
				Index: Index{
					ID:        nextIdxID,
					TableID:   tbl.ID,
					IsUnique:  pi.IsUnique,
					SortOrder: ii,
				},
			}
			nextIdxID++
			idx.Name = strings.TrimSpace(pi.Name)
			if idx.Name == "" {
				idx.Name = fmt.Sprintf("idx_%s_%d", pt.Name, ii)
			}
			for _, colName := range pi.Columns {
				key := strings.ToLower(pt.Name + "." + colName)
				if id, ok := columnMap[key]; ok {
					idx.ColumnIDs = append(idx.ColumnIDs, id)
				}
			}
			if len(idx.ColumnIDs) > 0 {
				tbl.Indexes = append(tbl.Indexes, idx)
			}
		}

		schema.Tables = append(schema.Tables, tbl)
		for _, fk := range pt.FKs {
			fk.FromTable = pt.Name
			appendParsedRelation(schema, &nextRelID, tableMap, columnMap, fk)
		}
	}

	for _, fk := range alterFKs {
		appendParsedRelation(schema, &nextRelID, tableMap, columnMap, fk)
	}
	return schema
}

func appendParsedRelation(schema *Schema, nextRelID *uint, tableMap, columnMap map[string]uint, fk parsedForeignKey) {
	fromTable := strings.ToLower(fk.FromTable)
	toTable := strings.ToLower(fk.ToTable)
	ftID, ok1 := tableMap[fromTable]
	ttID, ok2 := tableMap[toTable]
	fcKey := fromTable + "." + strings.ToLower(fk.FromColumn)
	tcKey := toTable + "." + strings.ToLower(fk.ToColumn)
	fcID, ok3 := columnMap[fcKey]
	tcID, ok4 := columnMap[tcKey]
	if !ok1 || !ok2 || !ok3 || !ok4 {
		return
	}
	name := strings.TrimSpace(fk.Name)
	if name == "" {
		name = DefaultFKName(fk.FromTable, fk.FromColumn)
	}
	schema.Relations = append(schema.Relations, Relation{
		ID:           *nextRelID,
		Name:         name,
		FromTableID:  ftID,
		FromColumnID: fcID,
		ToTableID:    ttID,
		ToColumnID:   tcID,
		OnDelete:     fk.OnDelete,
		OnUpdate:     fk.OnUpdate,
	})
	*nextRelID++
}
