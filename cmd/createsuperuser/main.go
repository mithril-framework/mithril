// Command createsuperuser creates a superuser for the reference app.
// In Mithril projects use `go run . createsuperuser` instead.
package main

import (
	"context"
	"log"
	"os"

	"github.com/mithril-framework/mithril/config"
	"github.com/mithril-framework/mithril/db"
	"github.com/mithril-framework/mithril/users/createsuperuser"
)

func main() {
	_ = config.LoadDotEnv(".env")
	dsn := config.DSNFromEnv()
	if dsn == "" {
		log.Fatal("Set DATABASE_URL or DB_HOST (and DB_USER, DB_PASSWORD, DB_NAME) in the environment or .env")
	}
	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()
	if err := createsuperuser.Run(ctx, pool, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
