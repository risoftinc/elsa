package dbdesign

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// --- Projects ---

func (s *Store) ListProjects() ([]Project, error) {
	var list []Project
	err := s.db.Order("name asc").Find(&list).Error
	return list, err
}

func (s *Store) CreateProject(name, description, dialect string) (*Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	dialect = normalizeDialect(dialect)
	taken, err := s.projectNameExists(name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("project name %q already exists", name)
	}
	p := &Project{Name: name, Description: strings.TrimSpace(description), Dialect: dialect}
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) GetProject(id uint) (*Project, error) {
	var p Project
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *Store) UpdateProject(id uint, name, description, dialect string) (*Project, error) {
	var p Project
	if err := s.db.First(&p, id).Error; err != nil {
		return nil, ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("project name is required")
	}
	taken, err := s.projectNameExists(name, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("project name %q already exists", name)
	}
	p.Name = name
	p.Description = strings.TrimSpace(description)
	if dialect := normalizeDialect(dialect); dialect != "" {
		p.Dialect = dialect
	}
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) DeleteProject(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", id).Delete(&Relation{}).Error; err != nil {
			return err
		}
		var tables []Table
		if err := tx.Where("project_id = ?", id).Find(&tables).Error; err != nil {
			return err
		}
		for _, t := range tables {
			if err := (&Store{db: tx}).deleteIndexesForTable(tx, t.ID); err != nil {
				return err
			}
			if err := tx.Where("table_id = ?", t.ID).Delete(&Column{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("project_id = ?", id).Delete(&Table{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Project{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *Store) projectNameExists(name string, excludeID uint) (bool, error) {
	var count int64
	q := s.db.Model(&Project{}).Where("LOWER(name) = LOWER(?)", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func normalizeDialect(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case DialectPostgres, "postgresql":
		return DialectPostgres
	case DialectSQLite:
		return DialectSQLite
	default:
		return DialectMySQL
	}
}

// GetSchema returns full project design
func (s *Store) GetSchema(projectID uint) (*Schema, error) {
	p, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	var tables []Table
	if err := s.db.Where("project_id = ?", projectID).Order("name asc").Find(&tables).Error; err != nil {
		return nil, err
	}
	dtos := make([]TableDTO, 0, len(tables))
	for _, t := range tables {
		var cols []Column
		if err := s.db.Where("table_id = ?", t.ID).Order("sort_order asc, id asc").Find(&cols).Error; err != nil {
			return nil, err
		}
		idxs, err := s.loadIndexesForTable(t.ID)
		if err != nil {
			return nil, err
		}
		dtos = append(dtos, TableDTO{Table: t, Columns: cols, Indexes: idxs})
	}
	var rels []Relation
	if err := s.db.Where("project_id = ?", projectID).Find(&rels).Error; err != nil {
		return nil, err
	}
	for i := range rels {
		rels[i].OnDelete = NormalizeFKActionDefault(rels[i].OnDelete)
		rels[i].OnUpdate = NormalizeFKActionDefault(rels[i].OnUpdate)
		if strings.TrimSpace(rels[i].Name) == "" {
			rels[i].Name = defaultRelationName(dtos, rels[i])
		}
	}
	return &Schema{Project: *p, Tables: dtos, Relations: rels}, nil
}

// --- Tables ---

func (s *Store) CreateTable(projectID uint, name string, posX, posY float64) (*TableDTO, error) {
	proj, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("table name is required")
	}
	t := &Table{ProjectID: projectID, Name: name, PosX: posX, PosY: posY}
	if err := s.db.Create(t).Error; err != nil {
		return nil, err
	}
	// default PK id column
	col := &Column{
		TableID:         t.ID,
		Name:            "id",
		DataType:        defaultPKType(proj.Dialect),
		IsPrimaryKey:    true,
		IsAutoIncrement: true,
		IsNullable:      false,
		SortOrder:       0,
	}
	if err := s.db.Create(col).Error; err != nil {
		return nil, err
	}
	return &TableDTO{Table: *t, Columns: []Column{*col}, Indexes: []IndexDTO{}}, nil
}

func defaultPKType(dialect string) string {
	switch dialect {
	case DialectPostgres:
		return "BIGSERIAL"
	case DialectSQLite:
		return "INTEGER"
	default:
		return "BIGINT"
	}
}

func (s *Store) UpdateTable(id uint, name string, posX, posY, width *float64) (*Table, error) {
	var t Table
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, ErrNotFound
	}
	if name = strings.TrimSpace(name); name != "" {
		t.Name = name
	}
	if posX != nil {
		t.PosX = *posX
	}
	if posY != nil {
		t.PosY = *posY
	}
	if width != nil {
		t.Width = clampCanvasTableWidth(*width)
	}
	if err := s.db.Save(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func clampCanvasTableWidth(w float64) float64 {
	if w <= 0 {
		return 0
	}
	const minW, maxW = 200, 320
	if w < minW {
		return minW
	}
	if w > maxW {
		return maxW
	}
	return w
}

func (s *Store) DeleteTable(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var t Table
		if err := tx.First(&t, id).Error; err != nil {
			return ErrNotFound
		}
		if err := tx.Where("from_table_id = ? OR to_table_id = ?", id, id).Delete(&Relation{}).Error; err != nil {
			return err
		}
		if err := (&Store{db: tx}).deleteIndexesForTable(tx, id); err != nil {
			return err
		}
		if err := tx.Where("table_id = ?", id).Delete(&Column{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Table{}, id).Error
	})
}

// --- Columns ---

type ColumnPayload struct {
	Name            string `json:"name"`
	DataType        string `json:"data_type"`
	IsPrimaryKey    bool   `json:"is_primary_key"`
	IsForeignKey    bool   `json:"is_foreign_key"`
	IsAutoIncrement bool   `json:"is_auto_increment"`
	IsNullable      bool   `json:"is_nullable"`
	IsUnique        bool   `json:"is_unique"`
	DefaultValue    string `json:"default_value"`
	SortOrder       int    `json:"sort_order"`
}

func (s *Store) CreateColumn(tableID uint, p ColumnPayload) (*Column, error) {
	var t Table
	if err := s.db.First(&t, tableID).Error; err != nil {
		return nil, ErrNotFound
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, fmt.Errorf("column name is required")
	}
	dt := strings.TrimSpace(p.DataType)
	if dt == "" {
		return nil, fmt.Errorf("data type is required")
	}
	defVal := strings.TrimSpace(p.DefaultValue)
	if err := ValidateColumnDefault(dt, defVal, p.IsPrimaryKey); err != nil {
		return nil, err
	}
	if p.IsPrimaryKey {
		p.IsUnique = false
		s.db.Model(&Column{}).Where("table_id = ? AND is_primary_key = ?", tableID, true).
			Update("is_primary_key", false)
	}
	if p.IsAutoIncrement {
		s.db.Model(&Column{}).Where("table_id = ? AND is_auto_increment = ?", tableID, true).
			Update("is_auto_increment", false)
	}
	c := &Column{
		TableID:         tableID,
		Name:            name,
		DataType:        dt,
		IsPrimaryKey:    p.IsPrimaryKey,
		IsForeignKey:    p.IsForeignKey,
		IsAutoIncrement: p.IsAutoIncrement,
		IsNullable:      p.IsNullable,
		IsUnique:        p.IsUnique,
		DefaultValue:    defVal,
		SortOrder:       p.SortOrder,
	}
	if err := s.db.Create(c).Error; err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Store) UpdateColumn(id uint, p ColumnPayload) (*Column, error) {
	var c Column
	if err := s.db.First(&c, id).Error; err != nil {
		return nil, ErrNotFound
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return nil, fmt.Errorf("column name is required")
	}
	dt := strings.TrimSpace(p.DataType)
	if dt == "" {
		return nil, fmt.Errorf("data type is required")
	}
	defVal := strings.TrimSpace(p.DefaultValue)
	if err := ValidateColumnDefault(dt, defVal, p.IsPrimaryKey); err != nil {
		return nil, err
	}
	if p.IsPrimaryKey {
		p.IsUnique = false
		s.db.Model(&Column{}).Where("table_id = ? AND is_primary_key = ? AND id <> ?", c.TableID, true, id).
			Update("is_primary_key", false)
	}
	if p.IsAutoIncrement {
		s.db.Model(&Column{}).Where("table_id = ? AND is_auto_increment = ? AND id <> ?", c.TableID, true, id).
			Update("is_auto_increment", false)
	}
	c.Name = name
	c.DataType = dt
	c.IsPrimaryKey = p.IsPrimaryKey
	c.IsForeignKey = p.IsForeignKey
	c.IsAutoIncrement = p.IsAutoIncrement
	c.IsNullable = p.IsNullable
	c.IsUnique = p.IsUnique
	c.DefaultValue = defVal
	c.SortOrder = p.SortOrder
	if err := s.db.Save(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) DeleteColumn(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("from_column_id = ? OR to_column_id = ?", id, id).Delete(&Relation{}).Error; err != nil {
			return err
		}
		if err := (&Store{db: tx}).cleanupIndexesForColumn(tx, id); err != nil {
			return err
		}
		res := tx.Delete(&Column{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// --- Relations ---

type RelationPayload struct {
	Name         string `json:"name"`
	FromTableID  uint   `json:"from_table_id"`
	FromColumnID uint   `json:"from_column_id"`
	ToTableID    uint   `json:"to_table_id"`
	ToColumnID   uint   `json:"to_column_id"`
	OnDelete     string `json:"on_delete"`
	OnUpdate     string `json:"on_update"`
}

func defaultRelationName(tables []TableDTO, rel Relation) string {
	fromTable, fromCol := "", ""
	for _, t := range tables {
		if t.ID != rel.FromTableID {
			continue
		}
		fromTable = t.Name
		for _, c := range t.Columns {
			if c.ID == rel.FromColumnID {
				fromCol = c.Name
				break
			}
		}
		break
	}
	return DefaultFKName(fromTable, fromCol)
}

func (s *Store) finalizeRelationPayload(p *RelationPayload) error {
	onDelete, err := NormalizeFKAction(p.OnDelete)
	if err != nil {
		return err
	}
	onUpdate, err := NormalizeFKAction(p.OnUpdate)
	if err != nil {
		return err
	}
	p.OnDelete = onDelete
	p.OnUpdate = onUpdate

	name := strings.TrimSpace(p.Name)
	if name == "" {
		var fromTable Table
		var fromCol Column
		if err := s.db.First(&fromTable, p.FromTableID).Error; err != nil {
			return err
		}
		if err := s.db.First(&fromCol, p.FromColumnID).Error; err != nil {
			return err
		}
		name = DefaultFKName(fromTable.Name, fromCol.Name)
	}
	normalized, err := NormalizeFKName(name)
	if err != nil {
		return err
	}
	p.Name = normalized
	return nil
}

func (s *Store) CreateRelation(projectID uint, p RelationPayload) (*Relation, error) {
	if _, err := s.GetProject(projectID); err != nil {
		return nil, err
	}
	if err := s.finalizeRelationPayload(&p); err != nil {
		return nil, err
	}
	r := &Relation{
		ProjectID:    projectID,
		Name:         p.Name,
		FromTableID:  p.FromTableID,
		FromColumnID: p.FromColumnID,
		ToTableID:    p.ToTableID,
		ToColumnID:   p.ToColumnID,
		OnDelete:     p.OnDelete,
		OnUpdate:     p.OnUpdate,
	}
	if err := s.db.Create(r).Error; err != nil {
		return nil, err
	}
	// mark FK column
	s.db.Model(&Column{}).Where("id = ?", p.FromColumnID).Update("is_foreign_key", true)
	return r, nil
}

func (s *Store) UpdateRelation(id uint, p RelationPayload) (*Relation, error) {
	var r Relation
	if err := s.db.First(&r, id).Error; err != nil {
		return nil, ErrNotFound
	}
	p.FromTableID = r.FromTableID
	p.FromColumnID = r.FromColumnID
	p.ToTableID = r.ToTableID
	p.ToColumnID = r.ToColumnID
	if err := s.finalizeRelationPayload(&p); err != nil {
		return nil, err
	}
	r.Name = p.Name
	r.OnDelete = p.OnDelete
	r.OnUpdate = p.OnUpdate
	if err := s.db.Save(&r).Error; err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) DeleteRelation(id uint) error {
	var rel Relation
	if err := s.db.First(&rel, id).Error; err != nil {
		return ErrNotFound
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&Relation{}, id).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&Relation{}).Where("from_column_id = ?", rel.FromColumnID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Model(&Column{}).Where("id = ?", rel.FromColumnID).Update("is_foreign_key", false).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ImportSQL parses DDL and builds the visual schema (replace clears existing tables first)
func (s *Store) ImportSQL(projectID uint, sql string, replace bool) (*ImportResult, error) {
	if _, err := s.GetProject(projectID); err != nil {
		return nil, err
	}
	parsed, alterFKs, err := ParseSQL(sql)
	if err != nil {
		return nil, err
	}

	result := &ImportResult{}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		st := &Store{db: tx}
		var layouts map[string]tableLayoutSnapshot
		if replace {
			var err error
			layouts, err = st.snapshotTableLayouts(projectID)
			if err != nil {
				return err
			}
			var tables []Table
			if err := tx.Where("project_id = ?", projectID).Find(&tables).Error; err != nil {
				return err
			}
			for _, t := range tables {
				if err := st.DeleteTable(t.ID); err != nil {
					return err
				}
			}
		}

		tableMap := make(map[string]uint)  // name -> table id
		columnMap := make(map[string]uint) // table.col -> column id
		newTableIdx := 0

		for i, pt := range parsed {
			posX, posY, width := importTableLayout(pt.Name, i, newTableIdx, layouts)
			if replace && layouts != nil {
				if _, ok := layouts[strings.ToLower(pt.Name)]; !ok {
					newTableIdx++
				}
			}
			t := &Table{ProjectID: projectID, Name: pt.Name, PosX: posX, PosY: posY, Width: width}
			if err := tx.Create(t).Error; err != nil {
				return err
			}
			result.TablesCreated++
			tableMap[strings.ToLower(pt.Name)] = t.ID

			hasPK := false
			for _, pc := range pt.Columns {
				if pc.IsPrimaryKey {
					hasPK = true
				}
			}
			if !hasPK && len(pt.Columns) == 0 {
				proj, _ := st.GetProject(projectID)
				pt.Columns = append(pt.Columns, parsedColumn{
					Name: "id", DataType: defaultPKType(proj.Dialect), IsPrimaryKey: true, IsAutoIncrement: true,
				})
			}

			for si, pc := range pt.Columns {
				c := &Column{
					TableID:         t.ID,
					Name:            pc.Name,
					DataType:        pc.DataType,
					IsPrimaryKey:    pc.IsPrimaryKey,
					IsAutoIncrement: pc.IsAutoIncrement,
					IsNullable:      pc.IsNullable,
					IsUnique:        pc.IsUnique,
					DefaultValue:    pc.DefaultValue,
					SortOrder:       si,
				}
				if err := tx.Create(c).Error; err != nil {
					return err
				}
				result.ColumnsCreated++
				columnMap[strings.ToLower(pt.Name+"."+pc.Name)] = c.ID
			}

			for _, fk := range pt.FKs {
				fk.FromTable = pt.Name
				linkFK(st, projectID, tableMap, columnMap, fk, result)
			}

			for ii, pi := range pt.Indexes {
				if err := st.createIndexFromParsed(t.ID, pi, columnMap, pt.Name, ii); err != nil {
					return err
				}
				result.IndexesCreated++
			}
		}

		for _, fk := range alterFKs {
			linkFK(st, projectID, tableMap, columnMap, fk, result)
		}
		return nil
	})
	return result, err
}

func linkFK(s *Store, projectID uint, tableMap, columnMap map[string]uint, fk parsedForeignKey, result *ImportResult) {
	fromTable := strings.ToLower(fk.FromTable)
	toTable := strings.ToLower(fk.ToTable)
	ftID, ok1 := tableMap[fromTable]
	ttID, ok2 := tableMap[toTable]
	if !ok1 || !ok2 {
		return
	}
	fcKey := fromTable + "." + strings.ToLower(fk.FromColumn)
	tcKey := toTable + "." + strings.ToLower(fk.ToColumn)
	fcID, ok3 := columnMap[fcKey]
	tcID, ok4 := columnMap[tcKey]
	if !ok3 || !ok4 {
		return
	}
	_, err := s.CreateRelation(projectID, RelationPayload{
		Name:        fkName(fk),
		FromTableID: ftID, FromColumnID: fcID, ToTableID: ttID, ToColumnID: tcID,
		OnDelete: fk.OnDelete, OnUpdate: fk.OnUpdate,
	})
	if err == nil {
		result.RelationsCreated++
	}
}

func fkName(fk parsedForeignKey) string {
	if strings.TrimSpace(fk.Name) != "" {
		return fk.Name
	}
	return DefaultFKName(fk.FromTable, fk.FromColumn)
}

type tableLayoutSnapshot struct {
	PosX  float64
	PosY  float64
	Width float64
}

func (s *Store) snapshotTableLayouts(projectID uint) (map[string]tableLayoutSnapshot, error) {
	var tables []Table
	if err := s.db.Where("project_id = ?", projectID).Find(&tables).Error; err != nil {
		return nil, err
	}
	out := make(map[string]tableLayoutSnapshot, len(tables))
	for _, t := range tables {
		out[strings.ToLower(t.Name)] = tableLayoutSnapshot{
			PosX:  t.PosX,
			PosY:  t.PosY,
			Width: t.Width,
		}
	}
	return out, nil
}

func importTableLayout(name string, parsedIndex, newTableIndex int, layouts map[string]tableLayoutSnapshot) (posX, posY, width float64) {
	if layouts != nil {
		if saved, ok := layouts[strings.ToLower(name)]; ok {
			return saved.PosX, saved.PosY, saved.Width
		}
		posX, posY = defaultGridTablePosition(newTableIndex)
		return posX, posY, 0
	}
	posX, posY = defaultGridTablePosition(parsedIndex)
	return posX, posY, 0
}

func defaultGridTablePosition(index int) (posX, posY float64) {
	return float64(40 + (index%4)*280), float64(40 + (index/4)*220)
}
