// Package config loads Mithril's typed configuration from the environment (and an optional .env file).
//
// Every setting the framework reads lives in Config, so the documentation can be generated from one
// place and nothing reads os.Getenv ad hoc.
package config

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the full framework configuration. Field comments document the environment variable.
type Config struct {
	AppName string // APP_NAME (default "mithril")
	Env     string // APP_ENV: development | production | staging | test. Unset behaves as production.
	Port    string // PORT (default 4000)

	JWTSecret string // JWT_SECRET: >= 32 bytes outside development

	DatabaseURL string // DATABASE_URL, or built from DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/DB_SSLMODE

	CORSOrigins []string // CORS_ORIGINS (comma-separated; default http://localhost:3000,http://localhost:4000)

	EnableRegister    bool // ENABLE_REGISTER: allow POST /auth/register
	EnableHelmet      bool // ENABLE_HELMET: security headers (default true)
	EnableCompression bool // ENABLE_COMPRESSION: gzip/brotli responses
	EnableAdmin       bool // ENABLE_ADMIN_PANEL or a .admin-panel-enabled file
	EnableMonitor     bool // ENABLE_MONITOR: /monitor dashboard (superuser only)
	EnableDocs        bool // ENABLE_DOCS: Swagger UI at /docs (default: on outside production)

	Timezone string // APP_TIMEZONE or TZ (default UTC)

	// Performance and operations. Defaults are tuned for production; see docs "Performance".
	LogRequests  string        // LOG_REQUESTS: errors (default: status >= 400 only) | all | none
	TrustProxy   bool          // TRUST_PROXY (default true): read the client IP from ProxyHeader, only when the peer is loopback/private
	ProxyHeader  string        // PROXY_HEADER (default X-Real-IP)
	PprofAddr    string        // PPROF_ADDR: private pprof listener, e.g. 127.0.0.1:6060 (off when empty)
	DBMaxConns   int32         // DB_MAX_CONNS (default: 4 per CPU, 10–50)
	DBMinConns   int32         // DB_MIN_CONNS (default 2)
	ReadTimeout  time.Duration // READ_TIMEOUT (default 30s)
	WriteTimeout time.Duration // WRITE_TIMEOUT (default 30s)
	IdleTimeout  time.Duration // IDLE_TIMEOUT keep-alive (default 120s)
	BodyLimit    int           // BODY_LIMIT bytes (default 4 MiB)
	Prefork      bool          // PREFORK: one process per CPU (Linux, stateless apps only; default false)
	// HighThroughput (HIGH_THROUGHPUT) drops the Date and default Content-Type headers for
	// stateless hot services doing very high request rates. Off by default.
	HighThroughput bool
}

// IsDevelopment reports whether APP_ENV explicitly names a non-production environment.
// An unset APP_ENV is not development: the secure behaviour is the default.
func (c Config) IsDevelopment() bool { return isDevelopmentEnv(c.Env) }

// IsProduction reports whether the app runs with production semantics (including unset APP_ENV).
func (c Config) IsProduction() bool { return !c.IsDevelopment() }

// Addr returns the listen address.
func (c Config) Addr() string { return ":" + c.Port }

// HasDatabase reports whether a database is configured.
func (c Config) HasDatabase() bool { return c.DatabaseURL != "" }

// FromEnv builds a Config from the environment. Call LoadDotEnv first to merge a .env file.
// It validates security-critical settings and returns an error rather than starting insecurely.
func FromEnv() (Config, error) {
	env := strings.TrimSpace(os.Getenv("APP_ENV"))
	c := Config{
		AppName:           envOr("APP_NAME", "mithril"),
		Env:               env,
		Port:              envOr("PORT", "4000"),
		DatabaseURL:       DSNFromEnv(),
		CORSOrigins:       splitList(envOr("CORS_ORIGINS", "http://localhost:3000,http://localhost:4000")),
		EnableRegister:    envBool("ENABLE_REGISTER", false),
		EnableHelmet:      envBool("ENABLE_HELMET", true),
		EnableCompression: envBool("ENABLE_COMPRESSION", false),
		EnableAdmin:       envBool("ENABLE_ADMIN_PANEL", false) || fileExists(".admin-panel-enabled"),
		EnableMonitor:     envBool("ENABLE_MONITOR", false),
		EnableDocs:        envBool("ENABLE_DOCS", isDevelopmentEnv(env)),
		Timezone:          firstNonEmpty(os.Getenv("APP_TIMEZONE"), os.Getenv("TZ"), "UTC"),
		LogRequests:       strings.ToLower(envOr("LOG_REQUESTS", "errors")),
		TrustProxy:        envBool("TRUST_PROXY", true),
		ProxyHeader:       envOr("PROXY_HEADER", "X-Real-IP"),
		PprofAddr:         strings.TrimSpace(os.Getenv("PPROF_ADDR")),
		DBMaxConns:        int32(envInt("DB_MAX_CONNS", 0)),
		DBMinConns:        int32(envInt("DB_MIN_CONNS", 0)),
		ReadTimeout:       envDuration("READ_TIMEOUT", 30*time.Second),
		WriteTimeout:      envDuration("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       envDuration("IDLE_TIMEOUT", 120*time.Second),
		BodyLimit:         envInt("BODY_LIMIT", 4<<20),
		Prefork:           envBool("PREFORK", false),
		HighThroughput:    envBool("HIGH_THROUGHPUT", false),
	}
	switch c.LogRequests {
	case "errors", "all", "none":
	default:
		return Config{}, fmt.Errorf("LOG_REQUESTS must be errors, all or none (got %q)", c.LogRequests)
	}
	secret, err := ResolveJWTSecret(os.Getenv("JWT_SECRET"), env)
	if err != nil {
		return Config{}, err
	}
	c.JWTSecret = secret
	return c, nil
}

// MinJWTSecretLen is the minimum accepted JWT_SECRET length outside development (HS256 wants >= 256 bits).
const MinJWTSecretLen = 32

// placeholderJWTSecrets are example values shipped in env files, k8s manifests and docs. They are
// public, so a server using one is trivially forgeable.
var placeholderJWTSecrets = map[string]bool{
	"secret":                           true,
	"changeme":                         true,
	"change-me":                        true,
	"change-me-in-production":          true,
	"your-secret":                      true,
	"your-secret-change-in-production": true,
	"your-super-secret-jwt-key-32-characters-long":  true,
	"your-secret-change-in-production-min-32-chars": true,
}

// ResolveJWTSecret validates the JWT secret for the given APP_ENV.
//   - Outside development (including an unset APP_ENV) the secret is required, must be at least
//     MinJWTSecretLen bytes and must not be a known placeholder.
//   - In development an empty secret is replaced by a random per-process value.
func ResolveJWTSecret(secret, appEnv string) (string, error) {
	secret = strings.TrimSpace(secret)
	dev := isDevelopmentEnv(appEnv)
	if secret == "" {
		if !dev {
			return "", errors.New("JWT_SECRET is required (generate one with `openssl rand -base64 32`), or set APP_ENV=development for local use")
		}
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return "", fmt.Errorf("generate dev JWT secret: %w", err)
		}
		log.Println("warning: JWT_SECRET not set; using a random per-process secret (APP_ENV=development). Tokens are invalidated on restart.")
		return hex.EncodeToString(b), nil
	}
	if placeholderJWTSecrets[strings.ToLower(secret)] {
		if !dev {
			return "", errors.New("JWT_SECRET is a public placeholder value; generate a real one with `openssl rand -base64 32`")
		}
		log.Println("warning: JWT_SECRET is a placeholder value; never deploy with it")
	}
	if len(secret) < MinJWTSecretLen && !dev {
		return "", fmt.Errorf("JWT_SECRET must be at least %d bytes (got %d)", MinJWTSecretLen, len(secret))
	}
	return secret, nil
}

// DSNFromEnv returns DATABASE_URL, or a URL built from DB_* variables when DB_HOST is set, or "".
// Credentials are URL-escaped, so passwords containing @ / : # ? work.
func DSNFromEnv() string {
	if s := strings.TrimSpace(os.Getenv("DATABASE_URL")); s != "" {
		return s
	}
	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	if host == "" {
		return ""
	}
	port := envOr("DB_PORT", "5432")
	if _, err := strconv.Atoi(port); err != nil {
		port = "5432"
	}
	u := &url.URL{
		Scheme: "postgres",
		Host:   host + ":" + port,
		Path:   "/" + envOr("DB_NAME", "mithril"),
	}
	user := envOr("DB_USER", "postgres")
	if pw := os.Getenv("DB_PASSWORD"); pw != "" {
		u.User = url.UserPassword(user, pw)
	} else {
		u.User = url.User(user)
	}
	q := url.Values{}
	q.Set("sslmode", envOr("DB_SSLMODE", "disable"))
	u.RawQuery = q.Encode()
	return u.String()
}

// LoadDotEnv merges KEY=VALUE lines from filename into the environment. Variables already present
// in the environment (even if empty) win. Supports comments, `export KEY=...`, and single or double
// quotes. A missing file is not an error.
func LoadDotEnv(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		key, value, ok := parseDotEnvLine(scanner.Text())
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
	return scanner.Err()
}

func parseDotEnvLine(line string) (key, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		value = value[1 : len(value)-1]
	} else if i := strings.Index(value, " #"); i >= 0 {
		value = strings.TrimSpace(value[:i]) // inline comment on an unquoted value
	}
	return key, value, key != ""
}

func isDevelopmentEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "development", "dev", "local", "test", "testing":
		return true
	}
	return false
}

func envOr(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(v) == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		log.Printf("warning: ignoring invalid %s=%q", key, v)
		return def
	}
	return n
}

// envDuration accepts Go durations ("30s", "2m") or plain seconds ("30"); "0" disables.
func envDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	d, err := time.ParseDuration(v)
	if err != nil || d < 0 {
		log.Printf("warning: ignoring invalid %s=%q", key, v)
		return def
	}
	return d
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
