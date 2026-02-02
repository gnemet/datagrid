package datagrid

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func TestParseParams(t *testing.T) {
	h := &Handler{}

	// Test case 1: Basic pagination
	u, _ := url.Parse("http://example.com/list?limit=25&offset=50&search=test")
	r := &http.Request{URL: u}
	params := h.ParseParams(r)

	if params.Limit != 25 {
		t.Errorf("Expected limit 25, got %d", params.Limit)
	}
	if params.Offset != 50 {
		t.Errorf("Expected offset 50, got %d", params.Offset)
	}
	if params.Search != "test" {
		t.Errorf("Expected search 'test', got '%s'", params.Search)
	}

	// Test case 2: Sorting
	u2, _ := url.Parse("http://example.com/list?sort=name:ASC&sort=created_at:DESC")
	r2 := &http.Request{URL: u2}
	params2 := h.ParseParams(r2)

	if len(params2.Sort) != 2 {
		t.Errorf("Expected 2 sort params, got %d", len(params2.Sort))
	}
}

func TestBuildWhere(t *testing.T) {
	cols := []UIColumn{
		{Field: "name", Type: "text"},
		{Field: "email", Type: "text"},
		{Field: "age", Type: "number"},
	}
	h := &Handler{Columns: cols}

	// Test case 1: Search string
	where, args := h.buildWhere(RequestParams{Search: "john"})
	if where == "" {
		t.Errorf("Expected WHERE clause, got empty")
	}
	if len(args) != 1 || args[0] != "john" {
		t.Errorf("Expected 1 arg 'john', got %v", args)
	}
}

func TestBuildOrder(t *testing.T) {
	h := &Handler{
		Columns: []UIColumn{
			{Field: "name"},
			{Field: "age"},
		},
	}

	// Test case: Multiple sorts
	sorts := []string{"name:ASC", "id:DESC"}
	order := h.buildOrder(sorts)
	expected := "ORDER BY name ASC, id DESC"
	if order != expected {
		t.Errorf("Expected '%s', got '%s'", expected, order)
	}

	// Test case: Invalid format
	sorts2 := []string{"name", "invalid:MODE"}
	order2 := h.buildOrder(sorts2)
	if order2 != "ORDER BY id DESC" { // Default
		t.Errorf("Expected default ORDER BY, got '%s'", order2)
	}
}

func TestIntegrationFetchData(t *testing.T) {
	// Try to load .env from project root
	_ = godotenv.Load("../../.env")

	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5433"
	}
	user := os.Getenv("DB_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		password = "soa123"
	}
	dbname := os.Getenv("DB_NAME")
	if dbname == "" {
		dbname = "db01"
	}
	schema := os.Getenv("DB_SCHEMA")
	if schema == "" {
		schema = "datagrid"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s",
		host, port, user, password, dbname, schema)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skip("Postgres not available, skipping integration test:", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Postgres not reachable, skipping integration test:", err)
		return
	}

	cols := []UIColumn{
		{Field: "id", Label: "ID", Visible: true},
		{Field: "user_name", Label: "User", Visible: true},
	}
	h := &Handler{DB: db, TableName: "responsibility", Columns: cols}

	params := RequestParams{Limit: 5, Offset: 0}
	result, err := h.FetchData(params)
	if err != nil {
		t.Fatalf("FetchData failed: %v", err)
	}

	if result.TotalCount < 2 { // Seeded in init_db.sql
		t.Errorf("Expected at least 2 records, got total count %d", result.TotalCount)
	}
	if len(result.Records) != 2 {
		t.Errorf("Expected 2 records due to limit and available data, got %d", len(result.Records))
	}

	// Check JSON expansion pattern
	if _, ok := result.Records[0]["_json"]; !ok {
		t.Errorf("Expected _json field in record for sidebar support")
	}
}

func contains(s, substr string) bool {
	return (len(s) >= len(substr)) && (fmt.Sprintf("%v", s) != "") && (len(substr) > 0) && (len(s) >= len(substr)) && (func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}
