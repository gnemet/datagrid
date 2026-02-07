package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gnemet/datagrid/database/cursorpool"
	"github.com/joho/godotenv"
)

// Catalog structures (simplified)
type CatalogColumn struct {
	Visible bool              `json:"visible"`
	Icon    string            `json:"icon"`
	Labels  map[string]string `json:"labels"`
}

type CatalogObject struct {
	Name    string `json:"name"`
	Columns []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"columns"`
}

type Catalog struct {
	Objects  []CatalogObject `json:"objects"`
	Datagrid struct {
		Defaults struct {
			PageSize []int `json:"page_size"`
		} `json:"defaults"`
		Columns map[string]CatalogColumn `json:"columns"`
	} `json:"datagrid"`
}

var (
	pool    *cursorpool.CursorPool
	tmpl    *template.Template
	catalog Catalog
	dbMode  string
)

func main() {
	_ = godotenv.Load()

	dbMode = os.Getenv("DB_MODE")
	if dbMode == "" {
		dbMode = cursorpool.ModeConnectless
	}

	// Load Catalog
	catData, err := os.ReadFile("catalog/personnel.json")
	if err != nil {
		log.Fatalf("Failed to read catalog: %v", err)
	}
	if err := json.Unmarshal(catData, &catalog); err != nil {
		log.Fatalf("Failed to unmarshal catalog: %v", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=datagrid,public",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	pool, err = cursorpool.NewCursorPool(connStr, 10, 5*time.Minute, 1*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Helper to get label or fallback to name
	getLabel := func(colName string) string {
		if c, ok := catalog.Datagrid.Columns[colName]; ok {
			if l, ok := c.Labels["en"]; ok {
				return l
			}
		}
		return colName
	}

	// Helper to get icon
	getIcon := func(colName string) string {
		if c, ok := catalog.Datagrid.Columns[colName]; ok {
			return c.Icon
		}
		return ""
	}

	// Load templates
	funcMap := template.FuncMap{
		"getLabel":      getLabel,
		"getIcon":       getIcon,
		"isConnectless": func() bool { return dbMode == cursorpool.ModeConnectless },
	}

	tmpl = template.Must(template.New("main").Funcs(funcMap).ParseGlob("pkg/datagrid/ui/templates/partials/datagrid/*.html"))
	tmpl = template.Must(tmpl.ParseFiles("pkg/datagrid/ui/templates/index.html"))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("pkg/datagrid/ui/static"))))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		displayCols := []string{}
		for _, col := range catalog.Objects[0].Columns {
			if c, ok := catalog.Datagrid.Columns[col.Name]; ok && c.Visible {
				displayCols = append(displayCols, col.Name)
			}
		}
		tmpl.ExecuteTemplate(w, "ui", map[string]interface{}{
			"Mode":        dbMode,
			"DisplayCols": displayCols,
			"PageSizes":   catalog.Datagrid.Defaults.PageSize,
		})
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		direction := r.URL.Query().Get("page")
		if direction == "" {
			direction = "NEXT"
		}

		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}

		sid := "catalog-ui-test"

		// Build query dynamically from catalog
		table := catalog.Objects[0].Name
		colNames := []string{}
		displayCols := []string{}
		for _, col := range catalog.Objects[0].Columns {
			colNames = append(colNames, col.Name)
			if c, ok := catalog.Datagrid.Columns[col.Name]; ok && c.Visible {
				displayCols = append(displayCols, col.Name)
			}
		}
		query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(colNames, ", "), table)

		var results []map[string]interface{}
		var err error

		if dbMode == cursorpool.ModeRefCursor {
			if _, err := pool.InitializeCursor(context.Background(), sid, query); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			results, err = pool.FetchPage(context.Background(), sid, direction, limit)
		} else {
			results, err = pool.QueryDirect(context.Background(), fmt.Sprintf("%s LIMIT %d", query, limit))
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		tmpl.ExecuteTemplate(w, "table", map[string]interface{}{
			"Rows": results,
			"Cols": displayCols,
		})
	})

	port := os.Getenv("DBTEST_PORT")
	if port == "" {
		port = "8090"
	}
	fmt.Printf("Catalog UI Test Server starting at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
