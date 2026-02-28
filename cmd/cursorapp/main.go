package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gnemet/datagrid"
	"github.com/gnemet/datagrid/database/cursorpool"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Application struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Author  string `yaml:"author"`
	} `yaml:"application"`
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Database []struct {
		Name     string `yaml:"name"`
		Host     string `yaml:"host"`
		Port     string `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		Schema   string `yaml:"schema"`
		Default  bool   `yaml:"default"`
	} `yaml:"database"`
	Catalog struct {
		Path string `yaml:"path"`
	} `yaml:"catalog"`
	CursorPool struct {
		MaxConnections int    `yaml:"max_connections"`
		IdleTimeout    string `yaml:"idle_timeout"`
		AbsTimeout     string `yaml:"abs_timeout"`
	} `yaml:"cursorpool"`
}

func loadConfig() (*Config, error) {
	_ = godotenv.Load()

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	expanded := os.ExpandEnv(string(data))
	var cfg Config
	err = yaml.Unmarshal([]byte(expanded), &cfg)
	return &cfg, err
}

var (
	db      *sql.DB
	pool    *cursorpool.CursorPool
	tmpl    *template.Template
	handler *datagrid.Handler
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	var dbCfg struct {
		Host, Port, User, Password, Database, Schema string
	}
	for _, d := range cfg.Database {
		if d.Default {
			dbCfg = struct{ Host, Port, User, Password, Database, Schema string }{
				d.Host, d.Port, d.User, d.Password, d.Database, d.Schema,
			}
			break
		}
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s,public",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.Database, dbCfg.Schema)

	// Standard DB for non-cursor operations if needed
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("Fatal error", "error", err)
		os.Exit(1)
	}

	// Initialize CursorPool
	idleTimeout, _ := time.ParseDuration(cfg.CursorPool.IdleTimeout)
	if idleTimeout == 0 {
		idleTimeout = 5 * time.Minute
	}
	absTimeout, _ := time.ParseDuration(cfg.CursorPool.AbsTimeout)
	if absTimeout == 0 {
		absTimeout = 1 * time.Hour
	}
	maxConns := cfg.CursorPool.MaxConnections
	if maxConns == 0 {
		maxConns = 10
	}

	pool, err = cursorpool.NewCursorPool(connStr, maxConns, idleTimeout, absTimeout)
	if err != nil {
		slog.Error("Failed to initialize CursorPool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Initialize Datagrid Handler (for metadata/parsing)
	catPath := os.Getenv("CATALOG_PATH")
	if catPath == "" {
		catPath = cfg.Catalog.Path
	}
	handler, err = datagrid.NewHandlerFromCatalog(db, catPath, "en")
	if err != nil {
		slog.Error("Failed to initialize datagrid handler", "error", err)
		os.Exit(1)
	}

	// Load templates
	funcMap := datagrid.TemplateFuncs()
	funcMap["T"] = func(s string) string { return s }
	tmpl = template.Must(template.New("main").Funcs(funcMap).ParseGlob("pkg/datagrid/ui/templates/partials/datagrid/*.html"))
	tmpl = template.Must(tmpl.ParseFiles("pkg/datagrid/ui/templates/cursor_index.html"))

	// Routes
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("pkg/datagrid/ui/static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hasJsonColumn := false
		for _, col := range handler.Columns {
			if strings.Contains(strings.ToLower(col.Type), "json") {
				hasJsonColumn = true
				break
			}
		}

		isPH := strings.Contains(strings.ToLower(handler.IconStyleLibrary), "phosphor")
		data := map[string]interface{}{
			"App": map[string]string{
				"Name":    cfg.Application.Name,
				"Version": cfg.Application.Version,
				"Author":  cfg.Application.Author,
			},
			"Title":            "Cursor-Enabled Personnel",
			"ListEndpoint":     "/list",
			"UIColumns":        handler.Columns,
			"CurrentLang":      "en",
			"LangsJSON":        `["en", "hu"]`,
			"Limit":            10,
			"Offset":           0,
			"HasJSONColumn":    hasJsonColumn,
			"IconStyleLibrary": strings.TrimSpace(handler.IconStyleLibrary),
			"IsPhosphor":       isPH,
			"DBMode":           os.Getenv("DB_MODE"),
		}
		tmpl.ExecuteTemplate(w, "cursor_index.html", data)
	})

	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		params := handler.ParseParams(r)
		query, _, err := handler.BuildGridSQL(params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sid := "cursor-app-session"
		direction := r.URL.Query().Get("dir")
		if direction == "" {
			direction = "FIRST"
		}
		limit := params.Limit

		ctx := context.Background()
		dbMode := os.Getenv("DB_MODE")
		if dbMode == "" {
			dbMode = "connectless"
		}

		var results []map[string]interface{}

		if dbMode == "refcursor" {
			// Re-initialize cursor on FIRST or if missing (simplified)
			if direction == "FIRST" {
				if _, err := pool.InitializeCursor(ctx, sid, query); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

			results, err = pool.FetchPage(ctx, sid, direction, limit)
			if err != nil {
				// If session died, try re-init
				if _, initErr := pool.InitializeCursor(ctx, sid, query); initErr == nil {
					results, err = pool.FetchPage(ctx, sid, direction, limit)
				}
			}
		} else {
			// Connectless mode: Offset/Limit translation
			// We'll use a simple session-based offset for this demo
			offset := 0
			// In a real app, this would be in URL or a more robust session
			// For this test, we'll use a hacky global or just demo FIRST/NEXT
			switch direction {
			case "NEXT":
				offset = 10
			case "PRIOR":
				offset = 0
			case "LAST":
				offset = 20 // Assuming small dataset for demo
			}

			queryWithLimit := fmt.Sprintf("%s LIMIT %d OFFSET %d", query, limit, offset)
			results, err = pool.QueryDirect(ctx, queryWithLimit)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tableResult := &datagrid.TableResult{
			Records:          results,
			UIColumns:        handler.Columns,
			Limit:            params.Limit,
			IconStyleLibrary: handler.IconStyleLibrary,
			IsPhosphor:       strings.Contains(strings.ToLower(handler.IconStyleLibrary), "phosphor"),
		}

		tmpl.ExecuteTemplate(w, "datagrid_table", tableResult)
	})

	fmt.Printf("CursorApp starting at http://localhost:%s\n", cfg.Server.Port)
	if err := http.ListenAndServe(":"+cfg.Server.Port, nil); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}
