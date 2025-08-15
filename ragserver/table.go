package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type Table struct {
	Title string
	Rows  []TableRow
}

type TableRow struct {
	Name        string
	Unit        string
	YearNumbers []YearNumber
}

type YearNumber struct {
	Year   int
	Number Number
}

type Number struct {
	Value float64
	Valid bool
}

func NewNumber(num float64) Number {
	return Number{num, true}
}

func (n Number) ToString() string {
	if !n.Valid {
		return "N/A"
	}
	if n.Value == float64(int(n.Value)) {
		return strconv.Itoa(int(n.Value))
	}
	return fmt.Sprintf("%.2f", n.Value)
}

func (t Table) ToContexts() []string {
	if len(t.Rows) == 0 {
		return nil
	}
	contexts := make([]string, 0, len(t.Rows)*len(t.Rows[0].YearNumbers))
	for _, aRow := range t.Rows {
		for _, yearNumber := range aRow.YearNumbers {
			aContext := fmt.Sprintf(
				"%s: %s for year %d is %s %s",
				t.Title,
				aRow.Name,
				yearNumber.Year,
				yearNumber.Number.ToString(),
				aRow.Unit,
			)
			contexts = append(contexts, aContext)
		}
	}
	return contexts
}

func NewTable(text string) (Table, error) {
	var (
		table Table
		years = []int{}
		lines = strings.Split(text, "\n")
		row   int
	)
	for i := 0; i < len(lines); i++ {
		if i == 0 {
			continue
		}
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if row == 0 {
			// Once we encounter Unit, it usually means a table header second column (first columns is empty)
			// Table title should be on the previous line. However, there could be a footnote there as well,
			// so we need to check for that and skip it if it is a footnote.
			if strings.ToLower(line) == "unit" {
				for j := i - 1; j > 0; j-- {
					_, ok := isNumber(lines[j])
					if ok {
						continue
					}
					table.Title = strings.TrimSpace(lines[j])
					break
				}
				if table.Title == "" {
					return Table{}, fmt.Errorf("could not extract table title")
				}
				continue
			}

			// We haven't found the table yet so keep skipping until we get to Unit and set title from previous line
			if table.Title == "" {
				continue
			}

			year, ok := isNumber(line)
			if !ok && len(years) == 0 {
				return Table{}, fmt.Errorf("could not extract years from table header")
			}
			if ok && year.Value < 1900 {
				// This is probably a footnote, skip
				continue
			}
			if ok {
				years = append(years, int(year.Value))
				continue
			}
			row += 1
			continue
		}

		// When we get there, we would have skipped first cell of the row, so we can assume that
		// the previous line is a category name and current line is a unit.
		// However there is an edge case where category name could be split over 2+ lines so we
		// need to check for that first as well as for possible footnote after multi line name.
		if len(table.Rows) < row {
			aRow := TableRow{
				Name:        strings.TrimSpace(lines[i-1]),
				YearNumbers: make([]YearNumber, 0, len(years)),
			}
			if len(lines) <= i-1 {
				return Table{}, fmt.Errorf("could not complete table row")
			}
			// Look at maximum of 3 more lines
			var (
				consecutiveNumbers int
				footnoteIdx        = -1
			)
			if len(lines) <= i-1+5 {
				continue
			}
			for j := 0; j < 5; j++ {
				_, ok := isNumberOrNotAvailable(lines[i+j])
				if !ok {
					if consecutiveNumbers == 1 && footnoteIdx == -1 {
						footnoteIdx = i + j - 1
					}
					consecutiveNumbers = 0
					continue
				}
				consecutiveNumbers += 1

				if consecutiveNumbers == 2 {
					if footnoteIdx > -1 {
						for k := i; k < footnoteIdx; k++ {
							aRow.Name += " " + strings.TrimSpace(lines[k])
						}
					} else {
						for k := i; k < i+j-2; k++ {
							aRow.Name += " " + strings.TrimSpace(lines[k])
						}
					}

					aRow.Unit = strings.TrimSpace(lines[i+j-2])
					i += j - 2
					break
				}
			}
			table.Rows = append(table.Rows, aRow)
			continue
		}

		// In case category name was split over two lines, we might need to set the unit here
		if table.Rows[row-1].Unit == "" {
			table.Rows[row-1].Unit = line
			continue
		}

		number, ok := isNumberOrNotAvailable(line)
		if ok {
			var year int
			if len(table.Rows[row-1].YearNumbers) < len(years) {
				year = years[len(table.Rows[row-1].YearNumbers)]
			}
			table.Rows[row-1].YearNumbers = append(table.Rows[row-1].YearNumbers, YearNumber{
				Year:   year,
				Number: number,
			})

			if len(table.Rows[row-1].YearNumbers) > len(years) {
				// There could be a footnote after the table if it's end of the page
				// In this case, I will assume the smallest value is a footnote and remove it
				smallestIdx := 0
				for i := 1; i < len(table.Rows[row-1].YearNumbers); i++ {
					if table.Rows[row-1].YearNumbers[i].Number.Value < table.Rows[row-1].YearNumbers[smallestIdx].Number.Value {
						smallestIdx = i
					}
				}
				table.Rows[row-1].YearNumbers = append(
					table.Rows[row-1].YearNumbers[0:smallestIdx],
					table.Rows[row-1].YearNumbers[smallestIdx+1:]...,
				)
				for idx := range table.Rows[row-1].YearNumbers {
					table.Rows[row-1].YearNumbers[idx].Year = years[idx]
				}
			}
			continue
		}

		if len(table.Rows[row-1].YearNumbers) == 0 {
			table.Rows = table.Rows[0 : len(table.Rows)-1]
			// This probably means end of the table
			return table, nil
		}

		row += 1
	}
	return table, nil
}

var (
	numRe, _ = regexp.Compile(`^(\d*\.?\d+|\d{1,3}(,\d{3})*(\.\d+)?)$`)
)

func isNumberOrNotAvailable(s string) (Number, bool) {
	if isNotAvailable(s) {
		return Number{}, true
	}
	return isNumber(s)
}

func isNotAvailable(s string) bool {
	s = strings.TrimSpace(s)
	if s == "-—" || s == "—" || s == "-" || strings.ToLower(s) == "n/a" || s == "" {
		return true
	}
	return false
}

func isNumber(s string) (Number, bool) {
	s = strings.TrimSpace(s)
	if numRe.MatchString(s) {
		value, err := strconv.ParseFloat(strings.Replace(s, ",", "", -1), 64)
		if err == nil {
			return Number{value, true}, true
		}
	}

	s = strings.Replace(s, "*", "", 1)           // Remove any trailing asterisks, for example 2020*
	s = strings.Replace(s, ",", "", 1)           // ,14 -> 14 (in case of parsing multiple footnotes, for example 12\n,14)
	s = strings.Replace(s, " (baseline)", "", 1) // 2019 (baseline) -> 2019 (to parse as a valid year)

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Number{}, false
	}
	return Number{value, true}, true
}
