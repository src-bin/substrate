package table

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Ftable writes the given cells (presumed to be in row-major order and with
// rows of equal length) to the given io.Writer in a layout suitable for
// terminals or plaintext files.
func Ftable(w io.Writer, cells [][]string) {
	if len(cells) == 0 {
		return
	}

	widths := make([]int, len(cells[0]))
	for _, row := range cells {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	delim, format := "+", "|"
	for _, width := range widths {
		delim += strings.Repeat("-", width+2) + "+"
		format += fmt.Sprintf(" %%-%ds |", width)
	}
	delim += "\n"
	format += "\n"

	fmt.Fprint(w, delim)
	for i, row := range cells {
		args := make([]interface{}, len(row))
		for i := 0; i < len(row); i++ {
			args[i] = row[i]
		}
		fmt.Fprintf(w, format, args...)
		if i == 0 {
			fmt.Fprint(w, delim)
		}
	}
	fmt.Fprint(w, delim)
}

// MakeCells allocates a slice of slices that can be filled in any then
// passed to Table or Ftable for printing or writing.
func MakeCells(width, height int) [][]string {
	cells := make([][]string, height)
	for i := 0; i < height; i++ {
		cells[i] = make([]string, width)
	}
	return cells
}

// Print writes the given cells (presumed to be in row-major order and with
// rows of equal length) to standard output in a layout suitable for terminals
// or plaintext files.
func Print(cells [][]string) {
	Ftable(os.Stdout, cells)
}
