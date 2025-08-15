package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Table represents a structured table with one or more years and as header columns
// and rows representing categories with numeric values for each year.
type Table struct {
	Title   string
	Rows    []TableRow
	years   []int
	hasUnit bool
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
	text  string // original text representation
}

func NewNumber(num float64, text string) Number {
	return Number{num, true, text}
}

func (n Number) ValidYear() bool {
	if n.text != "" && strings.Contains(n.text, ",") {
		return false
	}
	return n.Valid && n.Value >= 1900 && n.Value <= 2100
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

func NewTables(text string) ([]*Table, error) {
	return newUnitTable(text)
}

func newUnitTable(text string) ([]*Table, error) {
	var (
		tables = make([]*Table, 0, 1)
		lines  = strings.Split(text, "\n")
		row    int
	)
	for i := 0; i < len(lines); i++ {
		if i == 0 {
			continue
		}
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if len(tables) == 0 {
			tables = append(tables, new(Table))
		}
		table := tables[len(tables)-1]

		if row == 0 {
			// More complex tables will have a unit as a second column followed by columns for multiple years.
			// Table title should be on the previous line. However, there could be a footnote there as well,
			// so we need to check for that and skip it if it is a footnote.
			if strings.ToLower(line) == "unit" && table.Title == "" {
				for j := i - 1; j > 0; j-- {
					_, ok := isNumber(lines[j])
					if ok {
						continue
					}
					table.Title = strings.TrimSpace(lines[j])
					table.hasUnit = true
					break
				}
				if table.Title == "" {
					return nil, fmt.Errorf("could not extract table title")
				}
				continue
			}

			// Simpler table will not have unit and their second column will be a year
			// If we don't have a table title yet, assume it's a previous column
			year, ok := isNumber(line)
			if ok && year.ValidYear() && table.Title == "" {
				for j := i - 1; j > 0; j-- {
					_, ok := isNumber(lines[j])
					if ok {
						continue
					}
					table.Title = strings.TrimSpace(lines[j])
					table.years = append(table.years, int(year.Value))
					break
				}
				if table.Title == "" {
					return nil, fmt.Errorf("could not extract table title")
				}
				continue
			}

			// We haven't found the title yet so keep skipping until we get to unit
			// or first year column and set title from previous line
			if table.Title == "" {
				continue
			}

			// One or more year columns
			year, ok = isNumber(line)
			if ok && !year.ValidYear() {
				// This is probably a footnote, skip
				continue
			}
			if ok && year.ValidYear() {
				table.years = append(table.years, int(year.Value))
				continue
			}

			if len(table.years) == 0 {
				return nil, fmt.Errorf("could not extract years from table header")
			}

			// This means we are on a new row, header row has ended as we have processed last valid year
			processed, err := table.addNewRow(lines, i)
			if err != nil {
				return nil, err
			}
			i += processed
			row += 1
			continue
		}

		number, ok := isNumberOrNotAvailable(line)

		// We want to potentially create a new table here in case number appears to be a year
		// and it is a first number in the row. This probably means this is a second table
		// right under the first one with previous line being its title.
		if ok && number.ValidYear() && len(table.Rows) > 0 && len(table.Rows[row-1].YearNumbers) == 0 {
			tables = append(tables, new(Table))
			table = tables[len(tables)-1]
			table.Title = strings.TrimSpace(lines[i-1])
			table.years = []int{int(number.Value)}
			row = 0
			continue
		}

		// Otherwise we are just adding a new yearly value to our existing row
		if ok {
			var year int
			if len(table.Rows[row-1].YearNumbers) < len(table.years) {
				year = table.years[len(table.Rows[row-1].YearNumbers)]
			}
			table.Rows[row-1].YearNumbers = append(table.Rows[row-1].YearNumbers, YearNumber{
				Year:   year,
				Number: number,
			})

			continue
		}

		if len(table.Rows[row-1].YearNumbers) > len(table.years) {
			table.removeExtraYearlyValues(row)
		}

		if len(table.Rows[row-1].YearNumbers) == len(table.years) {
			processed, err := table.addNewRow(lines, i)
			if err != nil {
				return nil, err
			}

			i += processed
			row += 1
			continue
		}
	}

	for _, table := range tables {
		table.removeExtraRows()
	}

	return tables, nil
}

type orderedNumber struct {
	Number
	idx int
}

func (t *Table) removeExtraYearlyValues(row int) {
	// There could be a footnote after the table if it's end of the page.
	// Or there could be a footnote after a unit/title before yearly values begin.
	// There isn't a deterministic way to know which of the values we have captured
	// is a footnote. Footnotes are usually small numbers, up to mid double digits.
	// If out of all values all presumed yearly values are at least >50 as extra values,
	// we will asuume lowest extra values are footnotes and delete them.
	// Otherwise, we will assume footnotes are leftmost extra elements and remove those.
	// This is not a bulletproof logic, but it will work in good amount of cases.
	var (
		values                    = make([]orderedNumber, 0, len(t.Rows[row-1].YearNumbers))
		extraValues               = len(t.Rows[row-1].YearNumbers) - len(t.years)
		deleteSmallestExtraValues = true
	)

	for j := 0; j < len(t.Rows[row-1].YearNumbers); j++ {
		values = append(values, orderedNumber{
			Number: t.Rows[row-1].YearNumbers[j].Number,
			idx:    j,
		})
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Value < values[j].Value
	})

	for j := 0; j < extraValues; j++ {
		extraValue := values[j]
		for k := extraValues; k < len(values); k++ {
			if values[k].Number.Value-extraValue.Number.Value <= 50 {
				deleteSmallestExtraValues = false
				break
			}
		}
	}

	if deleteSmallestExtraValues {
		values = values[extraValues:]
		sort.Slice(values, func(i, j int) bool {
			return values[i].idx < values[j].idx
		})

		newYearNumbers := make([]YearNumber, 0, len(t.years))
		for _, v := range values {
			newYearNumbers = append(newYearNumbers, YearNumber{
				Number: v.Number,
			})
		}

		t.Rows[row-1].YearNumbers = newYearNumbers
	} else {
		t.Rows[row-1].YearNumbers = t.Rows[row-1].YearNumbers[extraValues:]
	}

	for idx := range t.Rows[row-1].YearNumbers {
		t.Rows[row-1].YearNumbers[idx].Year = t.years[idx]
	}
}

func (t *Table) removeExtraRows() {
	newRows := make([]TableRow, 0, len(t.Rows))
	for _, aRow := range t.Rows {
		if len(aRow.YearNumbers) != len(t.years) {
			continue
		}
		newRows = append(newRows, aRow)
	}

	// For  single column tables, a list of footnotes at the end of the page
	// can be confused a continuation of the table. In such case, footnotes
	// will all be listed in ascending order with each one being +1 from the
	// previous one. Remove any rows from the end that are a sequence of +1 numbers.
	if len(t.years) == 1 {
		toRemove := 0
		for i := len(newRows) - 1; i > 0; i-- {
			if int(newRows[i].YearNumbers[0].Number.Value) == int(newRows[i-1].YearNumbers[0].Number.Value)+1 {
				toRemove += 1
			}
		}
		if toRemove > 0 {
			newRows = newRows[:len(newRows)-toRemove-1]
		}
	}

	t.Rows = newRows
}

func (t *Table) addNewRow(lines []string, i int) (int, error) {
	if t.hasUnit {
		return t.newRowWithUnit(lines, i)
	}
	return t.newRow(lines, i)
}

func (t *Table) newRowWithUnit(lines []string, i int) (int, error) {
	aRow := TableRow{
		Name:        strings.TrimSpace(lines[i]),
		YearNumbers: make([]YearNumber, 0, len(t.years)),
	}
	if len(lines) <= i-1 {
		return 0, fmt.Errorf("could not complete table row")
	}

	var (
		l                  = len(lines)
		firstIdx           = -1
		consecutiveNumbers = 0
		j                  int
		numbers            = map[int]struct{}{}
	)
	max := i + 7
	if max > l {
		max = l
	}
	for j = i + 1; j < max; j++ {
		_, ok := isNumberOrNotAvailable(lines[j])
		if !ok {
			firstIdx = -1
			consecutiveNumbers = 0
			continue
		}
		numbers[j] = struct{}{}
		if firstIdx == -1 {
			firstIdx = j
		}
		consecutiveNumbers += 1

		if consecutiveNumbers != len(t.years) {
			continue
		}

		aRow.Unit = strings.TrimSpace(lines[firstIdx-1])
		for k := i + 1; k < firstIdx-1; k++ {
			_, ok := numbers[k]
			if ok {
				continue
			}
			aRow.Name += " " + strings.TrimSpace(lines[k])
		}

		break
	}

	t.Rows = append(t.Rows, aRow)

	return j - i - consecutiveNumbers, nil
}

func (t *Table) newRow(lines []string, i int) (int, error) {
	aRow := TableRow{
		Name:        strings.TrimSpace(lines[i]),
		YearNumbers: make([]YearNumber, 0, len(t.years)),
	}
	if len(lines) <= i-1 {
		return 0, fmt.Errorf("could not complete table row")
	}

	var (
		l                  = len(lines)
		firstIdx           = -1
		consecutiveNumbers = 0
		j                  int
		numbers            = map[int]struct{}{}
	)
	max := i + 6
	if l < max {
		max = l
	}
	for j = i + 1; j < max; j++ {
		_, ok := isNumberOrNotAvailable(lines[j])
		if !ok {
			firstIdx = -1
			consecutiveNumbers = 0
			continue
		}

		numbers[j] = struct{}{}
		if firstIdx == -1 {
			firstIdx = j
		}
		consecutiveNumbers += 1

		if consecutiveNumbers != len(t.years) {
			continue
		}

		for k := i + 1; k < firstIdx; k++ {
			_, ok := numbers[k]
			if ok {
				continue
			}
			aRow.Name += " " + strings.TrimSpace(lines[k])
		}

		break
	}

	t.Rows = append(t.Rows, aRow)

	return j - i - consecutiveNumbers, nil
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
	original := s
	if numRe.MatchString(s) {
		value, err := strconv.ParseFloat(strings.Replace(s, ",", "", -1), 64)
		if err == nil {
			return Number{value, true, original}, true
		}
	}

	s = strings.Replace(s, "*", "", 1)           // Remove any trailing asterisks, for example 2020*
	s = strings.Replace(s, " (baseline)", "", 1) // 2019 (baseline) -> 2019 (to parse as a valid year)
	s = strings.Replace(s, ",", "", 1)           // ,14 -> 14 (in case of parsing multiple footnotes, for example 12\n,14)

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return Number{}, false
	}
	return Number{value, true, original}, true
}
