package api

import (
	"fmt"
	"net/mail"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/openapi"
)

// FieldError is one failed validation rule.
type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// Validate checks struct fields against `validate` tags. Supported rules:
// required, min, max (length for strings/slices, value for numbers), email, url, uuid, oneof.
// Fields named in only (JSON names) are validated; nil only means all fields.
func Validate(v any, only map[string]bool) []FieldError {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	var errs []FieldError
	validateStruct(rv, only, &errs)
	return errs
}

func validateStruct(rv reflect.Value, only map[string]bool, errs *[]FieldError) {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			validateStruct(rv.Field(i), only, errs)
			continue
		}
		tag := f.Tag.Get("validate")
		if tag == "" || !f.IsExported() {
			continue
		}
		name := fieldLabel(f)
		if only != nil && !only[name] {
			continue
		}
		validateField(name, rv.Field(i), openapi.ParseRules(tag), errs)
	}
}

func fieldLabel(f reflect.StructField) string {
	for _, t := range []string{"path", "query"} {
		if v := f.Tag.Get(t); v != "" {
			return strings.Split(v, ",")[0]
		}
	}
	name, _ := openapi.FieldName(f)
	if name == "" {
		return f.Name
	}
	return name
}

func validateField(name string, v reflect.Value, rules map[string]string, errs *[]FieldError) {
	add := func(rule, msg string) { *errs = append(*errs, FieldError{Field: name, Rule: rule, Message: msg}) }
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			if _, ok := rules["required"]; ok {
				add("required", name+" is required")
			}
			return
		}
		v = v.Elem()
	}
	if _, ok := rules["required"]; ok && v.IsZero() {
		add("required", name+" is required")
		return
	}
	if v.IsZero() {
		return // optional and empty: other rules do not apply
	}
	for rule, arg := range rules {
		switch rule {
		case "min", "max":
			limit, err := strconv.ParseFloat(arg, 64)
			if err != nil {
				continue
			}
			var n float64
			unit := ""
			switch v.Kind() {
			case reflect.String:
				n, unit = float64(utf8.RuneCountInString(v.String())), " characters"
			case reflect.Slice, reflect.Map, reflect.Array:
				n, unit = float64(v.Len()), " items"
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n = float64(v.Int())
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				n = float64(v.Uint())
			case reflect.Float32, reflect.Float64:
				n = v.Float()
			default:
				continue
			}
			if rule == "min" && n < limit {
				add("min", fmt.Sprintf("%s must be at least %s%s", name, arg, unit))
			}
			if rule == "max" && n > limit {
				add("max", fmt.Sprintf("%s must be at most %s%s", name, arg, unit))
			}
		case "email":
			if a, err := mail.ParseAddress(v.String()); err != nil || a.Address != v.String() {
				add("email", name+" must be a valid email address")
			}
		case "url":
			if u, err := url.ParseRequestURI(v.String()); err != nil || u.Host == "" {
				add("url", name+" must be a valid URL")
			}
		case "uuid":
			if v.Kind() == reflect.String {
				if _, err := uuid.Parse(v.String()); err != nil {
					add("uuid", name+" must be a UUID")
				}
			}
		case "oneof":
			s := fmt.Sprint(v.Interface())
			ok := false
			for _, o := range strings.Fields(arg) {
				if o == s {
					ok = true
				}
			}
			if !ok {
				add("oneof", fmt.Sprintf("%s must be one of: %s", name, arg))
			}
		}
	}
}
