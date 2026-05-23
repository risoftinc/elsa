package envmanager

import "time"

// Environment is a named group (local, staging, production, etc.)
type Environment struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Variable is an env key shared across selected environments
type Variable struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Key         string    `json:"key" gorm:"uniqueIndex;not null"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Value stores the value of a variable for a specific environment
type Value struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	VariableID    uint   `json:"variable_id" gorm:"not null;uniqueIndex:idx_var_env"`
	EnvironmentID uint   `json:"environment_id" gorm:"not null;uniqueIndex:idx_var_env"`
	Value         string `json:"value"`
}

// Template stores a Go text/template body for export
type Template struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"uniqueIndex;not null"`
	Body      string    `json:"body" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// VariablePayload is used for create/update API requests
type VariablePayload struct {
	Key          string         `json:"key"`
	Description  string         `json:"description"`
	EnvironmentIDs []uint       `json:"environment_ids"`
	Values       []ValuePayload `json:"values"`
}

type ValuePayload struct {
	EnvironmentID uint   `json:"environment_id"`
	Value         string `json:"value"`
}

// VariableDetail includes values per environment for API responses
type VariableDetail struct {
	Variable
	EnvironmentIDs []uint        `json:"environment_ids"`
	Values         []ValueDetail `json:"values"`
}

type ValueDetail struct {
	EnvironmentID   uint   `json:"environment_id"`
	EnvironmentName string `json:"environment_name"`
	Value             string `json:"value"`
}

// RenderRequest for template / env export
type RenderRequest struct {
	EnvironmentID uint   `json:"environment_id"`
	Filename      string `json:"filename"`
}

// RenderResponse holds generated content
type RenderResponse struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
}
