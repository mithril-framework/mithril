package resource

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/api"
)

type post struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title" validate:"required,max=10"`
	Body      string    `json:"body" mithril:"text"`
	Secret    string    `json:"-"`
	AuthorID  uuid.UUID `json:"author_id" mithril:"owner"`
	Views     int64     `json:"views" db:"view_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func TestDecodeIgnoresServerManagedFields(t *testing.T) {
	r := Define[post](Options{Name: "posts"})
	attacker := uuid.New()
	body := `{"id":"` + attacker.String() + `","author_id":"` + attacker.String() + `","title":"hi","created_at":"2000-01-01T00:00:00Z","Secret":"x"}`
	v, present, err := r.decode([]byte(body), false)
	if err != nil {
		t.Fatal(err)
	}
	if v.ID != uuid.Nil || v.AuthorID != uuid.Nil || !v.CreatedAt.IsZero() || v.Secret != "" {
		t.Fatalf("read-only fields were assigned: %+v", v)
	}
	if !present["title"] || present["id"] || present["author_id"] {
		t.Fatalf("present = %v", present)
	}
}

func TestDecodeValidation(t *testing.T) {
	r := Define[post](Options{Name: "posts"})
	_, _, err := r.decode([]byte(`{"body":"x"}`), false)
	var ae *api.Error
	if !errors.As(err, &ae) || ae.Status != 422 || ae.Fields[0].Field != "title" {
		t.Fatalf("missing required title: %v", err)
	}
	if _, _, err := r.decode([]byte(`{"body":"x"}`), true); err != nil {
		t.Fatalf("partial update must only validate present fields: %v", err)
	}
	if _, _, err := r.decode([]byte(`{"title":"this is far too long"}`), true); err == nil {
		t.Fatal("max length not enforced")
	}
	if _, _, err := r.decode([]byte(`{"title":123}`), true); err == nil {
		t.Fatal("wrong type accepted")
	}
	if _, _, err := r.decode([]byte(`[1]`), true); err == nil {
		t.Fatal("non-object body accepted")
	}
}

func TestFieldsAndMeta(t *testing.T) {
	r := Define[post](Options{Name: "posts"})
	if r.pk.Column != "id" || r.owner.Column != "author_id" || r.orderBy.Column != "created_at" || r.updated == nil {
		t.Fatalf("pk=%v owner=%v order=%v", r.pk, r.owner, r.orderBy)
	}
	if got := r.byJSON["views"].Column; got != "view_count" {
		t.Fatalf("db tag ignored: %s", got)
	}
	m := r.Meta()
	kinds := map[string]string{}
	ro := map[string]bool{}
	for _, f := range m.Fields {
		kinds[f.Name], ro[f.Name] = f.Kind, f.ReadOnly
	}
	if kinds["body"] != "text" || kinds["views"] != "integer" || kinds["id"] != "uuid" || kinds["created_at"] != "datetime" {
		t.Fatalf("kinds = %v", kinds)
	}
	if !ro["id"] || !ro["author_id"] || !ro["updated_at"] || ro["title"] {
		t.Fatalf("readonly = %v", ro)
	}
	if _, ok := kinds["Secret"]; ok {
		t.Fatal(`json:"-" field exposed in meta`)
	}
	var codes []string
	for _, p := range r.Permissions() {
		codes = append(codes, p.Codename)
	}
	if strings.Join(codes, ",") != "posts.view,posts.add,posts.change,posts.change_any,posts.delete,posts.delete_any" {
		t.Fatalf("permissions = %v", codes)
	}
}

func TestInvalidDefinitions(t *testing.T) {
	type noPK struct{ Name string }
	if _, err := New[noPK](Options{Name: "x"}); err == nil {
		t.Fatal("struct without pk accepted")
	}
	type badOwner struct {
		ID    uuid.UUID
		Owner string `mithril:"owner"`
	}
	if _, err := New[badOwner](Options{Name: "x"}); err == nil {
		t.Fatal("non-uuid owner accepted")
	}
	if _, err := New[post](Options{Name: "a/b"}); err == nil {
		t.Fatal("invalid name accepted")
	}
	type plain struct {
		ID   int64
		Name string
	}
	r, err := New[plain](Options{Name: "plains"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Permissions()) != 4 {
		t.Fatalf("resource without owner must have view/add/change/delete, got %v", r.Permissions())
	}
	if r.byJSON["Name"] == nil || r.byJSON["Name"].Column != "name" {
		t.Fatal("default column should be snake_case")
	}
}

func TestSQLIdentifiersAreQuoted(t *testing.T) {
	r := Define[post](Options{Name: "posts", Table: `we"ird`})
	if r.table() != `"we""ird"` {
		t.Fatalf("table = %s", r.table())
	}
}
