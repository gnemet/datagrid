package datagrid

import (
	"fmt"
	"io"
)

func (h *Handler) StreamCSV(w io.Writer, p RequestParams) error {
	// 1. Generate Hybrid SQL
	query, configJSON, err := h.BuildGridSQL(p)
	if err != nil {
		return err
	}

	rows, err := h.DB.Query("SELECT datagrid.datagrid_execute_csv($1, $2)", query, configJSON)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err == nil {
			fmt.Fprintln(w, line)
		}
	}
	return nil
}
