package utils

import (
	"reflect"
	"sort"
	"strings"
)

// SortField represents a field name and sort order (asc/desc).
type SortField struct {
	Field string
	Order string // "asc" or "desc"
}

// ParseSortParams parses a sort string like "name,-created_at" into SortField slice.
func ParseSortParams(sortStr string) []SortField {
	if sortStr == "" {
		return []SortField{}
	}
	var fields []SortField
	parts := strings.Split(sortStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		field := SortField{Field: part, Order: "asc"}
		if strings.HasPrefix(part, "-") {
			field.Field = part[1:]
			field.Order = "desc"
		}
		fields = append(fields, field)
	}
	return fields
}

// SortSlice sorts slice in place by the given fields (struct field names).
func SortSlice(slice interface{}, fields []SortField) {
	if len(fields) == 0 {
		return
	}
	sort.Slice(slice, func(i, j int) bool {
		for _, field := range fields {
			result := compareFields(slice, i, j, field.Field)
			if result != 0 {
				if field.Order == "desc" {
					return result > 0
				}
				return result < 0
			}
		}
		return false
	})
}

func compareFields(slice interface{}, i, j int, fieldName string) int {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		return 0
	}
	elem1 := v.Index(i)
	elem2 := v.Index(j)
	if elem1.Kind() == reflect.Ptr {
		elem1 = elem1.Elem()
	}
	if elem2.Kind() == reflect.Ptr {
		elem2 = elem2.Elem()
	}
	field1 := elem1.FieldByName(fieldName)
	field2 := elem2.FieldByName(fieldName)
	if !field1.IsValid() || !field2.IsValid() {
		return 0
	}
	switch field1.Kind() {
	case reflect.String:
		return strings.Compare(field1.String(), field2.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field1.Int() < field2.Int() {
			return -1
		} else if field1.Int() > field2.Int() {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		if field1.Float() < field2.Float() {
			return -1
		} else if field1.Float() > field2.Float() {
			return 1
		}
		return 0
	case reflect.Bool:
		if field1.Bool() == field2.Bool() {
			return 0
		} else if field1.Bool() {
			return 1
		}
		return -1
	}
	return 0
}
