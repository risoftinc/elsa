package dbdesign

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type IndexPayload struct {
	Name      string `json:"name"`
	IsUnique  bool   `json:"is_unique"`
	ColumnIDs []uint `json:"column_ids"`
	SortOrder int    `json:"sort_order"`
}

func (s *Store) loadIndexesForTable(tableID uint) ([]IndexDTO, error) {
	var indexes []Index
	if err := s.db.Where("table_id = ?", tableID).Order("sort_order asc, id asc").Find(&indexes).Error; err != nil {
		return nil, err
	}
	dtos := make([]IndexDTO, 0, len(indexes))
	for _, idx := range indexes {
		var cols []IndexColumn
		if err := s.db.Where("index_id = ?", idx.ID).Order("seq asc, id asc").Find(&cols).Error; err != nil {
			return nil, err
		}
		ids := make([]uint, 0, len(cols))
		for _, c := range cols {
			ids = append(ids, c.ColumnID)
		}
		dtos = append(dtos, IndexDTO{Index: idx, ColumnIDs: ids})
	}
	return dtos, nil
}

func (s *Store) deleteIndexesForTable(tx *gorm.DB, tableID uint) error {
	var indexes []Index
	if err := tx.Where("table_id = ?", tableID).Find(&indexes).Error; err != nil {
		return err
	}
	for _, idx := range indexes {
		if err := tx.Where("index_id = ?", idx.ID).Delete(&IndexColumn{}).Error; err != nil {
			return err
		}
	}
	return tx.Where("table_id = ?", tableID).Delete(&Index{}).Error
}

func (s *Store) validateIndexPayload(tableID uint, p IndexPayload) error {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		return fmt.Errorf("index name is required")
	}
	if len(p.ColumnIDs) == 0 {
		return fmt.Errorf("index must include at least one column")
	}
	seen := make(map[uint]struct{}, len(p.ColumnIDs))
	for _, colID := range p.ColumnIDs {
		if colID == 0 {
			return fmt.Errorf("invalid column in index")
		}
		if _, ok := seen[colID]; ok {
			return fmt.Errorf("duplicate column in index")
		}
		seen[colID] = struct{}{}
		var c Column
		if err := s.db.First(&c, colID).Error; err != nil {
			return fmt.Errorf("column not found")
		}
		if c.TableID != tableID {
			return fmt.Errorf("column %q does not belong to this table", c.Name)
		}
	}
	return nil
}

func (s *Store) replaceIndexColumns(tx *gorm.DB, indexID uint, columnIDs []uint) error {
	if err := tx.Where("index_id = ?", indexID).Delete(&IndexColumn{}).Error; err != nil {
		return err
	}
	for seq, colID := range columnIDs {
		ic := &IndexColumn{IndexID: indexID, ColumnID: colID, Seq: seq}
		if err := tx.Create(ic).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateIndex(tableID uint, p IndexPayload) (*IndexDTO, error) {
	var t Table
	if err := s.db.First(&t, tableID).Error; err != nil {
		return nil, ErrNotFound
	}
	if err := s.validateIndexPayload(tableID, p); err != nil {
		return nil, err
	}
	idx := &Index{
		TableID:   tableID,
		Name:      strings.TrimSpace(p.Name),
		IsUnique:  p.IsUnique,
		SortOrder: p.SortOrder,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(idx).Error; err != nil {
			return err
		}
		return s.replaceIndexColumns(tx, idx.ID, p.ColumnIDs)
	})
	if err != nil {
		return nil, err
	}
	return &IndexDTO{Index: *idx, ColumnIDs: append([]uint(nil), p.ColumnIDs...)}, nil
}

func (s *Store) UpdateIndex(id uint, p IndexPayload) (*IndexDTO, error) {
	var idx Index
	if err := s.db.First(&idx, id).Error; err != nil {
		return nil, ErrNotFound
	}
	if err := s.validateIndexPayload(idx.TableID, p); err != nil {
		return nil, err
	}
	idx.Name = strings.TrimSpace(p.Name)
	idx.IsUnique = p.IsUnique
	idx.SortOrder = p.SortOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&idx).Error; err != nil {
			return err
		}
		return s.replaceIndexColumns(tx, idx.ID, p.ColumnIDs)
	})
	if err != nil {
		return nil, err
	}
	return &IndexDTO{Index: idx, ColumnIDs: append([]uint(nil), p.ColumnIDs...)}, nil
}

func (s *Store) DeleteIndex(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("index_id = ?", id).Delete(&IndexColumn{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Index{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

func (s *Store) cleanupIndexesForColumn(tx *gorm.DB, columnID uint) error {
	var links []IndexColumn
	if err := tx.Where("column_id = ?", columnID).Find(&links).Error; err != nil {
		return err
	}
	indexIDs := make(map[uint]struct{})
	for _, l := range links {
		indexIDs[l.IndexID] = struct{}{}
		if err := tx.Delete(&IndexColumn{}, l.ID).Error; err != nil {
			return err
		}
	}
	for indexID := range indexIDs {
		var count int64
		if err := tx.Model(&IndexColumn{}).Where("index_id = ?", indexID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Delete(&Index{}, indexID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) createIndexFromParsed(tableID uint, pi parsedIndex, columnMap map[string]uint, tableName string, sortOrder int) error {
	if len(pi.Columns) == 0 {
		return nil
	}
	colIDs := make([]uint, 0, len(pi.Columns))
	for _, colName := range pi.Columns {
		key := strings.ToLower(tableName + "." + strings.ToLower(colName))
		id, ok := columnMap[key]
		if !ok {
			return fmt.Errorf("index column %q not found", colName)
		}
		colIDs = append(colIDs, id)
	}
	name := strings.TrimSpace(pi.Name)
	if name == "" {
		name = fmt.Sprintf("idx_%s_%d", tableName, sortOrder)
	}
	p := IndexPayload{
		Name:      name,
		IsUnique:  pi.IsUnique,
		ColumnIDs: colIDs,
		SortOrder: sortOrder,
	}
	if err := s.validateIndexPayload(tableID, p); err != nil {
		return err
	}
	idx := &Index{
		TableID:   tableID,
		Name:      name,
		IsUnique:  pi.IsUnique,
		SortOrder: sortOrder,
	}
	if err := s.db.Create(idx).Error; err != nil {
		return err
	}
	return s.replaceIndexColumns(s.db, idx.ID, colIDs)
}
