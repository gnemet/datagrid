package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gnemet/datagrid"
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

	// Load templates
	tmpl = template.Must(template.ParseGlob("ui/templates/partials/datagrid/*.html"))
	tmpl = template.Must(tmpl.ParseFiles("ui/templates/index.html"))

	// Datagrid Setup (Using Catalog from Config)
	gridHandler, err := datagrid.NewHandlerFromCatalog(db, cfg.Catalog.Path, "en")
	if err != nil {
		log.Fatalf("Failed to initialize handler from catalog: %v", err)
	}

	// Routes
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("ui/static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]interface{}{
			"App": map[string]string{
				"Name":    cfg.Application.Name,
				"Version": cfg.Application.Version,
				"Author":  cfg.Application.Author,
			},
			"Title":        "Personnel Records", // Specific page title
			"ListEndpoint": "/list",
			"Limit":        10,
			"Offset":       0,
			"UIColumns":    gridHandler.Columns,
			"LangsJSON":    `["en", "hu"]`,
			"CurrentLang":  "en",
		}
		tmpl.ExecuteTemplate(w, "index.html", data)
	})

	http.HandleFunc("/list", func(w http.ResponseWriter, r *http.Request) {
		params := gridHandler.ParseParams(r)
		result, err := gridHandler.FetchData(params)
		if err != nil {
			log.Printf("FetchData error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "datagrid_table", result)
	})

	fmt.Printf("Server starting at http://localhost:%s\n", cfg.Server.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Server.Port, nil))
}
