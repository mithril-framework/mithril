// Command acl manages roles, permissions, and superusers (Django-style ACL).
// Database: set DATABASE_URL or DB_* (same as migrate targets). Optional: load .env yourself or use make targets.
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"mithril-rev/database/repositories"
	"mithril-rev/internal/db"
)

func main() {
	loadEnvFile(".env")
	ctx := context.Background()
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
		log.Fatal("Set DATABASE_URL or DB_HOST (and DB_USER, DB_PASSWORD, DB_NAME) in the environment or .env")
	}
	dsn := db.DSNFromEnv()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	aclRepo := repositories.NewACLRepository(pool)

	args := os.Args[1:]
	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	switch args[0] {
	case "superuser":
		if len(args) < 3 {
			log.Fatal("usage: acl superuser set|unset EMAIL")
		}
		email := args[2]
		switch args[1] {
		case "set":
			if err := aclRepo.SetSuperuserByEmail(ctx, email, true); err != nil {
				log.Fatal(err)
			}
			fmt.Println("superuser enabled for", email)
		case "unset":
			if err := aclRepo.SetSuperuserByEmail(ctx, email, false); err != nil {
				log.Fatal(err)
			}
			fmt.Println("superuser disabled for", email)
		default:
			log.Fatal("usage: acl superuser set|unset EMAIL")
		}
	case "role":
		if len(args) < 2 {
			log.Fatal("usage: acl role create|delete NAME [description]")
		}
		switch args[1] {
		case "create":
			if len(args) < 3 {
				log.Fatal("usage: acl role create NAME [description]")
			}
			name := args[2]
			desc := ""
			if len(args) > 3 {
				desc = strings.Join(args[3:], " ")
			}
			r, err := aclRepo.CreateRole(ctx, name, desc)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("role %s id=%s\n", r.Name, r.ID)
		case "delete":
			if len(args) < 3 {
				log.Fatal("usage: acl role delete NAME")
			}
			if err := aclRepo.DeleteRoleByName(ctx, args[2]); err != nil {
				log.Fatal(err)
			}
			fmt.Println("deleted role", args[2])
		default:
			log.Fatal("usage: acl role create|delete ...")
		}
	case "permission":
		if len(args) < 2 {
			log.Fatal("usage: acl permission create|delete CODENAME [description]")
		}
		switch args[1] {
		case "create":
			if len(args) < 3 {
				log.Fatal("usage: acl permission create CODENAME [description]")
			}
			codename := args[2]
			desc := ""
			if len(args) > 3 {
				desc = strings.Join(args[3:], " ")
			}
			p, err := aclRepo.CreatePermission(ctx, codename, desc)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("permission %s id=%s\n", p.Codename, p.ID)
		case "delete":
			if len(args) < 3 {
				log.Fatal("usage: acl permission delete CODENAME")
			}
			if err := aclRepo.DeletePermissionByCodename(ctx, args[2]); err != nil {
				log.Fatal(err)
			}
			fmt.Println("deleted permission", args[2])
		default:
			log.Fatal("usage: acl permission create|delete ...")
		}
	case "assign":
		if len(args) < 2 {
			log.Fatal("usage: acl assign role USER_EMAIL ROLE_NAME | assign permission role ROLE_NAME CODENAME | assign permission user USER_EMAIL CODENAME")
		}
		switch args[1] {
		case "role":
			if len(args) < 4 {
				log.Fatal("usage: acl assign role USER_EMAIL ROLE_NAME")
			}
			uid, err := aclRepo.GetUserIDByEmail(ctx, args[2])
			if err != nil {
				log.Fatal(err)
			}
			rid, err := aclRepo.GetRoleIDByName(ctx, args[3])
			if err != nil {
				log.Fatal(err)
			}
			if err := aclRepo.AssignRole(ctx, uid, rid); err != nil {
				log.Fatal(err)
			}
			fmt.Println("assigned role", args[3], "to", args[2])
		case "permission":
			if len(args) < 3 {
				log.Fatal("usage: acl assign permission role|user ...")
			}
			switch args[2] {
			case "role":
				if len(args) < 5 {
					log.Fatal("usage: acl assign permission role ROLE_NAME CODENAME")
				}
				rid, err := aclRepo.GetRoleIDByName(ctx, args[3])
				if err != nil {
					log.Fatal(err)
				}
				pid, err := aclRepo.GetPermissionIDByCodename(ctx, args[4])
				if err != nil {
					log.Fatal(err)
				}
				if err := aclRepo.AssignPermissionToRole(ctx, rid, pid); err != nil {
					log.Fatal(err)
				}
				fmt.Println("assigned permission", args[4], "to role", args[3])
			case "user":
				if len(args) < 5 {
					log.Fatal("usage: acl assign permission user USER_EMAIL CODENAME")
				}
				uid, err := aclRepo.GetUserIDByEmail(ctx, args[3])
				if err != nil {
					log.Fatal(err)
				}
				pid, err := aclRepo.GetPermissionIDByCodename(ctx, args[4])
				if err != nil {
					log.Fatal(err)
				}
				if err := aclRepo.AssignPermissionToUser(ctx, uid, pid); err != nil {
					log.Fatal(err)
				}
				fmt.Println("assigned permission", args[4], "to user", args[3])
			default:
				log.Fatal("usage: acl assign permission role|user ...")
			}
		default:
			log.Fatal("usage: acl assign role|permission ...")
		}
	case "revoke":
		if len(args) < 2 {
			log.Fatal("usage: acl revoke role USER_EMAIL ROLE_NAME | revoke permission role ROLE_NAME CODENAME | revoke permission user USER_EMAIL CODENAME")
		}
		switch args[1] {
		case "role":
			if len(args) < 4 {
				log.Fatal("usage: acl revoke role USER_EMAIL ROLE_NAME")
			}
			uid, err := aclRepo.GetUserIDByEmail(ctx, args[2])
			if err != nil {
				log.Fatal(err)
			}
			rid, err := aclRepo.GetRoleIDByName(ctx, args[3])
			if err != nil {
				log.Fatal(err)
			}
			if err := aclRepo.RevokeRole(ctx, uid, rid); err != nil {
				log.Fatal(err)
			}
			fmt.Println("revoked role", args[3], "from", args[2])
		case "permission":
			if len(args) < 3 {
				log.Fatal("usage: acl revoke permission role|user ...")
			}
			switch args[2] {
			case "role":
				if len(args) < 5 {
					log.Fatal("usage: acl revoke permission role ROLE_NAME CODENAME")
				}
				rid, err := aclRepo.GetRoleIDByName(ctx, args[3])
				if err != nil {
					log.Fatal(err)
				}
				pid, err := aclRepo.GetPermissionIDByCodename(ctx, args[4])
				if err != nil {
					log.Fatal(err)
				}
				if err := aclRepo.RevokePermissionFromRole(ctx, rid, pid); err != nil {
					log.Fatal(err)
				}
				fmt.Println("revoked permission", args[4], "from role", args[3])
			case "user":
				if len(args) < 5 {
					log.Fatal("usage: acl revoke permission user USER_EMAIL CODENAME")
				}
				uid, err := aclRepo.GetUserIDByEmail(ctx, args[3])
				if err != nil {
					log.Fatal(err)
				}
				pid, err := aclRepo.GetPermissionIDByCodename(ctx, args[4])
				if err != nil {
					log.Fatal(err)
				}
				if err := aclRepo.RevokePermissionFromUser(ctx, uid, pid); err != nil {
					log.Fatal(err)
				}
				fmt.Println("revoked permission", args[4], "from user", args[3])
			default:
				log.Fatal("usage: acl revoke permission role|user ...")
			}
		default:
			log.Fatal("usage: acl revoke role|permission ...")
		}
	case "list":
		if len(args) < 2 {
			log.Fatal("usage: acl list permissions|roles")
		}
		switch args[1] {
		case "permissions":
			list, err := aclRepo.ListPermissions(ctx)
			if err != nil {
				log.Fatal(err)
			}
			for _, p := range list {
				fmt.Println(p.Codename, "\t", p.Description)
			}
		case "roles":
			list, err := aclRepo.ListRoles(ctx)
			if err != nil {
				log.Fatal(err)
			}
			for _, r := range list {
				fmt.Println(r.Name, "\t", r.Description)
			}
		default:
			log.Fatal("usage: acl list permissions|roles")
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  acl superuser set|unset EMAIL
  acl role create NAME [description]
  acl role delete NAME
  acl permission create CODENAME [description]
  acl permission delete CODENAME
  acl assign role USER_EMAIL ROLE_NAME
  acl assign permission role ROLE_NAME CODENAME
  acl assign permission user USER_EMAIL CODENAME
  acl revoke role USER_EMAIL ROLE_NAME
  acl revoke permission role ROLE_NAME CODENAME
  acl revoke permission user USER_EMAIL CODENAME
  acl list permissions
  acl list roles
`)
}

func loadEnvFile(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, `"`)
		os.Setenv(key, value)
	}
}
