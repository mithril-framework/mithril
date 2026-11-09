package resource

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mithril-framework/mithril/openapi"
)

// field is one persisted struct field.
type field struct {
	GoName   string
	Index    []int
	Column   string
	JSON     string // "" when not exposed over the API (json:"-")
	Type     reflect.Type
	PK       bool
	Owner    bool
	ReadOnly bool // server-managed (pk, owner, created_at, updated_at, mithril:"readonly")
	Text     bool // long text (admin textarea)
	Rules    map[string]string
}

func (f *field) writable() bool { return f.JSON != "" && !f.ReadOnly && !f.PK && !f.Owner }

func (f *field) quoted() string { return pgx.Identifier{f.Column}.Sanitize() }

// parseFields reflects T's persisted fields.
//
// Tags: db:"column" (default snake_case of the field name; "-" skips), json:"name",
// mithril:"pk,readonly,owner,text", validate:"...".
// The primary key is the field tagged mithril:"pk", else the field whose column is "id".
// created_at / updated_at columns are read-only and maintained by the database / framework.
func parseFields(t reflect.Type, ownerField string) ([]*field, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("resource type %s must be a struct", t)
	}
	var out []*field
	var walk func(t reflect.Type, index []int)
	walk = func(t reflect.Type, index []int) {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			idx := append(append([]int{}, index...), i)
			if sf.Anonymous && sf.Type.Kind() == reflect.Struct && sf.Tag.Get("db") == "" {
				walk(sf.Type, idx)
				continue
			}
			if !sf.IsExported() {
				continue
			}
			col := sf.Tag.Get("db")
			if col == "-" {
				continue
			}
			if col == "" {
				col = snakeCase(sf.Name)
			}
			jsonName, _ := openapi.FieldName(sf)
			f := &field{GoName: sf.Name, Index: idx, Column: col, JSON: jsonName, Type: sf.Type,
				Rules: openapi.ParseRules(sf.Tag.Get("validate"))}
			for _, opt := range strings.Split(sf.Tag.Get("mithril"), ",") {
				switch strings.TrimSpace(opt) {
				case "pk":
					f.PK = true
				case "readonly":
					f.ReadOnly = true
				case "owner":
					f.Owner = true
				case "text":
					f.Text = true
				}
			}
			if ownerField != "" && (sf.Name == ownerField || col == ownerField) {
				f.Owner = true
			}
			if col == "created_at" || col == "updated_at" {
				f.ReadOnly = true
			}
			out = append(out, f)
		}
	}
	walk(t, nil)

	var pk, owner *field
	for _, f := range out {
		if f.PK {
			if pk != nil {
				return nil, fmt.Errorf("%s: more than one mithril:\"pk\" field", t)
			}
			pk = f
		}
	}
	if pk == nil {
		for _, f := range out {
			if f.Column == "id" {
				f.PK, pk = true, f
			}
		}
	}
	if pk == nil {
		return nil, fmt.Errorf("%s: no primary key (tag a field mithril:\"pk\" or name its column id)", t)
	}
	if !isUUID(pk.Type) && !isInt(pk.Type) {
		return nil, fmt.Errorf("%s: primary key %s must be uuid.UUID or an integer", t, pk.GoName)
	}
	pk.ReadOnly = true
	for _, f := range out {
		if f.Owner {
			if owner != nil {
				return nil, fmt.Errorf("%s: more than one owner field", t)
			}
			if !isUUID(f.Type) {
				return nil, fmt.Errorf("%s: owner field %s must be uuid.UUID", t, f.GoName)
			}
			owner = f
		}
	}
	if ownerField != "" && owner == nil {
		return nil, fmt.Errorf("%s: owner field %q not found", t, ownerField)
	}
	return out, nil
}

var (
	uuidType = reflect.TypeOf(uuid.UUID{})
	timeType = reflect.TypeOf(time.Time{})
)

func isUUID(t reflect.Type) bool { return t == uuidType }

func isInt(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		return true
	}
	return false
}

// kind is the admin UI widget type for a field.
func (f *field) kind() string {
	t := f.Type
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t == uuidType:
		return "uuid"
	case t == timeType:
		return "datetime"
	}
	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		if f.Text {
			return "text"
		}
		if hasKey(f.Rules, "email") {
			return "email"
		}
		return "string"
	}
	return "json"
}

func hasKey(m map[string]string, k string) bool { _, ok := m[k]; return ok }

func snakeCase(s string) string {
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		n = n*10 + int(r-'0')
	}
	return n
}

func itoa(n int) string { return fmt.Sprint(n) }
