package utils

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// SortField represents a single sort field
type SortField struct {
	Field     string `json:"field"`     // Field name to sort by
	Direction string `json:"direction"` // Sort direction (asc/desc)
}

// SortConfig holds configuration for sorting
type SortConfig struct {
	AllowedFields []string `json:"allowed_fields"` // Fields that can be sorted
	DefaultField  string   `json:"default_field"`  // Default sort field
	DefaultDir    string   `json:"default_dir"`    // Default sort direction
}

// DefaultSortConfig returns the default sort configuration
func DefaultSortConfig() *SortConfig {
	return &SortConfig{
		AllowedFields: []string{"id", "created_at", "updated_at"},
		DefaultField:  "id",
		DefaultDir:    "asc",
	}
}

// ParseSortRequest parses sort parameters from query string
func ParseSortRequest(sortStr string, config *SortConfig) ([]SortField, error) {
	if config == nil {
		config = DefaultSortConfig()
	}

	if sortStr == "" {
		return []SortField{{
			Field:     config.DefaultField,
			Direction: config.DefaultDir,
		}}, nil
	}

	var sortFields []SortField
	fields := strings.Split(sortStr, ",")

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		// Parse field and direction
		parts := strings.Split(field, ":")
		fieldName := parts[0]
		direction := "asc"

		if len(parts) > 1 {
			direction = strings.ToLower(parts[1])
		}

		// Validate direction
		if direction != "asc" && direction != "desc" {
			return nil, fmt.Errorf("invalid sort direction: %s", direction)
		}

		// Validate field is allowed
		if !isFieldAllowed(fieldName, config.AllowedFields) {
			return nil, fmt.Errorf("field '%s' is not allowed for sorting", fieldName)
		}

		sortFields = append(sortFields, SortField{
			Field:     fieldName,
			Direction: direction,
		})
	}

	if len(sortFields) == 0 {
		return []SortField{{
			Field:     config.DefaultField,
			Direction: config.DefaultDir,
		}}, nil
	}

	return sortFields, nil
}

// isFieldAllowed checks if a field is allowed for sorting
func isFieldAllowed(field string, allowedFields []string) bool {
	for _, allowed := range allowedFields {
		if field == allowed {
			return true
		}
	}
	return false
}

// SortSlice sorts a slice of structs based on sort fields
func SortSlice(slice interface{}, sortFields []SortField) error {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("expected pointer to slice")
	}

	sliceValue := rv.Elem()
	if sliceValue.Len() == 0 {
		return nil
	}

	// Create a sortable wrapper
	sortable := &sortableSlice{
		slice:       sliceValue,
		sortFields:  sortFields,
		elementType: sliceValue.Type().Elem(),
	}

	sort.Sort(sortable)
	return nil
}

// sortableSlice implements sort.Interface for sorting slices
type sortableSlice struct {
	slice       reflect.Value
	sortFields  []SortField
	elementType reflect.Type
}

// Len returns the length of the slice
func (s *sortableSlice) Len() int {
	return s.slice.Len()
}

// Swap swaps elements at indices i and j
func (s *sortableSlice) Swap(i, j int) {
	vi := s.slice.Index(i)
	vj := s.slice.Index(j)

	// Swap the values
	temp := reflect.New(s.elementType).Elem()
	temp.Set(vi)
	vi.Set(vj)
	vj.Set(temp)
}

// Less compares elements at indices i and j
func (s *sortableSlice) Less(i, j int) bool {
	vi := s.slice.Index(i)
	vj := s.slice.Index(j)

	// Compare based on sort fields
	for _, sortField := range s.sortFields {
		fieldI := s.getFieldValue(vi, sortField.Field)
		fieldJ := s.getFieldValue(vj, sortField.Field)

		comparison := s.compareValues(fieldI, fieldJ)
		if comparison != 0 {
			if sortField.Direction == "desc" {
				return comparison > 0
			}
			return comparison < 0
		}
	}

	return false
}

// getFieldValue gets the value of a field from a struct
func (s *sortableSlice) getFieldValue(v reflect.Value, fieldName string) reflect.Value {
	// Handle pointer to struct
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Find the field by name
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		// Try case-insensitive match
		fieldType := v.Type()
		for i := 0; i < fieldType.NumField(); i++ {
			fieldType := fieldType.Field(i)
			if strings.EqualFold(fieldType.Name, fieldName) {
				field = v.Field(i)
				break
			}
		}
	}

	return field
}

// compareValues compares two values and returns -1, 0, or 1
func (s *sortableSlice) compareValues(a, b reflect.Value) int {
	// Handle nil values
	if !a.IsValid() && !b.IsValid() {
		return 0
	}
	if !a.IsValid() {
		return -1
	}
	if !b.IsValid() {
		return 1
	}

	// Handle different types
	if a.Type() != b.Type() {
		// Convert to strings for comparison
		aStr := fmt.Sprintf("%v", a.Interface())
		bStr := fmt.Sprintf("%v", b.Interface())
		return strings.Compare(aStr, bStr)
	}

	// Compare based on type
	switch a.Kind() {
	case reflect.String:
		return strings.Compare(a.String(), b.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ai := a.Int()
		bi := b.Int()
		if ai < bi {
			return -1
		} else if ai > bi {
			return 1
		}
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		au := a.Uint()
		bu := b.Uint()
		if au < bu {
			return -1
		} else if au > bu {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		af := a.Float()
		bf := b.Float()
		if af < bf {
			return -1
		} else if af > bf {
			return 1
		}
		return 0
	case reflect.Bool:
		ab := a.Bool()
		bb := b.Bool()
		if !ab && bb {
			return -1
		} else if ab && !bb {
			return 1
		}
		return 0
	default:
		// For other types, convert to string and compare
		aStr := fmt.Sprintf("%v", a.Interface())
		bStr := fmt.Sprintf("%v", b.Interface())
		return strings.Compare(aStr, bStr)
	}
}

// SortStructs sorts a slice of structs by multiple fields
func SortStructs(slice interface{}, sortFields []SortField) error {
	return SortSlice(slice, sortFields)
}

// CreateSortConfig creates a sort configuration with allowed fields
func CreateSortConfig(allowedFields []string, defaultField, defaultDir string) *SortConfig {
	if defaultField == "" {
		defaultField = "id"
	}
	if defaultDir == "" {
		defaultDir = "asc"
	}

	return &SortConfig{
		AllowedFields: allowedFields,
		DefaultField:  defaultField,
		DefaultDir:    defaultDir,
	}
}

// ValidateSortFields validates that all sort fields are allowed
func ValidateSortFields(sortFields []SortField, config *SortConfig) error {
	if config == nil {
		return nil
	}

	for _, field := range sortFields {
		if !isFieldAllowed(field.Field, config.AllowedFields) {
			return fmt.Errorf("field '%s' is not allowed for sorting", field.Field)
		}
	}

	return nil
}

// GetSortString converts sort fields back to query string format
func GetSortString(sortFields []SortField) string {
	var parts []string
	for _, field := range sortFields {
		parts = append(parts, fmt.Sprintf("%s:%s", field.Field, field.Direction))
	}
	return strings.Join(parts, ",")
}
