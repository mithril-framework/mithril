package dbms

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	sessionCookieName = "adminergo_sid"
	maxRowsDefault    = 100
)

type session struct {
	ID       string
	CSRF     string
	Host     string
	Port     string
	User     string
	Password string
	DB       string
	SSLMode  string
	LoggedIn bool
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

type dbObject struct {
	Name      string
	Type      string
	RowsEst   sql.NullInt64
	TableSize sql.NullString
	IndexSize sql.NullString
}

type columnMeta struct {
	Name     string
	Nullable bool
	DataType string
}

// uiPage drives the phpMyAdmin-style chrome (sidebar highlight + tab bar).
type uiPage struct {
	DB     string
	Schema string
	Table  string
	Tab    string // home | structure | sql | import | export | operations | privileges
}

var store = &sessionStore{sessions: map[string]*session{}}
var backupDBStreamer = streamDatabaseBackupGzip

// ListenAndServe starts the PostgreSQL web UI (Adminer-style). If addr is empty, uses DBMS_ADDR, then ADDR, then ":5050".
func ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", withSession(handleHome))
	mux.HandleFunc("/login", withSession(handleLogin))
	mux.HandleFunc("/logout", withSession(handleLogout))
	mux.HandleFunc("/databases", withSession(auth(handleDatabases)))
	mux.HandleFunc("/schemas", withSession(auth(handleSchemas)))
	mux.HandleFunc("/tables", withSession(auth(handleTables)))
	mux.HandleFunc("/indexes", withSession(auth(handleIndexes)))
	mux.HandleFunc("/table", withSession(auth(handleTableStructure)))
	mux.HandleFunc("/rows", withSession(auth(handleRows)))
	mux.HandleFunc("/insert", withSession(auth(handleInsert)))
	mux.HandleFunc("/update", withSession(auth(handleUpdate)))
	mux.HandleFunc("/delete", withSession(auth(handleDelete)))
	mux.HandleFunc("/query", withSession(auth(handleQuery)))
	mux.HandleFunc("/export/csv", withSession(auth(handleExportCSV)))
	mux.HandleFunc("/export/sql", withSession(auth(handleExportSQL)))
	mux.HandleFunc("/backup/db", withSession(auth(handleBackupDB)))
	mux.HandleFunc("/import/sql", withSession(auth(handleImportSQL)))

	if strings.TrimSpace(addr) == "" {
		addr = env("DBMS_ADDR", env("ADDR", ":5050"))
	}
	log.Printf("mithril-rev dbms listening at http://127.0.0.1%s", addr)
	return http.ListenAndServe(addr, mux)
}

func withSession(next func(http.ResponseWriter, *http.Request, *session)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := ensureSession(w, r)
		if err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		next(w, r, s)
	}
}

func auth(next func(http.ResponseWriter, *http.Request, *session)) func(http.ResponseWriter, *http.Request, *session) {
	return func(w http.ResponseWriter, r *http.Request, s *session) {
		if !s.LoggedIn {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		next(w, r, s)
	}
}

func ensureSession(w http.ResponseWriter, r *http.Request) (*session, error) {
	c, _ := r.Cookie(sessionCookieName)
	if c != nil {
		if s := store.get(c.Value); s != nil {
			return s, nil
		}
	}

	id, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	csrf, err := randomHex(16)
	if err != nil {
		return nil, err
	}
	s := &session{
		ID:      id,
		CSRF:    csrf,
		Port:    "5432",
		SSLMode: "disable",
	}
	store.set(s)
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return s, nil
}

func (ss *sessionStore) get(id string) *session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	return ss.sessions[id]
}

func (ss *sessionStore) set(s *session) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.sessions[s.ID] = s
}

func (ss *sessionStore) delete(id string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.sessions, id)
}

func handleHome(w http.ResponseWriter, r *http.Request, s *session) {
	if s.LoggedIn {
		http.Redirect(w, r, "/databases", http.StatusSeeOther)
		return
	}
	writePage(w, "Login", `<div class="login-page">`+renderLogin("", s)+`</div>`)
}

func handleLogin(w http.ResponseWriter, r *http.Request, s *session) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if !validCSRF(r, s) {
		writePage(w, "Login", `<div class="login-page">`+renderLogin("Invalid CSRF token.", s)+`</div>`)
		return
	}
	s.Host = r.FormValue("host")
	s.Port = r.FormValue("port")
	if s.Port == "" {
		s.Port = "5432"
	}
	s.User = r.FormValue("user")
	s.Password = r.FormValue("password")
	s.DB = r.FormValue("db")
	s.SSLMode = r.FormValue("sslmode")
	if s.SSLMode == "" {
		s.SSLMode = "disable"
	}

	if err := pingSessionDB(r.Context(), s); err != nil {
		writePage(w, "Login", `<div class="login-page">`+renderLogin("Connection failed: "+html.EscapeString(err.Error()), s)+`</div>`)
		return
	}
	s.LoggedIn = true
	http.Redirect(w, r, "/databases", http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request, s *session) {
	if r.Method != http.MethodPost || !validCSRF(r, s) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	store.delete(s.ID)
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleDatabases(w http.ResponseWriter, r *http.Request, s *session) {
	ui := uiPage{Tab: "home"}
	db, err := openSessionDB(r.Context(), s, "postgres")
	if err != nil {
		writeAppErr(w, r, s, ui, "Databases", err)
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), `SELECT datname FROM pg_database WHERE datallowconn = true ORDER BY datname`)
	if err != nil {
		writeAppErr(w, r, s, ui, "Databases", err)
		return
	}
	defer rows.Close()

	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			writeAppErr(w, r, s, ui, "Databases", err)
			return
		}
		dbs = append(dbs, name)
	}

	var b strings.Builder
	b.WriteString(`<h2>Databases</h2><table class="data-table"><thead><tr><th>Database</th><th>Action</th></tr></thead><tbody>`)
	for _, d := range dbs {
		b.WriteString(`<tr><td>` + html.EscapeString(d) + `</td><td class="actions"><a href="/schemas?db=` + escURL(d) + `">Open</a></td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	b.WriteString(`<div class="notice">PostgreSQL server — pick a database to browse schemas and tables.</div>`)
	writeApp(w, r, s, ui, "Databases", b.String())
}

func handleSchemas(w http.ResponseWriter, r *http.Request, s *session) {
	dbName := resolveDBContext(r, s)
	ui := uiPage{DB: dbName, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Schemas", err)
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), `SELECT schema_name FROM information_schema.schemata ORDER BY schema_name`)
	if err != nil {
		writeAppErr(w, r, s, ui, "Schemas", err)
		return
	}
	defer rows.Close()
	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			writeAppErr(w, r, s, ui, "Schemas", err)
			return
		}
		schemas = append(schemas, name)
	}
	var b strings.Builder
	b.WriteString("<h2>Schemas in " + html.EscapeString(dbName) + `</h2><table class="data-table"><thead><tr><th>Schema</th><th>Action</th></tr></thead><tbody>`)
	for _, sc := range schemas {
		b.WriteString(`<tr><td>` + html.EscapeString(sc) + `</td><td class="actions"><a href="/tables?db=` + escURL(dbName) + `&schema=` + escURL(sc) + `">Structure</a></td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	writeApp(w, r, s, ui, "Schemas", b.String())
}

func handleTables(w http.ResponseWriter, r *http.Request, s *session) {
	dbName := resolveDBContext(r, s)
	schema := normalizeSchema(q(r, "schema", "public"))
	ui := uiPage{DB: dbName, Schema: schema, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Tables", err)
		return
	}
	defer db.Close()

	objects, sourceACount, sourceBCount, resolvedSchema, err := loadSchemaObjects(r.Context(), db, schema)
	if err != nil {
		writeAppErr(w, r, s, ui, "Tables", err)
		return
	}
	schema = resolvedSchema
	ui.Schema = schema

	var tables, views, matViews []dbObject
	for _, obj := range objects {
		switch classifyTableType(obj.Type) {
		case "table":
			tables = append(tables, obj)
		case "view":
			views = append(views, obj)
		case "materialized":
			matViews = append(matViews, obj)
		default:
			tables = append(tables, obj)
		}
	}

	var b strings.Builder
	b.WriteString("<h2>Objects in " + html.EscapeString(dbName+"."+schema) + "</h2>")
	total := len(tables) + len(views) + len(matViews)
	if total == 0 {
		b.WriteString(`<p>No tables or views found in this schema.</p>`)
		b.WriteString(`<p>Check schema permissions and verify objects exist in ` + html.EscapeString(dbName+"."+schema) + `.</p>`)
		b.WriteString(renderSchemaDiagnostics(r.Context(), db, q(r, "db", ""), dbName, schema, sourceACount, sourceBCount))
	}
	b.WriteString(renderObjectSection("Tables", dbName, schema, tables))
	b.WriteString(renderObjectSection("Views", dbName, schema, views))
	b.WriteString(renderObjectSection("Materialized Views", dbName, schema, matViews))
	writeApp(w, r, s, ui, "Tables", b.String())
}

func loadSchemaObjects(ctx context.Context, db *sql.DB, schema string) ([]dbObject, int, int, string, error) {
	objectsA, err := loadObjectsFromCatalog(ctx, db, schema)
	if err != nil {
		return nil, 0, 0, schema, err
	}
	objectsB, err := loadObjectsFromViews(ctx, db, schema)
	if err != nil {
		return nil, 0, 0, schema, err
	}

	combined := dedupeObjects(append(objectsA, objectsB...))
	if len(combined) > 0 {
		return combined, len(objectsA), len(objectsB), schema, nil
	}

	resolved, ok, err := resolveSchemaNameCaseInsensitive(ctx, db, schema)
	if err != nil {
		return nil, 0, 0, schema, err
	}
	if ok && resolved != schema {
		objectsA2, err := loadObjectsFromCatalog(ctx, db, resolved)
		if err != nil {
			return nil, 0, 0, schema, err
		}
		objectsB2, err := loadObjectsFromViews(ctx, db, resolved)
		if err != nil {
			return nil, 0, 0, schema, err
		}
		combined2 := dedupeObjects(append(objectsA2, objectsB2...))
		return combined2, len(objectsA2), len(objectsB2), resolved, nil
	}

	return combined, len(objectsA), len(objectsB), schema, nil
}

func loadObjectsFromCatalog(ctx context.Context, db *sql.DB, schema string) ([]dbObject, error) {
	rows, err := db.QueryContext(ctx, `
SELECT
	c.relname AS table_name,
	c.relkind,
	CASE WHEN c.relkind IN ('r', 'p', 'f') THEN c.reltuples::bigint ELSE NULL END AS rows_est,
	CASE WHEN c.relkind IN ('r', 'p', 'f') THEN pg_size_pretty(pg_table_size(c.oid)) ELSE NULL END AS table_size,
	CASE WHEN c.relkind IN ('r', 'p', 'f') THEN pg_size_pretty(pg_indexes_size(c.oid)) ELSE NULL END AS index_size
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = $1
  AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
ORDER BY c.relname`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []dbObject
	for rows.Next() {
		var t dbObject
		if err := rows.Scan(&t.Name, &t.Type, &t.RowsEst, &t.TableSize, &t.IndexSize); err != nil {
			return nil, err
		}
		objects = append(objects, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return objects, nil
}

func loadObjectsFromViews(ctx context.Context, db *sql.DB, schema string) ([]dbObject, error) {
	rows, err := db.QueryContext(ctx, `
SELECT tablename AS object_name, 'BASE TABLE' AS object_type
FROM pg_tables
WHERE schemaname = $1
UNION ALL
SELECT viewname AS object_name, 'VIEW' AS object_type
FROM pg_views
WHERE schemaname = $1
UNION ALL
SELECT matviewname AS object_name, 'MATERIALIZED VIEW' AS object_type
FROM pg_matviews
WHERE schemaname = $1
ORDER BY object_name`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []dbObject
	for rows.Next() {
		var t dbObject
		if err := rows.Scan(&t.Name, &t.Type); err != nil {
			return nil, err
		}
		objects = append(objects, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return objects, nil
}

func dedupeObjects(in []dbObject) []dbObject {
	seen := make(map[string]bool, len(in))
	out := make([]dbObject, 0, len(in))
	for _, o := range in {
		key := strings.ToLower(o.Name) + "|" + classifyTableType(o.Type)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, o)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func resolveSchemaNameCaseInsensitive(ctx context.Context, db *sql.DB, schema string) (string, bool, error) {
	var resolved string
	err := db.QueryRowContext(ctx, `SELECT nspname FROM pg_namespace WHERE lower(nspname) = lower($1) ORDER BY nspname LIMIT 1`, schema).Scan(&resolved)
	if errors.Is(err, sql.ErrNoRows) {
		return schema, false, nil
	}
	if err != nil {
		return schema, false, err
	}
	return resolved, true, nil
}

func renderSchemaDiagnostics(ctx context.Context, db *sql.DB, requestedDB, resolvedDB, schema string, sourceACount, sourceBCount int) string {
	var currentDB, currentUser string
	_ = db.QueryRowContext(ctx, "SELECT current_database(), current_user").Scan(&currentDB, &currentUser)

	var b strings.Builder
	b.WriteString(`<details><summary>Diagnostics</summary>`)
	b.WriteString(`<ul>`)
	b.WriteString(`<li>requested_db: ` + html.EscapeString(blankDefault(requestedDB, "(none)")) + `</li>`)
	b.WriteString(`<li>resolved_db: ` + html.EscapeString(resolvedDB) + `</li>`)
	b.WriteString(`<li>current_database: ` + html.EscapeString(currentDB) + `</li>`)
	b.WriteString(`<li>current_user: ` + html.EscapeString(currentUser) + `</li>`)
	b.WriteString(`<li>normalized_schema: ` + html.EscapeString(schema) + `</li>`)
	b.WriteString(`<li>catalog_count(pg_class): ` + strconv.Itoa(sourceACount) + `</li>`)
	b.WriteString(`<li>fallback_count(pg_tables/views/matviews): ` + strconv.Itoa(sourceBCount) + `</li>`)
	b.WriteString(`</ul></details>`)
	return b.String()
}

func handleIndexes(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Indexes", err)
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), `
SELECT
	i.relname AS index_name,
	ix.indisprimary,
	ix.indisunique,
	am.amname AS method,
	array_to_string(array_agg(a.attname ORDER BY ord.ordinality), ', ') AS columns
FROM pg_class t
JOIN pg_namespace ns ON ns.oid = t.relnamespace
JOIN pg_index ix ON ix.indrelid = t.oid
JOIN pg_class i ON i.oid = ix.indexrelid
JOIN pg_am am ON am.oid = i.relam
LEFT JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS ord(attnum, ordinality) ON true
LEFT JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ord.attnum
WHERE ns.nspname = $1 AND t.relname = $2
GROUP BY i.relname, ix.indisprimary, ix.indisunique, am.amname
ORDER BY ix.indisprimary DESC, ix.indisunique DESC, i.relname`, schema, table)
	if err != nil {
		writeAppErr(w, r, s, ui, "Indexes", err)
		return
	}
	defer rows.Close()

	type idxRow struct {
		Name    string
		Primary bool
		Unique  bool
		Method  string
		Cols    sql.NullString
	}

	var idxs []idxRow
	for rows.Next() {
		var ir idxRow
		if err := rows.Scan(&ir.Name, &ir.Primary, &ir.Unique, &ir.Method, &ir.Cols); err != nil {
			writeAppErr(w, r, s, ui, "Indexes", err)
			return
		}
		idxs = append(idxs, ir)
	}
	if err := rows.Err(); err != nil {
		writeAppErr(w, r, s, ui, "Indexes", err)
		return
	}

	var b strings.Builder
	b.WriteString("<h2>Indexes: " + html.EscapeString(schema+"."+table) + "</h2>")
	if len(idxs) == 0 {
		b.WriteString("<p>No indexes found for this object.</p>")
		writeApp(w, r, s, ui, "Indexes", b.String())
		return
	}
	b.WriteString(`<table class="data-table"><thead><tr><th>Name</th><th>Type</th><th>Method</th><th>Columns</th></tr></thead><tbody>`)
	for _, ix := range idxs {
		kind := "INDEX"
		if ix.Primary {
			kind = "PRIMARY"
		} else if ix.Unique {
			kind = "UNIQUE"
		}
		b.WriteString("<tr><td>" + html.EscapeString(ix.Name) + "</td><td>" + kind + "</td><td>" + html.EscapeString(ix.Method) + "</td><td>" + html.EscapeString(ix.Cols.String) + "</td></tr>")
	}
	b.WriteString(`</tbody></table>`)
	writeApp(w, r, s, ui, "Indexes", b.String())
}

func renderObjectSection(title, dbName, schema string, objs []dbObject) string {
	var b strings.Builder
	b.WriteString("<h3>" + html.EscapeString(title) + "</h3>")
	if len(objs) == 0 {
		b.WriteString("<p>No " + strings.ToLower(html.EscapeString(title)) + " found.</p>")
		return b.String()
	}
	b.WriteString(`<table class="data-table"><thead><tr><th>Name</th><th>Type</th><th>Rows (est)</th><th>Table size</th><th>Index size</th><th>Actions</th></tr></thead><tbody>`)
	for _, t := range objs {
		base := "db=" + escURL(dbName) + "&schema=" + escURL(schema) + "&table=" + escURL(t.Name)
		rowsEst := "-"
		if t.RowsEst.Valid {
			rowsEst = strconv.FormatInt(t.RowsEst.Int64, 10)
		}
		tableSize := "-"
		if t.TableSize.Valid {
			tableSize = t.TableSize.String
		}
		indexSize := "-"
		if t.IndexSize.Valid {
			indexSize = t.IndexSize.String
		}
		b.WriteString("<tr><td>" + html.EscapeString(t.Name) + "</td><td>" + html.EscapeString(t.Type) + "</td><td>" + html.EscapeString(rowsEst) + "</td><td>" + html.EscapeString(tableSize) + "</td><td>" + html.EscapeString(indexSize) + "</td><td class=\"actions\">")
		b.WriteString(`<a href="/table?` + base + `">structure</a> | `)
		b.WriteString(`<a href="/indexes?` + base + `">indexes</a> | `)
		b.WriteString(`<a href="/rows?` + base + `">rows</a> | `)
		b.WriteString(`<a href="/insert?` + base + `">insert</a> | `)
		b.WriteString(`<a href="/export/csv?` + base + `">csv</a> | `)
		b.WriteString(`<a href="/export/sql?` + base + `">sql</a>`)
		b.WriteString(`</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func classifyTableType(raw string) string {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "R", "P":
		return "table"
	case "F":
		return "table"
	case "V":
		return "view"
	case "M":
		return "materialized"
	case "BASE TABLE":
		return "table"
	case "VIEW":
		return "view"
	case "MATERIALIZED VIEW":
		return "materialized"
	default:
		return "table"
	}
}

func handleTableStructure(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Table Structure", err)
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), `
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`, schema, table)
	if err != nil {
		writeAppErr(w, r, s, ui, "Table Structure", err)
		return
	}
	defer rows.Close()

	var b strings.Builder
	b.WriteString("<h2>Structure: " + html.EscapeString(schema+"."+table) + "</h2>")
	b.WriteString(`<table class="data-table"><thead><tr><th>Column</th><th>Type</th><th>Nullable</th><th>Default</th></tr></thead><tbody>`)
	for rows.Next() {
		var col, typ, nullable string
		var def sql.NullString
		if err := rows.Scan(&col, &typ, &nullable, &def); err != nil {
			writeAppErr(w, r, s, ui, "Table Structure", err)
			return
		}
		b.WriteString("<tr><td>" + html.EscapeString(col) + "</td><td>" + html.EscapeString(typ) + "</td><td>" + html.EscapeString(nullable) + "</td><td>" + html.EscapeString(def.String) + "</td></tr>")
	}
	b.WriteString(`</tbody></table>`)
	writeApp(w, r, s, ui, "Table Structure", b.String())
}

func handleRows(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "structure"}
	limit := q(r, "limit", strconv.Itoa(maxRowsDefault))
	limitN, _ := strconv.Atoi(limit)
	if limitN <= 0 || limitN > 1000 {
		limitN = maxRowsDefault
	}

	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Rows", err)
		return
	}
	defer db.Close()

	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d", quoteIdent(schema), quoteIdent(table), limitN)
	rows, err := db.QueryContext(r.Context(), query)
	if err != nil {
		writeAppErr(w, r, s, ui, "Rows", err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		writeAppErr(w, r, s, ui, "Rows", err)
		return
	}

	pkCols, _ := pkColumns(r.Context(), db, schema, table)
	var b strings.Builder
	b.WriteString(`<h2>Rows: ` + html.EscapeString(schema+"."+table) + `</h2>`)
	b.WriteString(`<p><a href="/insert?db=` + escURL(dbName) + `&schema=` + escURL(schema) + `&table=` + escURL(table) + `">Insert row</a></p>`)
	b.WriteString(`<table class="data-table"><thead><tr>`)
	for _, c := range cols {
		b.WriteString("<th>" + html.EscapeString(c) + "</th>")
	}
	b.WriteString("<th>Actions</th></tr></thead><tbody>")

	for rows.Next() {
		values := make([]any, len(cols))
		valPtrs := make([]any, len(cols))
		for i := range values {
			valPtrs[i] = &values[i]
		}
		if err := rows.Scan(valPtrs...); err != nil {
			writeAppErr(w, r, s, ui, "Rows", err)
			return
		}
		rowMap := map[string]string{}
		b.WriteString("<tr>")
		for i, c := range cols {
			sv := toCell(values[i])
			rowMap[c] = sv
			titleAttr, shown := rowsBrowseTD(sv)
			b.WriteString("<td" + titleAttr + ">" + html.EscapeString(shown) + "</td>")
		}

		editURL := "/update?db=" + escURL(dbName) + "&schema=" + escURL(schema) + "&table=" + escURL(table)
		delURL := "/delete?db=" + escURL(dbName) + "&schema=" + escURL(schema) + "&table=" + escURL(table)
		for _, pk := range pkCols {
			editURL += "&pk_" + escURL(pk) + "=" + escURL(rowMap[pk])
			delURL += "&pk_" + escURL(pk) + "=" + escURL(rowMap[pk])
		}
		b.WriteString(`<td class="actions"><a href="` + editURL + `">edit</a>`)
		b.WriteString(`<form method="post" action="` + delURL + `" style="display:inline" onsubmit="return confirm('Delete this row?');">`)
		b.WriteString(`<input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
		b.WriteString(`<button type="submit">delete</button></form></td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	writeApp(w, r, s, ui, "Rows", b.String())
}

func handleInsert(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Insert", err)
		return
	}
	defer db.Close()

	cols, err := tableColumns(r.Context(), db, schema, table)
	if err != nil {
		writeAppErr(w, r, s, ui, "Insert", err)
		return
	}
	metaByName, err := tableColumnMeta(r.Context(), db, schema, table)
	if err != nil {
		writeAppErr(w, r, s, ui, "Insert", err)
		return
	}

	if r.Method == http.MethodPost {
		if !validCSRF(r, s) {
			writeAppErr(w, r, s, ui, "Insert", errors.New("invalid CSRF token"))
			return
		}
		var names []string
		var params []string
		var args []any
		argNo := 1
		for _, c := range cols {
			val := r.FormValue("c_" + c)
			meta := metaByName[c]
			if val == "" && !meta.Nullable {
				continue
			}
			names = append(names, quoteIdent(c))
			params = append(params, fmt.Sprintf("$%d", argNo))
			args = append(args, coerceBlankToNull(val, meta.Nullable))
			argNo++
		}
		if len(names) == 0 {
			writeAppErr(w, r, s, ui, "Insert", errors.New("no values provided"))
			return
		}
		stmt := fmt.Sprintf("INSERT INTO %s.%s (%s) VALUES (%s)", quoteIdent(schema), quoteIdent(table), strings.Join(names, ", "), strings.Join(params, ", "))
		if _, err := db.ExecContext(r.Context(), stmt, args...); err != nil {
			writeAppErr(w, r, s, ui, "Insert", err)
			return
		}
		http.Redirect(w, r, "/rows?db="+escURL(dbName)+"&schema="+escURL(schema)+"&table="+escURL(table), http.StatusSeeOther)
		return
	}

	var b strings.Builder
	b.WriteString(`<h2>Insert into ` + html.EscapeString(schema+"."+table) + `</h2>`)
	b.WriteString(`<div class="form-box"><form method="post">`)
	b.WriteString(`<input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
	b.WriteString(`<input type="hidden" name="db" value="` + html.EscapeString(dbName) + `">`)
	for _, c := range cols {
		b.WriteString(`<div><label>` + html.EscapeString(c) + ` <input name="c_` + html.EscapeString(c) + `"></label></div>`)
	}
	b.WriteString(`<button type="submit">Insert</button></form></div>`)
	writeApp(w, r, s, ui, "Insert", b.String())
}

func handleUpdate(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Update", err)
		return
	}
	defer db.Close()

	cols, err := tableColumns(r.Context(), db, schema, table)
	if err != nil {
		writeAppErr(w, r, s, ui, "Update", err)
		return
	}
	metaByName, err := tableColumnMeta(r.Context(), db, schema, table)
	if err != nil {
		writeAppErr(w, r, s, ui, "Update", err)
		return
	}
	pkCols, err := pkColumns(r.Context(), db, schema, table)
	if err != nil || len(pkCols) == 0 {
		writeAppErr(w, r, s, ui, "Update", errors.New("table must have primary key for update"))
		return
	}
	pkVals, ok := pkValuesFromRequest(r, pkCols)
	if !ok {
		writeAppErr(w, r, s, ui, "Update", errors.New("missing primary key values in URL"))
		return
	}

	if r.Method == http.MethodPost {
		if !validCSRF(r, s) {
			writeAppErr(w, r, s, ui, "Update", errors.New("invalid CSRF token"))
			return
		}
		setParts := []string{}
		args := []any{}
		argNo := 1
		for _, c := range cols {
			if contains(pkCols, c) {
				continue
			}
			val := r.FormValue("c_" + c)
			setParts = append(setParts, fmt.Sprintf("%s=$%d", quoteIdent(c), argNo))
			args = append(args, coerceBlankToNull(val, metaByName[c].Nullable))
			argNo++
		}
		where, wArgs := pkWhereClause(pkCols, pkVals, argNo)
		args = append(args, wArgs...)
		stmt := fmt.Sprintf("UPDATE %s.%s SET %s WHERE %s", quoteIdent(schema), quoteIdent(table), strings.Join(setParts, ", "), where)
		if _, err := db.ExecContext(r.Context(), stmt, args...); err != nil {
			writeAppErr(w, r, s, ui, "Update", err)
			return
		}
		http.Redirect(w, r, "/rows?db="+escURL(dbName)+"&schema="+escURL(schema)+"&table="+escURL(table), http.StatusSeeOther)
		return
	}

	current, err := fetchOneByPK(r.Context(), db, schema, table, cols, pkCols, pkVals)
	if err != nil {
		writeAppErr(w, r, s, ui, "Update", err)
		return
	}

	var b strings.Builder
	b.WriteString(`<h2>Update ` + html.EscapeString(schema+"."+table) + `</h2>`)
	b.WriteString(`<div class="form-box"><form method="post"><input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
	b.WriteString(`<input type="hidden" name="db" value="` + html.EscapeString(dbName) + `">`)
	for _, c := range cols {
		val := current[c]
		ro := ""
		if contains(pkCols, c) {
			ro = ` readonly`
		}
		b.WriteString(`<div><label>` + html.EscapeString(c) + ` <input name="c_` + html.EscapeString(c) + `" value="` + html.EscapeString(val) + `"` + ro + `></label></div>`)
	}
	b.WriteString(`<button type="submit">Update</button></form></div>`)
	writeApp(w, r, s, ui, "Update", b.String())
}

func handleDelete(w http.ResponseWriter, r *http.Request, s *session) {
	if r.Method != http.MethodPost || !validCSRF(r, s) {
		http.Redirect(w, r, "/databases", http.StatusSeeOther)
		return
	}
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "structure"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Delete", err)
		return
	}
	defer db.Close()

	pkCols, err := pkColumns(r.Context(), db, schema, table)
	if err != nil || len(pkCols) == 0 {
		writeAppErr(w, r, s, ui, "Delete", errors.New("table must have primary key for delete"))
		return
	}
	pkVals, ok := pkValuesFromRequest(r, pkCols)
	if !ok {
		writeAppErr(w, r, s, ui, "Delete", errors.New("missing primary key values in URL"))
		return
	}
	where, args := pkWhereClause(pkCols, pkVals, 1)
	stmt := fmt.Sprintf("DELETE FROM %s.%s WHERE %s", quoteIdent(schema), quoteIdent(table), where)
	if _, err := db.ExecContext(r.Context(), stmt, args...); err != nil {
		writeAppErr(w, r, s, ui, "Delete", err)
		return
	}
	http.Redirect(w, r, "/rows?db="+escURL(dbName)+"&schema="+escURL(schema)+"&table="+escURL(table), http.StatusSeeOther)
}

func handleQuery(w http.ResponseWriter, r *http.Request, s *session) {
	dbName := resolveDBContext(r, s)
	ui := uiPage{DB: dbName, Schema: normalizeSchema(q(r, "schema", "public")), Tab: "sql"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "SQL Query", err)
		return
	}
	defer db.Close()

	var sqlIn, out string
	if r.Method == http.MethodPost {
		if !validCSRF(r, s) {
			writeAppErr(w, r, s, ui, "SQL Query", errors.New("invalid CSRF token"))
			return
		}
		sqlIn = r.FormValue("sql")
		out, err = runQueryHTML(r.Context(), db, sqlIn)
		if err != nil {
			out = `<p class="err">` + html.EscapeString(err.Error()) + `</p>`
		}
	}

	var b strings.Builder
	b.WriteString("<h2>SQL Query (" + html.EscapeString(dbName) + ")</h2>")
	b.WriteString(`<div class="form-box"><form method="post"><input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
	b.WriteString(`<input type="hidden" name="db" value="` + html.EscapeString(dbName) + `">`)
	b.WriteString(`<textarea name="sql" rows="10" style="width:100%">` + html.EscapeString(sqlIn) + `</textarea><br>`)
	b.WriteString(`<button type="submit">Run</button></form></div>`)
	b.WriteString(out)
	writeApp(w, r, s, ui, "SQL Query", b.String())
}

func handleExportCSV(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "export"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Export CSV", err)
		return
	}
	defer db.Close()

	rows, err := db.QueryContext(r.Context(), fmt.Sprintf("SELECT * FROM %s.%s", quoteIdent(schema), quoteIdent(table)))
	if err != nil {
		writeAppErr(w, r, s, ui, "Export CSV", err)
		return
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s.csv", schema, table))
	cw := csv.NewWriter(w)
	_ = cw.Write(cols)
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return
		}
		rec := make([]string, len(cols))
		for i := range vals {
			rec[i] = toCell(vals[i])
		}
		_ = cw.Write(rec)
	}
	cw.Flush()
}

func handleExportSQL(w http.ResponseWriter, r *http.Request, s *session) {
	dbName, schema, table := trio(r, s)
	ui := uiPage{DB: dbName, Schema: schema, Table: table, Tab: "export"}
	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Export SQL", err)
		return
	}
	defer db.Close()

	w.Header().Set("Content-Type", "application/sql")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s.sql", schema, table))
	_, _ = fmt.Fprintf(w, "-- adminergo SQL export\n")
	_, _ = fmt.Fprintf(w, "-- table %s.%s\n\n", schema, table)

	rows, err := db.QueryContext(r.Context(), `
SELECT column_name, data_type
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`, schema, table)
	if err != nil {
		_, _ = io.WriteString(w, "-- error: "+err.Error()+"\n")
		return
	}
	defer rows.Close()

	type colDef struct{ Name, Type string }
	var defs []colDef
	for rows.Next() {
		var d colDef
		_ = rows.Scan(&d.Name, &d.Type)
		defs = append(defs, d)
	}
	_, _ = fmt.Fprintf(w, "CREATE TABLE IF NOT EXISTS %s.%s (\n", quoteIdent(schema), quoteIdent(table))
	for i, d := range defs {
		sep := ","
		if i == len(defs)-1 {
			sep = ""
		}
		_, _ = fmt.Fprintf(w, "  %s %s%s\n", quoteIdent(d.Name), d.Type, sep)
	}
	_, _ = io.WriteString(w, ");\n\n")
}

func handleImportSQL(w http.ResponseWriter, r *http.Request, s *session) {
	dbName := resolveDBContext(r, s)
	ui := uiPage{DB: dbName, Schema: normalizeSchema(q(r, "schema", "public")), Tab: "import"}
	if r.Method == http.MethodGet {
		var b strings.Builder
		b.WriteString(`<h2>Import SQL into ` + html.EscapeString(dbName) + `</h2>`)
		b.WriteString(`<div class="form-box"><form method="post"><input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
		b.WriteString(`<input type="hidden" name="db" value="` + html.EscapeString(dbName) + `">`)
		b.WriteString(`<textarea name="sql" rows="14" style="width:100%"></textarea><br><button type="submit">Execute</button></form></div>`)
		writeApp(w, r, s, ui, "Import SQL", b.String())
		return
	}
	if !validCSRF(r, s) {
		writeAppErr(w, r, s, ui, "Import SQL", errors.New("invalid CSRF token"))
		return
	}

	db, err := openSessionDB(r.Context(), s, dbName)
	if err != nil {
		writeAppErr(w, r, s, ui, "Import SQL", err)
		return
	}
	defer db.Close()
	sqlIn := r.FormValue("sql")
	if _, err := db.ExecContext(r.Context(), sqlIn); err != nil {
		writeAppErr(w, r, s, ui, "Import SQL", err)
		return
	}
	writeApp(w, r, s, ui, "Import SQL", `<p>Import executed successfully.</p>`)
}

func handleBackupDB(w http.ResponseWriter, r *http.Request, s *session) {
	if r.Method != http.MethodPost || !validCSRF(r, s) {
		http.Redirect(w, r, "/databases", http.StatusSeeOther)
		return
	}

	dbName := resolveDBContext(r, s)
	filename := backupFilename(dbName, time.Now())
	if err := backupDBStreamer(r.Context(), w, s, dbName, filename); err != nil {
		ui := uiPage{DB: dbName, Schema: normalizeSchema(q(r, "schema", "public")), Tab: "home"}
		writeAppErr(w, r, s, ui, "Backup DB", err)
		return
	}
}

func streamDatabaseBackupGzip(ctx context.Context, w http.ResponseWriter, s *session, dbName, filename string) error {
	db, err := openSessionDB(ctx, s, dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	schema := normalizeSchema("public")
	tables, err := listBackupTables(ctx, db, schema)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	gz := gzip.NewWriter(w)
	defer gz.Close()
	out := bufio.NewWriter(gz)
	defer out.Flush()

	if err := writeBackupHeader(out, dbName); err != nil {
		return err
	}
	for _, table := range tables {
		if err := writeCreateTableDDL(ctx, out, db, schema, table); err != nil {
			return err
		}
	}
	for _, table := range tables {
		if err := writeTableIndexesDDL(ctx, out, db, schema, table); err != nil {
			return err
		}
	}
	for _, table := range tables {
		if err := writeTableDataCopy(ctx, out, db, schema, table); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(out, "COMMIT;\n"); err != nil {
		return err
	}
	return nil
}

func writeBackupHeader(w io.Writer, dbName string) error {
	_, err := fmt.Fprintf(w,
		"-- adminergo backup (pure Go)\n-- database: %s\n-- generated_at_utc: %s\n\nBEGIN;\n\nSET client_encoding = 'UTF8';\nSET standard_conforming_strings = on;\nSET check_function_bodies = false;\nSET client_min_messages = warning;\n\n",
		dbName,
		time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func listBackupTables(ctx context.Context, db *sql.DB, schema string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT c.relname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = $1
  AND c.relkind IN ('r', 'p')
ORDER BY c.relname`, schema)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

func writeCreateTableDDL(ctx context.Context, w io.Writer, db *sql.DB, schema, table string) error {
	cols, err := tableColumnsDDL(ctx, db, schema, table)
	if err != nil {
		return err
	}
	constraints, err := tableConstraintsDDL(ctx, db, schema, table)
	if err != nil {
		return err
	}
	allDefs := append(cols, constraints...)
	if len(allDefs) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "DROP TABLE IF EXISTS %s.%s CASCADE;\n", quoteIdent(schema), quoteIdent(table)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "CREATE TABLE %s.%s (\n  %s\n);\n\n", quoteIdent(schema), quoteIdent(table), strings.Join(allDefs, ",\n  ")); err != nil {
		return err
	}
	return nil
}

func tableColumnsDDL(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT
  column_name,
  format_type(a.atttypid, a.atttypmod) AS data_type,
  is_nullable,
  column_default,
  identity_generation
FROM information_schema.columns c
JOIN pg_namespace n ON n.nspname = c.table_schema
JOIN pg_class cls ON cls.relname = c.table_name AND cls.relnamespace = n.oid
JOIN pg_attribute a ON a.attrelid = cls.oid AND a.attname = c.column_name AND a.attnum > 0 AND NOT a.attisdropped
WHERE c.table_schema = $1 AND c.table_name = $2
ORDER BY c.ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var defs []string
	for rows.Next() {
		var name, dataType, isNullable string
		var colDefault sql.NullString
		var identity sql.NullString
		if err := rows.Scan(&name, &dataType, &isNullable, &colDefault, &identity); err != nil {
			return nil, err
		}
		def := quoteIdent(name) + " " + dataType
		if identity.Valid {
			def += " GENERATED " + identity.String + " AS IDENTITY"
		} else if colDefault.Valid {
			def += " DEFAULT " + colDefault.String
		}
		if !strings.EqualFold(isNullable, "YES") {
			def += " NOT NULL"
		}
		defs = append(defs, def)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return defs, nil
}

func tableConstraintsDDL(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT conname, pg_get_constraintdef(oid, true)
FROM pg_constraint
WHERE conrelid = $1::regclass
  AND contype IN ('p', 'u', 'f')
ORDER BY CASE contype WHEN 'p' THEN 1 WHEN 'u' THEN 2 ELSE 3 END, conname`,
		fmt.Sprintf("%s.%s", quoteIdent(schema), quoteIdent(table)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var defs []string
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			return nil, err
		}
		defs = append(defs, "CONSTRAINT "+quoteIdent(name)+" "+def)
	}
	return defs, rows.Err()
}

func writeTableIndexesDDL(ctx context.Context, w io.Writer, db *sql.DB, schema, table string) error {
	rows, err := db.QueryContext(ctx, `
SELECT indexdef
FROM pg_indexes
WHERE schemaname = $1
  AND tablename = $2
  AND indexdef NOT ILIKE 'CREATE UNIQUE INDEX %_pkey ON %'
ORDER BY indexname`, schema, table)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var def string
		if err := rows.Scan(&def); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s;\n", strings.TrimSuffix(def, ";")); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\n")
	return err
}

func writeTableDataCopy(ctx context.Context, w io.Writer, db *sql.DB, schema, table string) error {
	cols, err := tableColumns(ctx, db, schema, table)
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return nil
	}
	if _, err := fmt.Fprintf(w, "COPY %s.%s (%s) FROM stdin;\n", quoteIdent(schema), quoteIdent(table), joinQuoted(cols)); err != nil {
		return err
	}
	query := fmt.Sprintf("SELECT %s FROM %s.%s", joinQuoted(cols), quoteIdent(schema), quoteIdent(table))
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	rawVals := make([]any, len(cols))
	scanDest := make([]any, len(cols))
	for i := range rawVals {
		scanDest[i] = &rawVals[i]
	}
	for rows.Next() {
		if err := rows.Scan(scanDest...); err != nil {
			return err
		}
		for i, v := range rawVals {
			if i > 0 {
				if _, err := io.WriteString(w, "\t"); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, toCopyValue(v)); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = io.WriteString(w, "\\.\n\n")
	return err
}

func toCopyValue(v any) string {
	if v == nil {
		return `\N`
	}
	switch t := v.(type) {
	case []byte:
		return escapeCopyText(string(t))
	case time.Time:
		return escapeCopyText(t.UTC().Format(time.RFC3339Nano))
	default:
		return escapeCopyText(fmt.Sprint(v))
	}
}

func escapeCopyText(s string) string {
	r := strings.NewReplacer(
		`\`, `\\`,
		"\t", `\t`,
		"\n", `\n`,
		"\r", `\r`,
	)
	return r.Replace(s)
}

func backupFilename(dbName string, t time.Time) string {
	return fmt.Sprintf("%s-%s.sql.gz", dbName, t.Format("20060102-150405"))
}

func openSessionDB(ctx context.Context, s *session, dbOverride string) (*sql.DB, error) {
	if dbOverride == "" {
		dbOverride = s.DB
	}
	if dbOverride == "" {
		dbOverride = "postgres"
	}
	db, err := sql.Open("pgx", pgDSN(s, dbOverride))
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	_, _ = db.ExecContext(ctx, "SET statement_timeout = 5000")
	return db, nil
}

func pingSessionDB(ctx context.Context, s *session) error {
	db, err := openSessionDB(ctx, s, s.DB)
	if err != nil {
		return err
	}
	return db.Close()
}

func pgDSN(s *session, dbName string) string {
	u := &url.URL{
		Scheme: "postgres",
		Host:   blankDefault(s.Host, "127.0.0.1") + ":" + blankDefault(s.Port, "5432"),
		Path:   "/" + normalizeDB(dbName),
	}
	if s.User != "" {
		u.User = url.UserPassword(s.User, s.Password)
	}
	q := u.Query()
	q.Set("sslmode", blankDefault(s.SSLMode, "disable"))
	u.RawQuery = q.Encode()
	return u.String()
}

func tableColumns(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, nil
}

func tableColumnMeta(ctx context.Context, db *sql.DB, schema, table string) (map[string]columnMeta, error) {
	rows, err := db.QueryContext(ctx, `
SELECT column_name, is_nullable, data_type
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2
ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]columnMeta{}
	for rows.Next() {
		var name, nullable, dataType string
		if err := rows.Scan(&name, &nullable, &dataType); err != nil {
			return nil, err
		}
		out[name] = columnMeta{
			Name:     name,
			Nullable: strings.EqualFold(nullable, "YES"),
			DataType: dataType,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func pkColumns(ctx context.Context, db *sql.DB, schema, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT kcu.column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name
 AND tc.table_schema = kcu.table_schema
WHERE tc.constraint_type = 'PRIMARY KEY'
  AND tc.table_schema = $1
  AND tc.table_name = $2
ORDER BY kcu.ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, nil
}

func fetchOneByPK(ctx context.Context, db *sql.DB, schema, table string, cols, pkCols []string, pkVals map[string]string) (map[string]string, error) {
	where, args := pkWhereClause(pkCols, pkVals, 1)
	stmt := fmt.Sprintf("SELECT %s FROM %s.%s WHERE %s",
		joinQuoted(cols), quoteIdent(schema), quoteIdent(table), where)
	row := db.QueryRowContext(ctx, stmt, args...)
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	if err := row.Scan(ptrs...); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for i, c := range cols {
		out[c] = toCell(vals[i])
	}
	return out, nil
}

func pkValuesFromRequest(r *http.Request, pkCols []string) (map[string]string, bool) {
	vals := map[string]string{}
	for _, c := range pkCols {
		v := r.URL.Query().Get("pk_" + c)
		if v == "" {
			return nil, false
		}
		vals[c] = v
	}
	return vals, true
}

func pkWhereClause(pkCols []string, pkVals map[string]string, start int) (string, []any) {
	parts := make([]string, 0, len(pkCols))
	args := make([]any, 0, len(pkCols))
	for i, c := range pkCols {
		parts = append(parts, fmt.Sprintf("%s=$%d", quoteIdent(c), start+i))
		args = append(args, pkVals[c])
	}
	return strings.Join(parts, " AND "), args
}

func runQueryHTML(ctx context.Context, db *sql.DB, q string) (string, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return "", nil
	}
	if isLikelySelect(q) {
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		var b strings.Builder
		b.WriteString(`<h3>Result</h3><table class="data-table"><thead><tr>`)
		for _, c := range cols {
			b.WriteString("<th>" + html.EscapeString(c) + "</th>")
		}
		b.WriteString(`</tr></thead><tbody>`)
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return "", err
			}
			b.WriteString("<tr>")
			for _, v := range vals {
				sc := toCell(v)
				b.WriteString("<td" + cellTitleAttr(sc) + ">" + html.EscapeString(sc) + "</td>")
			}
			b.WriteString("</tr>")
		}
		b.WriteString(`</tbody></table>`)
		return b.String(), nil
	}
	res, err := db.ExecContext(ctx, q)
	if err != nil {
		return "", err
	}
	aff, _ := res.RowsAffected()
	return fmt.Sprintf("<p>Affected rows: %d</p>", aff), nil
}

func appCSS() string {
	return `body{margin:0;font-family:Arial,sans-serif;background:#f5f6f7;}
.container{display:flex;height:100vh;}
.sidebar{width:260px;background:#e9ecef;border-right:1px solid #ccc;padding:10px;overflow-y:auto;flex-shrink:0;}
.sidebar h2{font-size:18px;margin:0 0 10px;color:#f39c12;}
.tree ul{list-style:none;padding-left:15px;margin:4px 0;}
.tree li{font-size:13px;margin:3px 0;}
.tree a{color:#0366d6;text-decoration:none;}
.tree a:hover{text-decoration:underline;}
.tree .muted{color:#555;}
.main{flex:1;display:flex;flex-direction:column;min-width:0;}
.topbar{background:#ddd;padding:10px;border-bottom:1px solid #bbb;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:8px;}
.topbar span{font-size:14px;}
.topbar-actions button,.tabs a.tab,.form-box button{padding:6px 10px;border:1px solid #bbb;background:#fff;cursor:pointer;font-size:13px;border-radius:5px;display:inline-block;text-decoration:none;color:inherit;box-sizing:border-box;}
.topbar-actions button{background:#e8e8e8;}
.tabs{background:#f1f1f1;padding:8px;border-bottom:1px solid #ccc;}
.tabs a.tab{margin-right:5px;}
.tabs a.tab:hover{background:#eaeaea;}
.tabs a.tab.tab-active{background:#dfe8f5;border-color:#7c98c4;font-weight:bold;}
.content{padding:15px;overflow:auto;flex:1;}
.data-table{width:100%;border-collapse:separate;border-spacing:0;background:#fff;margin-bottom:20px;border-radius:5px;overflow:hidden;border:1px solid #ccc;}
.data-table th,.data-table td{border:1px solid #ccc;padding:8px;font-size:13px;text-align:left;vertical-align:top;}
.data-table th{background:#f7f7f7;}
.data-table thead th{max-width:24rem;line-height:1.25;overflow-wrap:break-word;word-break:normal;}
.data-table tbody td{max-width:28rem;max-height:7.5rem;overflow:auto;overflow-wrap:break-word;word-break:normal;}
.data-table tbody td.actions{max-width:none;max-height:none;min-width:8rem;overflow:visible;white-space:nowrap;}
.actions a{margin-right:5px;color:#007bff;text-decoration:none;}
.actions a:hover{text-decoration:underline;}
.form-box{background:#fff;border:1px solid #ccc;padding:10px;border-radius:5px;margin-bottom:16px;}
.form-box input,.form-box textarea{padding:5px;margin-right:5px;border:1px solid #ccc;border-radius:5px;}
.notice{margin-top:10px;padding:10px;background:#eef5ff;border:1px solid #aac;font-size:13px;border-radius:5px;}
.err{color:#a00;}
.login-page{min-height:100vh;display:flex;align-items:center;justify-content:center;padding:20px;}
.login-panel{max-width:420px;width:100%;}
.login-panel h2{margin-top:0;color:#f39c12;}
.login-panel .subtitle{margin:-6px 0 12px;color:#666;font-size:14px;}
h2{font-size:18px;margin-top:0;}
h3{font-size:15px;}
textarea,input,button{font-family:inherit;}
`
}

func writePage(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>`+html.EscapeString(title)+`</title>
<style>`+appCSS()+`</style></head><body>`+body+`</body></html>`)
}

func renderLogin(err string, s *session) string {
	var b strings.Builder
	b.WriteString(`<div class="form-box login-panel">`)
	b.WriteString(`<h2>phpMyAdmin</h2>`)
	b.WriteString(`<p class="subtitle">PostgreSQL (AdminerGo)</p>`)
	if err != "" {
		b.WriteString(`<p class="err">` + err + `</p>`)
	}
	b.WriteString(`<form method="post" action="/login">`)
	b.WriteString(`<input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
	b.WriteString(`<div><label>Host <input name="host" value="` + html.EscapeString(s.Host) + `" size="28"></label></div>`)
	b.WriteString(`<div><label>Port <input name="port" value="` + html.EscapeString(s.Port) + `" size="8"></label></div>`)
	b.WriteString(`<div><label>User <input name="user" value="` + html.EscapeString(s.User) + `" size="28"></label></div>`)
	b.WriteString(`<div><label>Password <input type="password" name="password" size="28"></label></div>`)
	b.WriteString(`<div><label>Database <input name="db" value="` + html.EscapeString(s.DB) + `" size="28"></label></div>`)
	b.WriteString(`<div><label>SSL Mode <input name="sslmode" value="` + html.EscapeString(s.SSLMode) + `" size="14"></label></div>`)
	b.WriteString(`<p><button type="submit">Connect</button></p></form></div>`)
	return b.String()
}

const maxSidebarTables = 180

func writeApp(w http.ResponseWriter, r *http.Request, s *session, ui uiPage, title, content string) {
	body := renderAppShell(r.Context(), s, r, ui, content)
	writePage(w, title, body)
}

func writeAppErr(w http.ResponseWriter, r *http.Request, s *session, ui uiPage, title string, err error) {
	msg := `<div class="form-box"><h2>` + html.EscapeString(title) + `</h2><p class="err">` + html.EscapeString(err.Error()) + `</p></div>`
	writeApp(w, r, s, ui, title, msg)
}

func renderAppShell(ctx context.Context, s *session, r *http.Request, ui uiPage, content string) string {
	var b strings.Builder
	b.WriteString(`<div class="container">`)
	b.WriteString(buildSidebarTree(ctx, s, ui))
	b.WriteString(`<div class="main">`)
	b.WriteString(renderTopBar(s, ui))
	b.WriteString(renderTabs(s, ui))
	b.WriteString(`<div class="content">`)
	b.WriteString(content)
	b.WriteString(`</div></div></div>`)
	return b.String()
}

func effectiveDB(ui uiPage, s *session) string {
	d := strings.TrimSpace(ui.DB)
	if d == "" {
		d = strings.TrimSpace(s.DB)
	}
	return normalizeDB(d)
}

func effectiveSchema(ui uiPage) string {
	return normalizeSchema(ui.Schema)
}

func tabActive(ui uiPage, key string) bool {
	switch key {
	case "home":
		return ui.Tab == "home"
	case "structure":
		return ui.Tab == "structure"
	case "sql", "search", "query":
		return ui.Tab == "sql"
	case "export":
		return ui.Tab == "export"
	case "import":
		return ui.Tab == "import"
	case "operations":
		return ui.Tab == "operations"
	case "privileges":
		return ui.Tab == "privileges"
	default:
		return false
	}
}

func renderTabs(s *session, ui uiPage) string {
	db := escURL(effectiveDB(ui, s))
	sch := escURL(effectiveSchema(ui))
	structureHref := "/tables?db=" + db + "&schema=" + sch
	tabs := []struct{ label, key, href string }{
		{"Databases", "home", "/databases"},
		{"Structure", "structure", structureHref},
		{"SQL", "sql", "/query?db=" + db},
		{"Search", "search", "/query?db=" + db},
		{"Query", "query", "/query?db=" + db},
		{"Export", "export", "/databases"},
		{"Import", "import", "/import/sql?db=" + db},
		{"Operations", "operations", "#"},
		{"Privileges", "privileges", "#"},
	}
	var b strings.Builder
	b.WriteString(`<div class="tabs">`)
	for _, t := range tabs {
		cls := "tab"
		if tabActive(ui, t.key) {
			cls = "tab tab-active"
		}
		b.WriteString(`<a href="` + html.EscapeString(t.href) + `" class="` + cls + `">` + html.EscapeString(t.label) + `</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderTopBar(s *session, ui uiPage) string {
	dbLabel := strings.TrimSpace(ui.DB)
	if dbLabel == "" {
		dbLabel = strings.TrimSpace(s.DB)
	}
	if dbLabel == "" {
		dbLabel = "postgres"
	}
	backupDB := normalizeDB(dbLabel)
	var b strings.Builder
	b.WriteString(`<div class="topbar">`)
	b.WriteString(`<span><b>Server:</b> PostgreSQL | <b>Host:</b> ` + html.EscapeString(blankDefault(s.Host, "127.0.0.1")) + `:` + html.EscapeString(blankDefault(s.Port, "5432")) + ` | <b>User:</b> ` + html.EscapeString(s.User) + ` | <b>Database:</b> ` + html.EscapeString(dbLabel) + `</span>`)
	b.WriteString(`<span class="topbar-actions">`)
	b.WriteString(`<form method="post" action="/backup/db">`)
	b.WriteString(`<input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
	b.WriteString(`<input type="hidden" name="db" value="` + html.EscapeString(backupDB) + `">`)
	b.WriteString(`<button type="submit">Backup DB</button></form>`)
	b.WriteString(`<form method="post" action="/logout">`)
	b.WriteString(`<input type="hidden" name="csrf" value="` + html.EscapeString(s.CSRF) + `">`)
	b.WriteString(`<button type="submit">Logout</button></form>`)
	b.WriteString(`</span></div>`)
	return b.String()
}

func buildSidebarTree(ctx context.Context, s *session, ui uiPage) string {
	expandDB := strings.TrimSpace(ui.DB)
	var b strings.Builder
	b.WriteString(`<div class="sidebar">`)
	b.WriteString(`<h2>phpMyAdmin</h2>`)
	b.WriteString(`<div class="tree"><ul>`)
	b.WriteString(`<li><a href="/databases">📁 <span class="muted">Home</span></a></li>`)

	pgdb, err := openSessionDB(ctx, s, "postgres")
	if err != nil {
		b.WriteString(`</ul></div><p class="err">` + html.EscapeString(err.Error()) + `</p></div>`)
		return b.String()
	}
	defer pgdb.Close()

	dRows, err := pgdb.QueryContext(ctx, `SELECT datname FROM pg_database WHERE datallowconn = true ORDER BY datname`)
	if err != nil {
		b.WriteString(`</ul></div><p class="err">` + html.EscapeString(err.Error()) + `</p></div>`)
		return b.String()
	}
	defer dRows.Close()
	var dbs []string
	for dRows.Next() {
		var name string
		if err := dRows.Scan(&name); err != nil {
			b.WriteString(`</ul></div><p class="err">` + html.EscapeString(err.Error()) + `</p></div>`)
			return b.String()
		}
		dbs = append(dbs, name)
	}
	if err := dRows.Err(); err != nil {
		b.WriteString(`</ul></div><p class="err">` + html.EscapeString(err.Error()) + `</p></div>`)
		return b.String()
	}

	wantSchema := normalizeSchema(ui.Schema)
	wantTable := strings.TrimSpace(ui.Table)

	for _, d := range dbs {
		encD := html.EscapeString(d)
		if expandDB == "" || d != expandDB {
			b.WriteString(`<li><a href="/schemas?db=` + escURL(d) + `">📁 ` + encD + `</a></li>`)
			continue
		}

		b.WriteString(`<li><span class="muted">📁</span> <b>` + encD + `</b>`)
		db2, err := openSessionDB(ctx, s, d)
		if err != nil {
			b.WriteString(`<p class="err">` + html.EscapeString(err.Error()) + `</p></li>`)
			continue
		}
		schRows, err := db2.QueryContext(ctx, `SELECT schema_name FROM information_schema.schemata ORDER BY schema_name`)
		if err != nil {
			_ = db2.Close()
			b.WriteString(`<p class="err">` + html.EscapeString(err.Error()) + `</p></li>`)
			continue
		}
		var schemas []string
		for schRows.Next() {
			var sn string
			if err := schRows.Scan(&sn); err != nil {
				break
			}
			schemas = append(schemas, sn)
		}
		_ = schRows.Close()

		b.WriteString(`<ul>`)
		for _, sc := range schemas {
			encS := html.EscapeString(sc)
			if wantSchema == "" || sc != wantSchema {
				b.WriteString(`<li><a href="/tables?db=` + escURL(d) + `&schema=` + escURL(sc) + `">📁 ` + encS + `</a></li>`)
				continue
			}
			b.WriteString(`<li><span class="muted">📁</span> <b>` + encS + `</b>`)
			tRows, err := db2.QueryContext(ctx, `
SELECT c.relname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = $1 AND c.relkind IN ('r','p','v','m')
ORDER BY c.relname
LIMIT $2`, sc, maxSidebarTables+1)
			if err != nil {
				b.WriteString(`<p class="err">` + html.EscapeString(err.Error()) + `</p></li>`)
				continue
			}
			var tables []string
			for tRows.Next() {
				var tn string
				if err := tRows.Scan(&tn); err != nil {
					break
				}
				tables = append(tables, tn)
			}
			_ = tRows.Close()

			b.WriteString(`<ul>`)
			trunc := false
			if len(tables) > maxSidebarTables {
				trunc = true
				tables = tables[:maxSidebarTables]
			}
			for _, tn := range tables {
				label := html.EscapeString(tn)
				ws, we := "", ""
				if wantTable != "" && tn == wantTable {
					ws, we = "<b>", "</b>"
				}
				b.WriteString(`<li><a href="/rows?db=` + escURL(d) + `&schema=` + escURL(sc) + `&table=` + escURL(tn) + `">📄 ` + ws + label + we + `</a></li>`)
			}
			if trunc {
				b.WriteString(`<li class="muted"><a href="/tables?db=` + escURL(d) + `&schema=` + escURL(sc) + `">… more</a></li>`)
			}
			b.WriteString(`</ul></li>`)
		}
		b.WriteString(`</ul>`)
		_ = db2.Close()
		b.WriteString(`</li>`)
	}

	b.WriteString(`</ul></div></div>`)
	return b.String()
}

func validCSRF(r *http.Request, s *session) bool {
	return r.FormValue("csrf") != "" && r.FormValue("csrf") == s.CSRF
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func joinQuoted(cols []string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = quoteIdent(c)
	}
	return strings.Join(out, ", ")
}

func toCell(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case []byte:
		return string(x)
	case time.Time:
		return x.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(x)
	}
}

// rowsBrowseCellDisplay: on /rows only — values longer than 500 runes show first 30 runes + "…".
func rowsBrowseCellDisplay(s string) string {
	const maxFullRunes = 150
	const headRunes = 30
	rr := []rune(s)
	if len(rr) <= maxFullRunes {
		return s
	}
	if len(rr) <= headRunes {
		return s
	}
	return string(rr[:headRunes]) + "…"
}

// rowsBrowseTD builds title attribute and visible text for /rows table cells (PK URLs still use full value).
func rowsBrowseTD(full string) (titleAttr string, display string) {
	display = rowsBrowseCellDisplay(full)
	if display != full {
		tip := full
		const maxTip = 4000
		if len(tip) > maxTip {
			tip = tip[:maxTip] + "…"
		}
		return ` title="` + html.EscapeString(tip) + `"`, display
	}
	return cellTitleAttr(full), display
}

// cellTitleAttr adds a native tooltip for long values (list / SQL result cells are max-height scrolled).
func cellTitleAttr(s string) string {
	if len(s) < 200 {
		return ""
	}
	tip := s
	const maxTip = 4000
	if len(tip) > maxTip {
		tip = tip[:maxTip] + "…"
	}
	return ` title="` + html.EscapeString(tip) + `"`
}

func isLikelySelect(q string) bool {
	t := strings.ToUpper(strings.TrimSpace(q))
	return strings.HasPrefix(t, "SELECT") || strings.HasPrefix(t, "WITH") || strings.HasPrefix(t, "SHOW")
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func trio(r *http.Request, s *session) (dbName, schema, table string) {
	dbName = resolveDBContext(r, s)
	schema = normalizeSchema(q(r, "schema", "public"))
	table = q(r, "table", "")
	return
}

func q(r *http.Request, key, fallback string) string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	return v
}

func normalizeSchema(v string) string {
	n := strings.TrimSpace(v)
	if n == "" {
		return "public"
	}
	return n
}

func normalizeDB(v string) string {
	n := strings.TrimSpace(v)
	if n == "" {
		return "postgres"
	}
	return n
}

func coerceBlankToNull(v string, nullable bool) any {
	if strings.TrimSpace(v) == "" && nullable {
		return nil
	}
	return v
}

func resolveDBContext(r *http.Request, s *session) string {
	urlDB := strings.TrimSpace(r.URL.Query().Get("db"))
	if urlDB != "" {
		s.DB = normalizeDB(urlDB)
		return s.DB
	}

	if r.Method == http.MethodPost {
		formDB := strings.TrimSpace(r.FormValue("db"))
		if formDB != "" {
			s.DB = normalizeDB(formDB)
			return s.DB
		}
	}

	if strings.TrimSpace(s.DB) != "" {
		s.DB = normalizeDB(s.DB)
		return s.DB
	}

	s.DB = "postgres"
	return s.DB
}

func blankDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func contains(sl []string, x string) bool {
	for _, v := range sl {
		if v == x {
			return true
		}
	}
	return false
}

func escURL(s string) string {
	r := strings.NewReplacer(" ", "%20", "#", "%23", "&", "%26", "?", "%3F", "=", "%3D")
	return r.Replace(s)
}

func env(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}
