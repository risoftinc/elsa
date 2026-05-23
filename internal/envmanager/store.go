package envmanager

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("not found")

// Store wraps database operations for the env manager
type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// --- Environments ---

func (s *Store) ListEnvironments() ([]Environment, error) {
	var envs []Environment
	err := s.db.Order("sort_order asc, name asc").Find(&envs).Error
	return envs, err
}

func (s *Store) CreateEnvironment(name string, sortOrder int) (*Environment, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	env := &Environment{Name: name, SortOrder: sortOrder}
	if err := s.db.Create(env).Error; err != nil {
		return nil, err
	}
	return env, nil
}

func (s *Store) UpdateEnvironment(id uint, name string, sortOrder int) (*Environment, error) {
	var env Environment
	if err := s.db.First(&env, id).Error; err != nil {
		return nil, ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	env.Name = name
	env.SortOrder = sortOrder
	if err := s.db.Save(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

func (s *Store) DeleteEnvironment(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("environment_id = ?", id).Delete(&Value{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Environment{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// --- Variables ---

func (s *Store) ListVariables() ([]VariableDetail, error) {
	var vars []Variable
	if err := s.db.Order("key asc").Find(&vars).Error; err != nil {
		return nil, err
	}

	result := make([]VariableDetail, 0, len(vars))
	for _, v := range vars {
		detail, err := s.getVariableDetail(v)
		if err != nil {
			return nil, err
		}
		result = append(result, *detail)
	}
	return result, nil
}

func (s *Store) GetVariable(id uint) (*VariableDetail, error) {
	var v Variable
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, ErrNotFound
	}
	return s.getVariableDetail(v)
}

func (s *Store) getVariableDetail(v Variable) (*VariableDetail, error) {
	var values []Value
	if err := s.db.Where("variable_id = ?", v.ID).Find(&values).Error; err != nil {
		return nil, err
	}

	envIDs := make([]uint, 0, len(values))
	valueDetails := make([]ValueDetail, 0, len(values))

	for _, val := range values {
		var env Environment
		if err := s.db.First(&env, val.EnvironmentID).Error; err != nil {
			continue
		}
		envIDs = append(envIDs, val.EnvironmentID)
		valueDetails = append(valueDetails, ValueDetail{
			EnvironmentID:   val.EnvironmentID,
			EnvironmentName: env.Name,
			Value:           val.Value,
		})
	}

	return &VariableDetail{
		Variable:       v,
		EnvironmentIDs: envIDs,
		Values:         valueDetails,
	}, nil
}

func (s *Store) CreateVariable(payload VariablePayload) (*VariableDetail, error) {
	key := strings.TrimSpace(payload.Key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	var detail *VariableDetail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		v := &Variable{Key: key, Description: strings.TrimSpace(payload.Description)}
		if err := tx.Create(v).Error; err != nil {
			return err
		}
		if err := s.syncValues(tx, v.ID, payload); err != nil {
			return err
		}
		store := &Store{db: tx}
		var err error
		detail, err = store.getVariableDetail(*v)
		return err
	})
	return detail, err
}

func (s *Store) UpdateVariable(id uint, payload VariablePayload) (*VariableDetail, error) {
	key := strings.TrimSpace(payload.Key)
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	var detail *VariableDetail
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var v Variable
		if err := tx.First(&v, id).Error; err != nil {
			return ErrNotFound
		}
		v.Key = key
		v.Description = strings.TrimSpace(payload.Description)
		if err := tx.Save(&v).Error; err != nil {
			return err
		}
		if err := tx.Where("variable_id = ?", id).Delete(&Value{}).Error; err != nil {
			return err
		}
		if err := s.syncValues(tx, id, payload); err != nil {
			return err
		}
		store := &Store{db: tx}
		var err error
		detail, err = store.getVariableDetail(v)
		return err
	})
	return detail, err
}

func (s *Store) syncValues(tx *gorm.DB, variableID uint, payload VariablePayload) error {
	seen := make(map[uint]bool)
	for _, vp := range payload.Values {
		if !containsUint(payload.EnvironmentIDs, vp.EnvironmentID) {
			continue
		}
		if seen[vp.EnvironmentID] {
			continue
		}
		seen[vp.EnvironmentID] = true
		val := &Value{
			VariableID:    variableID,
			EnvironmentID: vp.EnvironmentID,
			Value:         vp.Value,
		}
		if err := tx.Create(val).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteVariable(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("variable_id = ?", id).Delete(&Value{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Variable{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// EnvMap returns all key-value pairs for an environment (for templates / export)
func (s *Store) EnvMap(environmentID uint) (map[string]string, error) {
	var env Environment
	if err := s.db.First(&env, environmentID).Error; err != nil {
		return nil, ErrNotFound
	}

	var values []Value
	if err := s.db.Where("environment_id = ?", environmentID).Find(&values).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, val := range values {
		var v Variable
		if err := s.db.First(&v, val.VariableID).Error; err != nil {
			continue
		}
		result[v.Key] = val.Value
	}
	return result, nil
}

// ExportDotEnv generates KEY=value lines for one environment
func (s *Store) ExportDotEnv(environmentID uint) (string, error) {
	m, err := s.EnvMap(environmentID)
	if err != nil {
		return "", err
	}

	vars, err := s.ListVariables()
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, vd := range vars {
		val, ok := m[vd.Key]
		if !ok {
			continue
		}
		b.WriteString(formatDotEnvLine(vd.Key, val))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

func formatDotEnvLine(key, value string) string {
	if needsQuotes(value) {
		escaped := strings.ReplaceAll(value, `"`, `\"`)
		return fmt.Sprintf("%s=\"%s\"", key, escaped)
	}
	return fmt.Sprintf("%s=%s", key, value)
}

func needsQuotes(value string) bool {
	return strings.ContainsAny(value, " #\t\n\"'\\")
}

// --- Templates ---

func (s *Store) templateNameExists(name string, excludeID uint) (bool, error) {
	name = strings.TrimSpace(name)
	var count int64
	q := s.db.Model(&Template{}).Where("LOWER(name) = LOWER(?)", name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) ListTemplates() ([]Template, error) {
	var templates []Template
	err := s.db.Order("name asc").Find(&templates).Error
	return templates, err
}

func (s *Store) GetTemplate(id uint) (*Template, error) {
	var t Template
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &t, nil
}

func (s *Store) CreateTemplate(name, body string) (*Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("template body is required")
	}
	taken, err := s.templateNameExists(name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("template name %q already exists", name)
	}
	t := &Template{Name: name, Body: body}
	if err := s.db.Create(t).Error; err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) UpdateTemplate(id uint, name, body string) (*Template, error) {
	var t Template
	if err := s.db.First(&t, id).Error; err != nil {
		return nil, ErrNotFound
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("template body is required")
	}
	taken, err := s.templateNameExists(name, id)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("template name %q already exists", name)
	}
	t.Name = name
	t.Body = body
	if err := s.db.Save(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) DeleteTemplate(id uint) error {
	res := s.db.Delete(&Template{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func containsUint(slice []uint, id uint) bool {
	for _, v := range slice {
		if v == id {
			return true
		}
	}
	return false
}
