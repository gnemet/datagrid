package main

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"html/template"
	"log"
	"net/http"
	"os"

	"strings"

	"github.com/gnemet/datagrid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Application struct {
		Name                string `yaml:"name"`
		Version             string `yaml:"version"`
		Author              string `yaml:"author"`
		LOVChooserThreshold int    `yaml:"lov_chooser_threshold"`
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
}

func loadConfig() (*Config, error) {
	_ = godotenv.Load() // Ignore error as it might not exist in prod

	data, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	// Expand env vars in YAML
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	err = yaml.Unmarshal([]byte(expanded), &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

var (
	db   *sql.DB
	tmpl *template.Template
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var dbCfg struct {
		Host, Port, User, Password, Database, Schema string
	}

	for _, d := range cfg.Database {
		if d.Default {
			dbCfg.Host = d.Host
			dbCfg.Port = d.Port
			dbCfg.User = d.User
			dbCfg.Password = d.Password
			dbCfg.Database = d.Database
			dbCfg.Schema = d.Schema
			break
		}
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s,public",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.Database, dbCfg.Schema)

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Load templates with helper functions
	funcMap := datagrid.TemplateFuncs()
	funcMap["T"] = func(s string) string { return s } // Dummy T function

	tmpl = template.Must(template.New("main").Funcs(funcMap).ParseGlob("ui/templates/partials/datagrid/*.html"))
	tmpl = template.Must(tmpl.ParseFiles("ui/templates/index.html", "ui/templates/header.html", "ui/templates/footer.html"))

	// Catalog Discovery
	catalogs := make(map[string]string) // name -> title
	catFiles, _ := os.ReadDir("internal/data/catalog")
	for _, f := range catFiles {
		if strings.HasSuffix(f.Name(), ".json") {
			name := strings.TrimSuffix(f.Name(), ".json")
			// Try to read title from JSON
			content, _ := os.ReadFile("internal/data/catalog/" + f.Name())
			var c struct {
				Title string `json:"title"`
			}
			json.Unmarshal(content, &c)
			if c.Title == "" {
				c.Title = name
			}
			catalogs[name] = c.Title
		}
	}

	// Routes
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		catParam := r.URL.Query().Get("config")
		if catParam == "" {
			catParam = "personnel" // Default
		}

		catPath := fmt.Sprintf("internal/data/catalog/%s.json", catParam)
		gridHandler, err := datagrid.NewHandlerFromCatalog(db, catPath, "en")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load catalog %s: %v", catParam, err), http.StatusInternalServerError)
			return
		}
		gridHandler.LOVChooserThreshold = cfg.Application.LOVChooserThreshold

		hasJsonColumn := false
		for _, col := range gridHandler.Columns {
			if strings.Contains(strings.ToLower(col.Type), "json") {
				hasJsonColumn = true
				break
			}
		}

		data := map[string]interface{}{
			"App": map[string]string{
				"Name":    cfg.Application.Name,
				"Version": cfg.Application.Version,
				"Author":  cfg.Application.Author,
			},
			"Title":            gridHandler.Catalog.Title,
			"ListEndpoint":     "/list",
			"Limit":            10,
			"Offset":           0,
			"UIColumns":        gridHandler.Columns,
			"LangsJSON":        `["en", "hu"]`,
			"CurrentLang":      "en",
			"IconStyleLibrary": strings.TrimSpace(gridHandler.IconStyleLibrary),
			"IsPhosphor":       strings.Contains(strings.ToLower(gridHandler.IconStyleLibrary), "phosphor"),
			"HasJSONColumn":    hasJsonColumn,
			"PivotEndpoint":    "/pivot",
			"ViewMode":         gridHandler.Catalog.Type,
			"Catalogs":         catalogs,
			"CurrentCatalog":      catParam,
			"LOVChooserThreshold": cfg.Application.LOVChooserThreshold,
			"IsQueryMode":         gridHandler.IsQueryMode,
			"QueryParams":         gridHandler.QueryParams,
			"ExecuteEndpoint":     "/execute",
			"CurrentUser":         "",
		}

		tmpl.ExecuteTemplate(w, "index.html", data)
	})

	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		catParam := r.URL.Query().Get("config")
		if catParam == "" {
			catParam = "personnel"
		}
		catPath := fmt.Sprintf("internal/data/catalog/%s.json", catParam)
		gridHandler, err := datagrid.NewHandlerFromCatalog(db, catPath, "en")
		if err != nil {
			log.Printf("Error loading catalog %s: %v", catParam, err)
			http.Error(w, fmt.Sprintf("Error loading catalog: %v", err), http.StatusInternalServerError)
			return
		}
		gridHandler.LOVChooserThreshold = cfg.Application.LOVChooserThreshold
		gridHandler.ListEndpoint = "/list?config=" + catParam
		gridHandler.PivotEndpoint = "/pivot?config=" + catParam
		gridHandler.AppName = "Personnel Analytics"
		gridHandler.Catalogs = map[string]string{
			"personnel":        "Personnel & Payroll",
			"pivot_demo":       "Personnel & Payroll Pivot",
			"pivot_column_lov": "Personnel Analytics",
			"pivot_multi_test": "Personnel Analytics & Pivot",
		}
		gridHandler.ServeHTTP(w, r)

	})

	http.HandleFunc("/pivot", func(w http.ResponseWriter, r *http.Request) {
		catParam := r.URL.Query().Get("config")
		if catParam == "" {
			catParam = "personnel"
		}
		catPath := fmt.Sprintf("internal/data/catalog/%s.json", catParam)
		gridHandler, err := datagrid.NewHandlerFromCatalog(db, catPath, "en")
		if err != nil {
			log.Printf("Error loading catalog %s: %v", catParam, err)
			http.Error(w, fmt.Sprintf("Error loading catalog: %v", err), http.StatusInternalServerError)
			return
		}
		gridHandler.LOVChooserThreshold = cfg.Application.LOVChooserThreshold
		gridHandler.ListEndpoint = "/list?config=" + catParam
		gridHandler.PivotEndpoint = "/pivot?config=" + catParam
		gridHandler.AppName = "Personnel Analytics"
		gridHandler.Catalogs = map[string]string{
			"personnel":        "Personnel & Payroll",
			"pivot_demo":       "Personnel & Payroll Pivot",
			"pivot_column_lov": "Personnel Analytics",
			"pivot_multi_test": "Personnel Analytics & Pivot",
		}
		gridHandler.ServeHTTP(w, r)

	})

	http.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		catParam := r.URL.Query().Get("config")
		if catParam == "" {
			catParam = "query_demo"
		}
		catPath := fmt.Sprintf("internal/data/catalog/%s.json", catParam)
		gridHandler, err := datagrid.NewHandlerFromCatalog(db, catPath, "en")
		if err != nil {
			log.Printf("Error loading catalog %s: %v", catParam, err)
			http.Error(w, fmt.Sprintf("Error loading catalog: %v", err), http.StatusInternalServerError)
			return
		}
		gridHandler.LOVChooserThreshold = cfg.Application.LOVChooserThreshold
		gridHandler.ListEndpoint = "/list?config=" + catParam
		gridHandler.ExecuteEndpoint = "/execute?config=" + catParam
		gridHandler.AppName = "Personnel Analytics"
		gridHandler.ExecuteQuery(w, r)
	})

	fmt.Printf("Server starting at http://localhost:%s\n", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, nil))
}
