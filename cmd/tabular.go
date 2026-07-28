package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

func renderTable(headers []string, rows [][]any) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	headerCells := make([]any, len(headers))
	for i, header := range headers {
		headerCells[i] = header
	}
	t.AppendHeader(table.Row(headerCells))
	if len(rows) == 0 {
		t.AppendRow(table.Row{"No results"})
	} else {
		for _, row := range rows {
			cells := make([]any, len(row))
			for i, value := range row {
				cells[i] = tabularCell(value)
			}
			t.AppendRow(table.Row(cells))
		}
	}
	t.SetStyle(table.StyleRounded)
	t.Render()
}

func renderKeyValueTable(rows [][2]string) {
	values := make([][]any, 0, len(rows))
	for _, row := range rows {
		values = append(values, []any{row[0], row[1]})
	}
	renderTable([]string{"Field", "Value"}, values)
}

func tabularCell(value any) string {
	text := fmt.Sprint(value)
	return strings.ReplaceAll(strings.ReplaceAll(text, "\r", ""), "\n", "\\n")
}
