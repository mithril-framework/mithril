package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveJWTSecret(t *testing.T) {
	strong := "0123456789abcdef0123456789abcdef"
	cases := []struct {
		name, env, secret string
		wantErr           bool
	}{
		{"unset env, no secret", "", "", true},
		{"unset env, old default", "", "secret", true},
		{"production, placeholder", "production", "change-me-in-production", true},
		{"production, short", "production", "short-secret", true},
		{"staging, strong", "staging", strong, false},
		{"production, strong", "production", strong, false},
		{"development, empty -> random", "development", "", false},
		{"development, short allowed", "development", "dev", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveJWTSecret(tc.secret, tc.env)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if err == nil && (got == "" || got == "secret") {
				t.Fatalf("got weak secret %q", got)
			}
		})
	}
	a, _ := ResolveJWTSecret("", "development")
	b, _ := ResolveJWTSecret("", "development")
	if a == b || len(a) < MinJWTSecretLen {
		t.Fatalf("dev secrets must be random and >= %d chars", MinJWTSecretLen)
	}
}

func TestDSNEscapesCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db.local")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "p@ss/w:rd#?")
	t.Setenv("DB_NAME", "shop")
	t.Setenv("DB_SSLMODE", "require")
	want := "postgres://app:p%40ss%2Fw%3Ard%23%3F@db.local:5433/shop?sslmode=require"
	if got := DSNFromEnv(); got != want {
		t.Fatalf("DSN = %s, want %s", got, want)
	}
	t.Setenv("DB_HOST", "")
	if got := DSNFromEnv(); got != "" {
		t.Fatalf("no DB_HOST should mean no database, got %s", got)
	}
}

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	content := "# comment\nexport A_KEY=plain\nB_KEY=\"quoted # not comment\"\nC_KEY='single'\nD_KEY=value # inline\nE_KEY=fromfile\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"A_KEY", "B_KEY", "C_KEY", "D_KEY"} {
		os.Unsetenv(k)
		t.Cleanup(func() { os.Unsetenv(k) })
	}
	t.Setenv("E_KEY", "") // present but empty: the real environment wins
	if err := LoadDotEnv(p); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"A_KEY": "plain", "B_KEY": "quoted # not comment", "C_KEY": "single", "D_KEY": "value", "E_KEY": ""}
	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	if err := LoadDotEnv(filepath.Join(dir, "missing")); err != nil {
		t.Fatalf("missing file should be ignored: %v", err)
	}
}

func TestUnsetAppEnvIsProduction(t *testing.T) {
	if (Config{}).IsDevelopment() {
		t.Fatal("unset APP_ENV must not be development")
	}
}

func TestPerformanceDefaults(t *testing.T) {
	for _, k := range []string{"LOG_REQUESTS", "TRUST_PROXY", "PROXY_HEADER", "PPROF_ADDR", "DB_MAX_CONNS", "READ_TIMEOUT", "BODY_LIMIT", "PREFORK", "HIGH_THROUGHPUT"} {
		t.Setenv(k, "")
	}
	t.Setenv("APP_ENV", "development")
	c, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.LogRequests != "errors" || !c.TrustProxy || c.ProxyHeader != "X-Real-IP" || c.PprofAddr != "" ||
		c.ReadTimeout != 30*time.Second || c.IdleTimeout != 120*time.Second || c.BodyLimit != 4<<20 || c.Prefork || c.HighThroughput {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	t.Setenv("READ_TIMEOUT", "5")
	t.Setenv("WRITE_TIMEOUT", "1m")
	t.Setenv("DB_MAX_CONNS", "24")
	t.Setenv("LOG_REQUESTS", "ALL")
	c, err = FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if c.ReadTimeout != 5*time.Second || c.WriteTimeout != time.Minute || c.DBMaxConns != 24 || c.LogRequests != "all" {
		t.Fatalf("overrides not applied: %+v", c)
	}
	t.Setenv("LOG_REQUESTS", "verbose")
	if _, err := FromEnv(); err == nil {
		t.Fatal("invalid LOG_REQUESTS accepted")
	}
}
