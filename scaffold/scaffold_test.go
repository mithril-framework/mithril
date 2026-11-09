package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlanResource(t *testing.T) {
	dir := t.TempDir()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/shop\n\ngo 1.26\n"), 0o644))
	must(os.WriteFile(filepath.Join(dir, "resources.go"), []byte(`package main

import (
	"github.com/mithril-framework/mithril/resource"
)

func resources() []resource.Resource {
	return []resource.Resource{
		// mithril:resources
	}
}
`), 0o644))
	p, err := LoadProject(dir)
	must(err)
	fields, err := ParseFields([]string{"title:string:required", "body:text", "price:float", "published_at:time", "meta:json"})
	must(err)
	files, err := p.PlanResource(ResourceSpec{Model: "blogPost", Fields: fields, Owner: true}, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	must(err)
	if len(files) != 3 {
		t.Fatalf("files = %d", len(files))
	}
	model, mig, reg := files[0], files[1], files[2]
	if model.Path != "models/blog_post.go" || !strings.Contains(model.Content, "type BlogPost struct") ||
		!strings.Contains(model.Content, "`json:\"title\" validate:\"required,max=255\"`") ||
		!strings.Contains(model.Content, `mithril:"owner"`) || !strings.Contains(model.Content, `"encoding/json"`) {
		t.Fatalf("model:\n%s", model.Content)
	}
	if mig.Path != "migrations/20260102030405_create_blog_posts.sql" || !strings.Contains(mig.Content, "CREATE TABLE blog_posts") ||
		!strings.Contains(mig.Content, "owner_id UUID NOT NULL REFERENCES users(id)") || !strings.Contains(mig.Content, "-- +goose Down") {
		t.Fatalf("migration:\n%s", mig.Content)
	}
	if !strings.Contains(reg.Content, `resource.Define[models.BlogPost](resource.Options{Name: "blog_posts"}),`) ||
		!strings.Contains(reg.Content, `"example.com/shop/models"`) || !strings.Contains(reg.Content, "// mithril:resources") {
		t.Fatalf("registration:\n%s", reg.Content)
	}
	must(p.Apply(files))
	if _, err := p.PlanResource(ResourceSpec{Model: "BlogPost"}, time.Now()); err == nil {
		t.Fatal("existing model must not be overwritten")
	}
}

func TestParseFieldsErrors(t *testing.T) {
	for _, bad := range []string{"title", "title:nope", "id:string", "x:string:maybe", "9x:int"} {
		if _, err := ParseFields([]string{bad}); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if _, err := ParseFields([]string{"a:int", "a:int"}); err == nil {
		t.Error("duplicate accepted")
	}
}

func TestNaming(t *testing.T) {
	cases := map[string][2]string{"category": {"Category", "categories"}, "box": {"Box", "boxes"}, "userID": {"UserID", "user_ids"}, "day": {"Day", "days"}}
	for in, want := range cases {
		if got := Pascal(in); got != want[0] {
			t.Errorf("Pascal(%s) = %s", in, got)
		}
		if got := Plural(SnakeCase(in)); got != want[1] {
			t.Errorf("Plural(%s) = %s", in, got)
		}
	}
}
