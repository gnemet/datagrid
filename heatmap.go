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
	Rows        string            `json:"rows" yaml:"rows"`                             // SQL column for row labels (category axis)
	Columns     string            `json:"columns" yaml:"columns"`                       // SQL column for column labels (time axis)
	Value       string            `json:"value" yaml:"value"`                           // SQL column for the measure
	SortColumns string            `json:"sort_columns,omitempty" yaml:"sort_columns"`   // SQL column for column ordering
	SortRows    string            `json:"sort_rows,omitempty" yaml:"sort_rows"`         // "asc", "desc" (by row total), or empty (first-appearance)
	ColorMode   string            `json:"color_mode,omitempty" yaml:"color_mode"`       // "fixed" or "dynamic" (default: dynamic)
	ColorScale  string            `json:"color_scale,omitempty" yaml:"color_scale"`     // For dynamic: green, blue, red, orange, kpi, diverging
	Midpoint    *float64          `json:"midpoint,omitempty" yaml:"midpoint"`           // For diverging scale: center value
	ShowValues  bool              `json:"show_values,omitempty" yaml:"show_values"`     // Overlay numeric values on cells
	ShowTotals  bool              `json:"show_totals,omitempty" yaml:"show_totals"`     // Show row/column totals
	EmptyColor  string            `json:"empty_color,omitempty" yaml:"empty_color"`     // CSS color for null cells
	Format      string            `json:"format,omitempty" yaml:"format"`               // Printf format for values (e.g. "%.1f")
	CellTooltip string            `json:"cell_tooltip,omitempty" yaml:"cell_tooltip"`   // SQL column for tooltip text
	RowLink     string            `json:"row_link,omitempty" yaml:"row_link"`           // Link pattern for row labels
	RatingScale []HeatmapRating   `json:"rating_scale,omitempty" yaml:"rating_scale"`   // For fixed mode: threshold ranges
	Links       map[string]string `json:"links,omitempty" yaml:"links,omitempty"`       // Global link overrides
}

// HeatmapRating defines a color threshold range for fixed color mode.
// Compatible with KPI scorecard rating_scale format.
type HeatmapRating struct {
	Min   float64 `json:"min" yaml:"min"`
	Max   float64 `json:"max" yaml:"max"`
	Color string  `json:"color" yaml:"color"` // Hex color (e.g. "#3A7D66")
	Label string  `json:"label" yaml:"label"` // Legend label (e.g. "Excellent")
}

// HeatmapCell represents a single cell in the heatmap grid.
type HeatmapCell struct {
	Value     float64
	Formatted string
	Intensity float64 // 0.0–1.0 normalized intensity (for dynamic mode)
	CSSColor  string  // Pre-computed CSS color string
	Tooltip   string
	HasData   bool    // true if the cell has a SQL record
	IsNull    bool    // true if value column was NULL (distinct from zero)
}

// HeatmapRow represents a row in the heatmap.
type HeatmapRow struct {
	Label string
	Link  string
	Cells []HeatmapCell
	Total float64
	TotalFormatted string
}

// HeatmapLegendItem is a single entry in the legend.
type HeatmapLegendItem struct {
	Color string
	Label string
}

// HeatmapResult holds the complete heatmap data for template rendering.
type HeatmapResult struct {
	ColumnLabels   []string
	Rows           []HeatmapRow
	ColumnTotals   []float64
	ColumnTotalsFmt []string
	MinValue       float64
	MaxValue       float64
	ColorMode      string // "fixed" or "dynamic"
	ColorScale     string
	ShowValues     bool
	ShowTotals     bool
	EmptyColor     string
	TotalRows      int
	TotalCols      int
	// Legend data
	LegendItems    []HeatmapLegendItem // For fixed mode
	LegendGradient string              // For dynamic mode: CSS gradient string
	LegendMinLabel string
	LegendMaxLabel string
}

// ── KPI Palette (matches base.css) ──

var kpiPaletteStops = []struct {
	pos   float64 // 0.0–1.0
	r, g, b uint8
}{
	{0.0, 192, 87, 70},   // #C05746 — critical
	{0.25, 220, 158, 98},  // #DC9E62 — warning
	{0.5, 227, 216, 186},  // #E3D8BA — neutral
	{0.75, 138, 176, 142}, // #8AB08E — good
	{1.0, 58, 125, 102},   // #3A7D66 — exceptional
}

// ── Single-hue gradient presets ──

var heatmapHueMap = map[string]int{
	"green":  142,
	"blue":   217,
	"red":    0,
	"orange": 30,
	"purple": 270,
	"cyan":   180,
}

// ── Color helpers ──

// heatmapColorFixed returns a CSS color for a value using fixed rating scale thresholds.
func heatmapColorFixed(value float64, ratings []HeatmapRating) string {
	for _, r := range ratings {
		if value >= r.Min && value <= r.Max {
			return r.Color
		}
	}
	// Fallback: last rating or transparent
	if len(ratings) > 0 {
		return ratings[len(ratings)-1].Color
	}
	return "transparent"
}

// heatmapColorDynamic returns a CSS color for a normalized intensity (0–1) using named presets.
func heatmapColorDynamic(intensity float64, scale string) string {
	if scale == "kpi" {
		return interpolateKPIPalette(intensity)
	}

	hue := 142 // default green
	if h, ok := heatmapHueMap[scale]; ok {
		hue = h
	}

	// Smooth interpolation: dark/dim → bright/saturated
	sat := 50 + intensity*30   // 50% → 80%
	light := 12 + intensity*28 // 12% → 40%
	alpha := 0.3 + intensity*0.7

	return fmt.Sprintf("hsla(%d, %.0f%%, %.0f%%, %.2f)", hue, sat, light, alpha)
}

// heatmapColorDiverging returns a CSS color for a value around a midpoint.
// Below midpoint → red, at midpoint → neutral, above midpoint → green.
func heatmapColorDiverging(value, minVal, maxVal, midpoint float64) string {
	if value <= midpoint {
		// Red side: map [minVal, midpoint] → [0, 1] (intensity of red)
		rng := midpoint - minVal
		if rng <= 0 {
			return interpolateKPIPalette(0.5)
		}
		t := (midpoint - value) / rng // 0 = at midpoint, 1 = at min
		// Map to palette: midpoint=0.5 → min=0.0
		return interpolateKPIPalette(0.5 - t*0.5)
	}
	// Green side: map [midpoint, maxVal] → [0, 1] (intensity of green)
	rng := maxVal - midpoint
	if rng <= 0 {
		return interpolateKPIPalette(0.5)
	}
	t := (value - midpoint) / rng
	return interpolateKPIPalette(0.5 + t*0.5)
}

// interpolateKPIPalette smoothly interpolates through the KPI color stops at position t (0–1).
func interpolateKPIPalette(t float64) string {
	if t <= 0 {
		s := kpiPaletteStops[0]
		return fmt.Sprintf("rgb(%d, %d, %d)", s.r, s.g, s.b)
	}
	if t >= 1 {
		s := kpiPaletteStops[len(kpiPaletteStops)-1]
		return fmt.Sprintf("rgb(%d, %d, %d)", s.r, s.g, s.b)
	}

	// Find the two surrounding stops
	for i := 1; i < len(kpiPaletteStops); i++ {
		if t <= kpiPaletteStops[i].pos {
			s0 := kpiPaletteStops[i-1]
			s1 := kpiPaletteStops[i]
			localT := (t - s0.pos) / (s1.pos - s0.pos)
			r := float64(s0.r) + localT*(float64(s1.r)-float64(s0.r))
			g := float64(s0.g) + localT*(float64(s1.g)-float64(s0.g))
			b := float64(s0.b) + localT*(float64(s1.b)-float64(s0.b))
			return fmt.Sprintf("rgb(%.0f, %.0f, %.0f)", r, g, b)
		}
	}
	s := kpiPaletteStops[len(kpiPaletteStops)-1]
	return fmt.Sprintf("rgb(%d, %d, %d)", s.r, s.g, s.b)
}

// buildLegendGradient generates a CSS linear-gradient string for the legend bar.
func buildLegendGradient(scale string) string {
	if scale == "kpi" {
		return "linear-gradient(to right, #C05746, #DC9E62, #E3D8BA, #8AB08E, #3A7D66)"
	}
	if scale == "diverging" {
		return "linear-gradient(to right, #C05746, #E3D8BA, #3A7D66)"
	}
	hue := 142
	if h, ok := heatmapHueMap[scale]; ok {
		hue = h
	}
	return fmt.Sprintf("linear-gradient(to right, hsla(%d, 50%%, 12%%, 0.3), hsla(%d, 80%%, 40%%, 1.0))", hue, hue)
}

// ── Data builder ──

// HeatmapData builds a HeatmapResult from flat SQL records and a HeatmapConfig.
func HeatmapData(records []map[string]interface{}, cfg *HeatmapConfig) *HeatmapResult {
	if cfg == nil || len(records) == 0 {
		return &HeatmapResult{ColorMode: "dynamic", ColorScale: "green"}
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

	// 3. Build value maps
	type cellData struct {
		value  float64
		isNull bool
	}
	valueMap := map[string]map[string]cellData{}
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
			valueMap[rowLabel] = make(map[string]cellData)
		}

		rawVal := rec[cfg.Value]
		if rawVal == nil {
			valueMap[rowLabel][colLabel] = cellData{value: 0, isNull: true}
		} else {
			valueMap[rowLabel][colLabel] = cellData{value: extractFloat(rec, cfg.Value), isNull: false}
		}

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

	// 4. Find min/max for normalization (skip null cells)
	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64
	hasAny := false
	for _, cols := range valueMap {
		for _, cd := range cols {
			if cd.isNull {
				continue
			}
			hasAny = true
			if cd.value < minVal {
				minVal = cd.value
			}
			if cd.value > maxVal {
				maxVal = cd.value
			}
		}
	}
	if !hasAny {
		minVal = 0
		maxVal = 0
	}

	// Resolve config defaults
	format := cfg.Format
	if format == "" {
		format = "%.1f"
	}
	colorScale := cfg.ColorScale
	if colorScale == "" {
		colorScale = "green"
	}
	colorMode := cfg.ColorMode
	if colorMode == "" {
		colorMode = "dynamic"
	}
	isFixed := colorMode == "fixed" && len(cfg.RatingScale) > 0
	isDiverging := colorScale == "diverging"

	midpoint := 0.0
	if cfg.Midpoint != nil {
		midpoint = *cfg.Midpoint
	} else if isDiverging {
		midpoint = (minVal + maxVal) / 2
	}

	emptyCSS := "var(--hm-empty, rgba(255,255,255,0.03))"
	if cfg.EmptyColor != "" {
		emptyCSS = cfg.EmptyColor
	}

	// 5. Compute row totals for optional sorting
	rowTotals := make(map[string]float64)
	for _, rl := range rowOrder {
		total := 0.0
		for _, cl := range colLabels {
			if cd, ok := valueMap[rl][cl]; ok && !cd.isNull {
				total += cd.value
			}
		}
		rowTotals[rl] = total
	}

	// Sort rows if requested
	if cfg.SortRows == "desc" {
		sort.SliceStable(rowOrder, func(i, j int) bool {
			return rowTotals[rowOrder[i]] > rowTotals[rowOrder[j]]
		})
	} else if cfg.SortRows == "asc" {
		sort.SliceStable(rowOrder, func(i, j int) bool {
			return rowTotals[rowOrder[i]] < rowTotals[rowOrder[j]]
		})
	}

	// 6. Build result rows with pre-computed colors
	rows := make([]HeatmapRow, len(rowOrder))
	colTotals := make([]float64, len(colLabels))

	for i, rowLabel := range rowOrder {
		cells := make([]HeatmapCell, len(colLabels))
		for j, colLabel := range colLabels {
			cd, hasData := valueMap[rowLabel][colLabel]

			var cssColor string
			intensity := 0.0

			if !hasData || cd.isNull {
				cssColor = emptyCSS
			} else if cd.value == 0 && !isFixed {
				cssColor = emptyCSS
			} else if isFixed {
				cssColor = heatmapColorFixed(cd.value, cfg.RatingScale)
			} else if isDiverging {
				cssColor = heatmapColorDiverging(cd.value, minVal, maxVal, midpoint)
			} else {
				if maxVal > minVal {
					intensity = (cd.value - minVal) / (maxVal - minVal)
				} else if maxVal == minVal && maxVal > 0 {
					intensity = 1.0
				}
				cssColor = heatmapColorDynamic(intensity, colorScale)
			}

			tooltip := ""
			if cfg.CellTooltip != "" {
				if tm, ok := tooltipMap[rowLabel]; ok {
					tooltip = tm[colLabel]
				}
			}
			if tooltip == "" && hasData && !cd.isNull {
				tooltip = fmt.Sprintf("%s · %s: "+format, rowLabel, colLabel, cd.value)
			}

			formatted := ""
			if hasData && !cd.isNull {
				formatted = fmt.Sprintf(format, cd.value)
			}

			cells[j] = HeatmapCell{
				Value:     cd.value,
				Formatted: formatted,
				Intensity: intensity,
				CSSColor:  cssColor,
				Tooltip:   tooltip,
				HasData:   hasData && !cd.isNull,
				IsNull:    !hasData || cd.isNull,
			}

			if hasData && !cd.isNull {
				colTotals[j] += cd.value
			}
		}

		rows[i] = HeatmapRow{
			Label:          rowLabel,
			Cells:          cells,
			Total:          rowTotals[rowLabel],
			TotalFormatted: fmt.Sprintf(format, rowTotals[rowLabel]),
		}
	}

	// Format column totals
	colTotalsFmt := make([]string, len(colTotals))
	for i, ct := range colTotals {
		colTotalsFmt[i] = fmt.Sprintf(format, ct)
	}

	// 7. Build legend
	var legendItems []HeatmapLegendItem
	legendGradient := ""

	if isFixed {
		// Sort ratings ascending by Min for proper legend order
		sorted := make([]HeatmapRating, len(cfg.RatingScale))
		copy(sorted, cfg.RatingScale)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Min < sorted[j].Min
		})
		for _, r := range sorted {
			legendItems = append(legendItems, HeatmapLegendItem{
				Color: r.Color,
				Label: r.Label,
			})
		}
	} else {
		legendGradient = buildLegendGradient(colorScale)
	}

	return &HeatmapResult{
		ColumnLabels:    colLabels,
		Rows:            rows,
		ColumnTotals:    colTotals,
		ColumnTotalsFmt: colTotalsFmt,
		MinValue:        minVal,
		MaxValue:        maxVal,
		ColorMode:       colorMode,
		ColorScale:      colorScale,
		ShowValues:      cfg.ShowValues,
		ShowTotals:      cfg.ShowTotals,
		EmptyColor:      cfg.EmptyColor,
		TotalRows:       len(rows),
		TotalCols:       len(colLabels),
		LegendItems:     legendItems,
		LegendGradient:  legendGradient,
		LegendMinLabel:  fmt.Sprintf(format, minVal),
		LegendMaxLabel:  fmt.Sprintf(format, maxVal),
	}
}

// RenderHeatmap renders a heatmap visualization from pre-fetched records.
// This is the public API for external callers (e.g. jiramntr's BIQueryExecuteHandler).
// It writes the rendered HTML directly to w.
func RenderHeatmap(w io.Writer, records []map[string]interface{}, cfg *HeatmapConfig, title string, lang string) error {
	result := HeatmapData(records, cfg)

	funcs := TemplateFuncs()

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
