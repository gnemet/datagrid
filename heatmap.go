package datagrid

import (
	"fmt"
	"html/template"
	"io"
	"math"
	"sort"
)

// HeatmapConfig defines the heatmap visualization configuration.
// Parsed from the ## HEATMAP YAML block in a BI report .md file.
type HeatmapConfig struct {
	Rows        string `json:"rows" yaml:"rows"`                                   // SQL column for row labels (category axis)
	Columns     string `json:"columns" yaml:"columns"`                             // SQL column for column labels (time axis)
	Value       string `json:"value" yaml:"value"`                                 // SQL column for the measure
	SortColumns string `json:"sort_columns,omitempty" yaml:"sort_columns"`         // SQL column for column ordering
	ColorScale  string `json:"color_scale,omitempty" yaml:"color_scale"`           // green, blue, red, orange (default: green)
	ShowValues  bool   `json:"show_values,omitempty" yaml:"show_values"`           // Overlay numeric values on cells
	EmptyColor  string `json:"empty_color,omitempty" yaml:"empty_color"`           // CSS color for zero/null cells
	Format      string `json:"format,omitempty" yaml:"format"`                     // Printf format for values (e.g. "%.1f")
	CellTooltip string `json:"cell_tooltip,omitempty" yaml:"cell_tooltip"`         // SQL column for tooltip text
	RowLink     string `json:"row_link,omitempty" yaml:"row_link"`                 // Link pattern for row labels
	Links       map[string]string `json:"links,omitempty" yaml:"links,omitempty"`  // Global link overrides
}

// HeatmapCell represents a single cell in the heatmap grid.
type HeatmapCell struct {
	Value     float64
	Formatted string
	Intensity float64 // 0.0–1.0 normalized intensity
	Tooltip   string
	HasData   bool
}

// HeatmapRow represents a row in the heatmap.
type HeatmapRow struct {
	Label string
	Link  string
	Cells []HeatmapCell
}

// HeatmapResult holds the complete heatmap data for template rendering.
type HeatmapResult struct {
	ColumnLabels []string
	Rows         []HeatmapRow
	MinValue     float64
	MaxValue     float64
	ColorScale   string
	ShowValues   bool
	EmptyColor   string
	TotalRows    int
	TotalCols    int
}

// heatmapColorScale defines the HSL hue for each named color scale.
var heatmapColorScale = map[string]int{
	"green":  142,
	"blue":   217,
	"red":    0,
	"orange": 30,
	"purple": 270,
	"cyan":   180,
}

// HeatmapData builds a HeatmapResult from flat SQL records and a HeatmapConfig.
func HeatmapData(records []map[string]interface{}, cfg *HeatmapConfig) *HeatmapResult {
	if cfg == nil || len(records) == 0 {
		return &HeatmapResult{ColorScale: "green"}
	}

	// 1. Collect unique row labels (ordered by first appearance)
	rowOrder := []string{}
	rowSeen := map[string]bool{}

	// 2. Collect unique column labels with optional sort keys
	type colEntry struct {
		label   string
		sortKey float64
	}
	colMap := map[string]colEntry{}

	// 3. Build a value map: rowLabel → colLabel → value
	valueMap := map[string]map[string]float64{}
	tooltipMap := map[string]map[string]string{}

	for _, rec := range records {
		rowLabel := fmt.Sprintf("%v", rec[cfg.Rows])
		colLabel := fmt.Sprintf("%v", rec[cfg.Columns])

		if !rowSeen[rowLabel] {
			rowSeen[rowLabel] = true
			rowOrder = append(rowOrder, rowLabel)
		}

		if _, exists := colMap[colLabel]; !exists {
			sortKey := 0.0
			if cfg.SortColumns != "" {
				sortKey = extractFloat(rec, cfg.SortColumns)
			}
			colMap[colLabel] = colEntry{label: colLabel, sortKey: sortKey}
		}

		if valueMap[rowLabel] == nil {
			valueMap[rowLabel] = make(map[string]float64)
		}
		valueMap[rowLabel][colLabel] = extractFloat(rec, cfg.Value)

		if cfg.CellTooltip != "" {
			if tooltipMap[rowLabel] == nil {
				tooltipMap[rowLabel] = make(map[string]string)
			}
			if v, ok := rec[cfg.CellTooltip]; ok && v != nil {
				tooltipMap[rowLabel][colLabel] = fmt.Sprintf("%v", v)
			}
		}
	}

	// Sort columns
	colEntries := make([]colEntry, 0, len(colMap))
	for _, e := range colMap {
		colEntries = append(colEntries, e)
	}
	if cfg.SortColumns != "" {
		sort.Slice(colEntries, func(i, j int) bool {
			return colEntries[i].sortKey < colEntries[j].sortKey
		})
	} else {
		sort.Slice(colEntries, func(i, j int) bool {
			return colEntries[i].label < colEntries[j].label
		})
	}
	colLabels := make([]string, len(colEntries))
	for i, e := range colEntries {
		colLabels[i] = e.label
	}

	// 4. Find min/max for normalization
	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64
	hasAny := false
	for _, cols := range valueMap {
		for _, v := range cols {
			hasAny = true
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if !hasAny {
		minVal = 0
		maxVal = 0
	}

	// 5. Build result rows
	format := cfg.Format
	if format == "" {
		format = "%.1f"
	}

	colorScale := cfg.ColorScale
	if colorScale == "" {
		colorScale = "green"
	}

	rows := make([]HeatmapRow, len(rowOrder))
	for i, rowLabel := range rowOrder {
		cells := make([]HeatmapCell, len(colLabels))
		for j, colLabel := range colLabels {
			val, hasData := valueMap[rowLabel][colLabel]
			intensity := 0.0
			if hasData && maxVal > minVal {
				intensity = (val - minVal) / (maxVal - minVal)
			} else if hasData && maxVal == minVal && maxVal > 0 {
				intensity = 1.0
			}

			tooltip := ""
			if cfg.CellTooltip != "" {
				if tm, ok := tooltipMap[rowLabel]; ok {
					tooltip = tm[colLabel]
				}
			}
			if tooltip == "" {
				tooltip = fmt.Sprintf("%s · %s: "+format, rowLabel, colLabel, val)
			}

			cells[j] = HeatmapCell{
				Value:     val,
				Formatted: fmt.Sprintf(format, val),
				Intensity: intensity,
				Tooltip:   tooltip,
				HasData:   hasData,
			}
		}
		rows[i] = HeatmapRow{
			Label: rowLabel,
			Cells: cells,
		}
	}

	return &HeatmapResult{
		ColumnLabels: colLabels,
		Rows:         rows,
		MinValue:     minVal,
		MaxValue:     maxVal,
		ColorScale:   colorScale,
		ShowValues:   cfg.ShowValues,
		EmptyColor:   cfg.EmptyColor,
		TotalRows:    len(rows),
		TotalCols:    len(colLabels),
	}
}

// RenderHeatmap renders a heatmap visualization from pre-fetched records.
// This is the public API for external callers (e.g. jiramntr's BIQueryExecuteHandler).
// It writes the rendered HTML directly to w.
func RenderHeatmap(w io.Writer, records []map[string]interface{}, cfg *HeatmapConfig, title string, lang string) error {
	result := HeatmapData(records, cfg)

	funcs := TemplateFuncs()

	// Add heatmap-specific template functions
	funcs["heatmapCellStyle"] = func(cell HeatmapCell, scale string, emptyColor string) template.CSS {
		if !cell.HasData || cell.Value == 0 {
			if emptyColor != "" {
				return template.CSS("background-color: " + emptyColor)
			}
			return template.CSS("background-color: var(--hm-empty, rgba(255,255,255,0.03))")
		}

		hue := 142 // default green
		if h, ok := heatmapColorScale[scale]; ok {
			hue = h
		}

		// Interpolate: low intensity → dark/dim, high intensity → bright/saturated
		sat := 50 + cell.Intensity*30       // 50% → 80%
		light := 12 + cell.Intensity*28     // 12% → 40%
		alpha := 0.3 + cell.Intensity*0.7   // 0.3 → 1.0

		return template.CSS(fmt.Sprintf(
			"background-color: hsla(%d, %.0f%%, %.0f%%, %.2f)",
			hue, sat, light, alpha,
		))
	}

	tmpl, err := template.New("heatmap_standalone").Funcs(funcs).ParseFS(UIAssets,
		"ui/templates/partials/datagrid/heatmap.html",
	)
	if err != nil {
		return fmt.Errorf("heatmap template parse error: %w", err)
	}

	data := map[string]interface{}{
		"Title":         title,
		"HeatmapResult": result,
		"Lang":          lang,
	}

	return tmpl.ExecuteTemplate(w, "datagrid_heatmap", data)
}
