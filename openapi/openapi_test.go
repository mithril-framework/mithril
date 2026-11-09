package openapi

import (
	"io"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestJSONHandlerCachesAndInvalidates(t *testing.T) {
	d := New("t", "1")
	d.AddOperation("GET", "/a", Operation{Summary: "A"})
	app := fiber.New()
	app.Get("/openapi.json", d.JSONHandler())
	get := func() string {
		resp, err := app.Test(httptest.NewRequest("GET", "/openapi.json", nil))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		return string(b)
	}
	first := get()
	if !strings.Contains(first, `"/a"`) || get() != first {
		t.Fatal("document not served or not stable")
	}
	d.AddOperation("POST", "/b/:id", Operation{Summary: "B"}) // registered after the first request
	if !strings.Contains(get(), `"/b/{id}"`) {
		t.Fatal("cache not invalidated after AddOperation")
	}
}

func TestConcurrentRenderAndMutate(t *testing.T) {
	d := New("t", "1")
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = d.render() }()
		go func() { defer wg.Done(); d.AddOperation("GET", "/x", Operation{}) }()
	}
	wg.Wait()
	b, err := d.render()
	if err != nil || !strings.Contains(string(b), `"/x"`) {
		t.Fatalf("render after concurrent mutation: %v", err)
	}
}
