// Command dbms runs the embedded PostgreSQL web UI (Adminer-style).
// Enable with: make dbms-enable or ENABLE_DBMS=true in the environment.
package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"mithril-rev/internal/dbms"
)

func main() {
	loadEnvFile(".env")
	if !dbms.Enabled() {
		log.Fatal("DBMS disabled. Run: make dbms-enable or set ENABLE_DBMS=true in .env")
	}
	if err := dbms.ListenAndServe(strings.TrimSpace(os.Getenv("DBMS_ADDR"))); err != nil {
		log.Fatal(err)
	}
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
