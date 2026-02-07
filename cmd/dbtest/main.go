package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
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

	tmpl = template.Must(template.New("ui").Funcs(funcMap).Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>CursorPool UI Test (Catalog Mode)</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css">
    <style>
        body { font-family: sans-serif; padding: 20px; background: #f4f4f9; }
        table { border-collapse: collapse; width: 100%; margin-top: 20px; background: white; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; white-space: nowrap; }
        .controls { margin-bottom: 20px; }
        button { padding: 10px 20px; cursor: pointer; }
        .icon { margin-right: 5px; color: #555; }
    </style>
</head>
<body>
    <h1>Catalog-Driven CursorPool (Mode: {{.Mode}})</h1>
    <div class="controls">
        <button hx-get="/data?page=FIRST" hx-target="#table-body">FIRST</button>
        <button hx-get="/data?page=PRIOR" hx-target="#table-body">PRIOR</button>
        <button hx-get="/data?page=NEXT" hx-target="#table-body">NEXT</button>
        <button hx-get="/data?page=LAST" hx-target="#table-body">LAST</button>
    </div>
    <div id="table-container">
        <table>
            <thead>
                <tr>
                    {{range .DisplayCols}}
                    <th>{{with getIcon .}}<i class="fa {{.}} icon"></i>{{end}}{{getLabel .}}</th>
                    {{end}}
                </tr>
            </thead>
            <tbody id="table-body" hx-get="/data?page=FIRST" hx-trigger="load">
            </tbody>
        </table>
    </div>
</body>
</html>
`))

	tmpl = template.Must(tmpl.New("table").Funcs(funcMap).Parse(`
{{$cols := .Cols}}
{{range .Rows}}
<tr>
    {{$row := .}}
    {{range $cols}}
    <td>{{index $row .}}</td>
    {{end}}
</tr>
{{else}}
<tr><td colspan="10">No records found</td></tr>
{{end}}
`))

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
		})
	})

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		direction := r.URL.Query().Get("page")
		if direction == "" {
			direction = "NEXT"
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
			results, err = pool.FetchPage(context.Background(), sid, direction, 10)
		} else {
			results, err = pool.QueryDirect(context.Background(), query+" LIMIT 10")
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
