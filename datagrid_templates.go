package datagrid

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"strings"
	tt "text/template"
)

// TemplateFuncs returns a map of standard datagrid template functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"renderRow": RenderRow,
		"replace":   strings.ReplaceAll,
		"contains":  strings.Contains,
		"to_lower":  strings.ToLower,
		"T":         func(s string) string { return s },
		"buildLink": BuildLink,

		"sub": func(a, b int) int { return a - b },
		"add": func(a, b int) int { return a + b },
		"formatNum": func(v interface{}) string {
			switch val := v.(type) {
			case float64:
				if val == 0 {
					return "0"
				}
				if val > 1000000 {
					return fmt.Sprintf("%.2fM", val/1000000)
				}
				if val > 1000 {
					return fmt.Sprintf("%.2fk", val/1000)
				}
				return fmt.Sprintf("%.2f", val)
			case int, int64:
				return fmt.Sprintf("%d", val)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
		// Query parameter helpers
		"inputType":    func(p QueryParam) string { return p.InputType() },
		"constantKey":  func(p QueryParam) string { return p.ConstantKey() },
		"displayLabel": func(p QueryParam) string { return p.DisplayLabel() },

		// lov-tree: indent label by depth
		"indentLabel": func(item LOVItem) string {
			prefix := ""
			for j := 0; j < item.Depth; j++ {
				prefix += "── "
			}
			return prefix + item.Label
		},

		// lov-grouped: ordered unique group labels
		"lovGroups": func(items []LOVItem) []string {
			seen := map[string]bool{}
			var groups []string
			for _, item := range items {
				if !seen[item.Group] {
					seen[item.Group] = true
					groups = append(groups, item.Group)
				}
			}
			return groups
		},

		// lov-grouped: items within a specific group
		"lovByGroup": func(items []LOVItem, group string) []LOVItem {
			var result []LOVItem
			for _, item := range items {
				if item.Group == group {
					result = append(result, item)
				}
			}
			return result
		},

		// pivot2: flatten tree for template rendering
		"flattenTree": FlattenTree,

		// toJSON serializes a value to JSON string for data attributes
		"toJSON": func(v interface{}) string {
			b, err := json.Marshal(v)
			if err != nil {
				return "{}"
			}
			return string(b)
		},
	}
}
// quote_ident is a helper for SQL templates to prevent injection
func quote_ident(s string) string {
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		for i, p := range parts {
			parts[i] = `"` + strings.ReplaceAll(p, `"`, `""`) + `"`
		}
		return strings.Join(parts, ".")
	}
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
// renderSQL renders a Go text/template for SQL generation
func (h *Handler) renderSQL(tmplName string, data interface{}) (string, error) {
	// For simplicity in this demo, we'll read from disk.
	tmplPath := fmt.Sprintf("internal/sql/templates/%s.tmpl", tmplName)
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		return "", err
	}

	funcMap := template.FuncMap{
		"add":         func(a, b int) int { return a + b },
		"quote_ident": quote_ident,
		"coalesce": func(s ...string) string {
			for _, v := range s {
				if v != "" {
					return v
				}
			}
			return ""
		},
	}

	tmpl, err := tt.New(tmplName).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", err
	}

	result := sb.String()
	// Automatically cast $1 to jsonb for Postgres type inference
	result = strings.ReplaceAll(result, "$1", "$1::jsonb")

	slog.Debug("Rendered SQL template", "template", tmplName, "sql", result)
	return result, nil
}
// RenderRow replaces %field% placeholders in a pattern with values from the row
func RenderRow(pattern string, row map[string]interface{}) template.HTML {
	result := pattern
	for k, v := range row {
		placeholder := fmt.Sprintf("%%%s%%", k)
		valStr := fmt.Sprintf("%v", v)
		result = strings.ReplaceAll(result, placeholder, valStr)
	}
	return template.HTML(result)
}
func isNumericType(t string) bool {
	t = strings.ToLower(t)
	return strings.Contains(t, "int") || strings.Contains(t, "num") || strings.Contains(t, "float") ||
		strings.Contains(t, "double") || strings.Contains(t, "decimal") || strings.Contains(t, "money")
}

// BuildLink constructs a hyperlink URL based on a pattern and row data.
// It resolves {value} to the current cell value and {column_name} to row map values.
// If linkPattern matches a key in globalLinks, that global pattern is used instead.
// The row parameter accepts both map[string]interface{} and map[string]string.
func BuildLink(linkPattern string, val interface{}, row interface{}, globalLinks map[string]string) string {
	if linkPattern == "" {
		return "#"
	}

	pattern := linkPattern
	if globalPattern, ok := globalLinks[linkPattern]; ok && globalPattern != "" {
		pattern = globalPattern
	}

	result := pattern

	// Replace {value} placeholder
	valStr := fmt.Sprintf("%v", val)
	result = strings.ReplaceAll(result, "{value}", valStr)

	// Replace {column_name} placeholders — support both map types
	switch r := row.(type) {
	case map[string]interface{}:
		for k, v := range r {
			placeholder := fmt.Sprintf("{%s}", k)
			if strings.Contains(result, placeholder) {
				result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
			}
		}
	case map[string]string:
		for k, v := range r {
			placeholder := fmt.Sprintf("{%s}", k)
			if strings.Contains(result, placeholder) {
				result = strings.ReplaceAll(result, placeholder, v)
			}
		}
	}

	// Second pass: resolve any remaining {key} placeholders from globalLinks
	// This handles indirect references, e.g. link="{url_src}{value}" where
	// url_src="{jira}" → after row substitution the result contains "{jira}"
	// which must be resolved from globalLinks["jira"] → "https://jira.url/browse/"
	for k, v := range globalLinks {
		placeholder := fmt.Sprintf("{%s}", k)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, v)
		}
	}

	return result
}
