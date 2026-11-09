package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/mithril-framework/mithril/openapi"
)

type greetIn struct {
	ID    uuid.UUID `path:"id"`
	Shout bool      `query:"shout"`
	Name  string    `json:"name" validate:"required,min=2"`
	Email string    `json:"email" validate:"email"`
}

type greetOut struct {
	Message string `json:"message"`
}

func TestTypedHandler(t *testing.T) {
	app := fiber.New()
	doc := openapi.New("t", "1")
	r := NewRouter(app, doc, "")
	Post(r, "/greet/:id", func(c fiber.Ctx, in *greetIn) (*greetOut, error) {
		msg := "hello " + in.Name + " " + in.ID.String()[:4]
		if in.Shout {
			msg = strings.ToUpper(msg)
		}
		return &greetOut{Message: msg}, nil
	}, Op{Summary: "Greet"})

	id := uuid.New()
	do := func(path, body string) (int, map[string]any) {
		req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		return resp.StatusCode, m
	}
	code, m := do("/greet/"+id.String()+"?shout=true", `{"name":"Ada","email":"ada@example.com"}`)
	if code != 201 || m["message"] != strings.ToUpper("hello Ada "+id.String()[:4]) {
		t.Fatalf("got %d %v", code, m)
	}
	code, m = do("/greet/"+id.String(), `{"name":"A","email":"nope"}`)
	if code != 422 || len(m["fields"].([]any)) != 2 {
		t.Fatalf("validation: %d %v", code, m)
	}
	if code, _ = do("/greet/not-a-uuid", `{"name":"Ada"}`); code != 400 {
		t.Fatalf("bad path param: %d", code)
	}
	if code, _ = do("/greet/"+id.String(), `{bad json`); code != 400 {
		t.Fatalf("bad json: %d", code)
	}

	ops := doc.Operations()
	if len(ops) != 1 || ops[0] != "POST /greet/{id}" {
		t.Fatalf("ops = %v", ops)
	}
	b, _ := json.Marshal(doc)
	s := string(b)
	for _, want := range []string{`"name":"id","in":"path"`, `"name":"shout","in":"query"`, `"format":"email"`, `"minLength":2`, `"201"`, `"422"`} {
		if !strings.Contains(s, want) {
			t.Errorf("spec missing %s", want)
		}
	}
	if strings.Contains(s, `"shout":{`) {
		t.Error("query field leaked into the body schema")
	}
}

func TestInternalErrorsAreHidden(t *testing.T) {
	app := fiber.New()
	r := NewRouter(app, nil, "")
	Get(r, "/boom", func(c fiber.Ctx, _ *Empty) (*greetOut, error) {
		return nil, io.ErrUnexpectedEOF
	}, Op{})
	resp, _ := app.Test(httptest.NewRequest("GET", "/boom", nil))
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 500 || strings.Contains(string(b), "EOF") {
		t.Fatalf("internal error leaked: %d %s", resp.StatusCode, b)
	}
}

func TestMiddlewareRunsBeforeHandler(t *testing.T) {
	app := fiber.New()
	r := NewRouter(app, nil, "")
	called := false
	deny := func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusForbidden) }
	Get(r, "/secret", func(c fiber.Ctx, _ *Empty) (*greetOut, error) {
		called = true
		return &greetOut{Message: "leak"}, nil
	}, Op{}, deny)
	resp, _ := app.Test(httptest.NewRequest("GET", "/secret", nil))
	if resp.StatusCode != 403 || called {
		t.Fatalf("middleware must run first: status %d, handler called=%v", resp.StatusCode, called)
	}
}
