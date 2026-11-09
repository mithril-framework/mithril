// Command acl manages roles, permissions, and superusers (Django-style ACL) for the reference app.
// In Mithril projects use `go run . acl ...` instead.
package main

import (
	"context"
	"log"
	"os"

	"github.com/mithril-framework/mithril/acl/aclcli"
	"github.com/mithril-framework/mithril/config"
	"github.com/mithril-framework/mithril/database/repositories"
	"github.com/mithril-framework/mithril/db"
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
	if err := aclcli.Run(ctx, repositories.NewACLRepository(pool), os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
