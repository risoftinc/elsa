package dbdesign

import "time"

const (
	DialectMySQL    = "mysql"
	DialectPostgres = "postgres"
	DialectSQLite   = "sqlite"
)

// Project is a separate database design workspace
type Project struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Description string    `json:"description"`
	Dialect     string    `json:"dialect" gorm:"not null;default:mysql"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Table is an entity on the design canvas
type Table struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ProjectID uint      `json:"project_id" gorm:"not null;index"`
	Name      string    `json:"name" gorm:"not null"`
	PosX      float64   `json:"pos_x"`
	PosY      float64   `json:"pos_y"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Column belongs to a table
type Column struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	TableID      uint   `json:"table_id" gorm:"not null;index"`
	Name         string `json:"name" gorm:"not null"`
	DataType     string `json:"data_type" gorm:"not null"`
	IsPrimaryKey bool   `json:"is_primary_key"`
	IsForeignKey bool   `json:"is_foreign_key"`
	IsNullable   bool   `json:"is_nullable"`
	IsUnique     bool   `json:"is_unique"`
	DefaultValue string `json:"default_value"`
	SortOrder    int    `json:"sort_order"`
}

// Relation links FK column to PK column
type Relation struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	ProjectID    uint   `json:"project_id" gorm:"not null;index"`
	Name         string `json:"name" gorm:"not null;default:''"`
	FromTableID  uint   `json:"from_table_id" gorm:"not null"`
	FromColumnID uint   `json:"from_column_id" gorm:"not null"`
	ToTableID    uint   `json:"to_table_id" gorm:"not null"`
	ToColumnID   uint   `json:"to_column_id" gorm:"not null"`
	OnDelete     string `json:"on_delete" gorm:"not null;default:RESTRICT"`
	OnUpdate     string `json:"on_update" gorm:"not null;default:RESTRICT"`
}

// Index is a named index on a table (composite supported via IndexColumn)
type Index struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	TableID   uint   `json:"table_id" gorm:"not null;index"`
	Name      string `json:"name" gorm:"not null"`
	IsUnique  bool   `json:"is_unique"`
	SortOrder int    `json:"sort_order"`
}

// IndexColumn maps an index to a column in order
type IndexColumn struct {
	ID       uint `json:"id" gorm:"primaryKey"`
	IndexID  uint `json:"index_id" gorm:"not null;index"`
	ColumnID uint `json:"column_id" gorm:"not null"`
	Seq      int  `json:"seq"`
}

// IndexDTO is an index with ordered column IDs for API/UI
type IndexDTO struct {
	Index
	ColumnIDs []uint `json:"column_ids"`
}

// Schema is the full project design for API responses
type Schema struct {
	Project   Project    `json:"project"`
	Tables    []TableDTO `json:"tables"`
	Relations []Relation `json:"relations"`
}

// TableDTO includes columns and indexes for the canvas
type TableDTO struct {
	Table
	Columns []Column   `json:"columns"`
	Indexes []IndexDTO `json:"indexes"`
}

// ExportResult holds generated SQL
type ExportResult struct {
	SQL      string `json:"sql"`
	Dialect  string `json:"dialect"`
	Filename string `json:"filename"`
}
