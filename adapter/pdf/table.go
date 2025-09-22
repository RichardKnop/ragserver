package pdf

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"go.uber.org/zap"
)

type Table struct {
	Title string
	Rows  []Row
}

type Row []string

type Cell struct {
	Text string
}

func parseTables(logger *zap.Logger, html io.Reader) ([]Table, error) {
	contents, err := io.ReadAll(html)
	if err != nil {
		return nil, err
	}

	unescaped := strings.Replace(string(contents), `\"`, `"`, -1)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(unescaped))
	if err != nil {
		return nil, err
	}

	var (
		tables              = []Table{}
		rowSpan, rowSpanIdx int
		rowSpanCell         string
	)

	doc.Find("table").Each(func(i int, tableSel *goquery.Selection) {
		aTable := Table{}
		tableSel.Find("tr").Each(func(index int, rowSel *goquery.Selection) {
			aRow := Row{}
			idx := 0
			rowSel.Find("td").Each(func(index int, cellSel *goquery.Selection) {
				if cellSel != nil {
					cellSpanStr, cellSpanExists := cellSel.Attr("rowspan")
					if cellSpanExists {
						span, err := strconv.Atoi(cellSpanStr)
						if err != nil {
							logger.Sugar().With("error", err).Error("failed to parse rowspan attribute")
						} else {
							rowSpan = span - 1
							rowSpanIdx = idx
							rowSpanCell = cellSel.Text()
						}
					} else if rowSpan > 0 && rowSpanIdx == idx {
						aRow = append(aRow, rowSpanCell)
						rowSpan -= 1
						idx += 1
					}
					aRow = append(aRow, cellSel.Text())
				}
				idx += 1
			})
			if !emptyRow(aRow) {
				aTable.Rows = append(aTable.Rows, aRow)
			} else {
				tables = append(tables, aTable)
				aTable = Table{}
			}
		})
		tables = append(tables, aTable)
	})

	for i, aTable := range tables {
		if len(aTable.Rows) > 1 && len(aTable.Rows[0]) < len(aTable.Rows[1]) {
			// Remove header row if it has fewer columns than the next row
			tables[i].Title = strings.Join(aTable.Rows[0], " ")
			tables[i].Rows = aTable.Rows[1:]
		}
	}

	return tables, nil
}

func (t Table) ToContexts() []string {
	if len(t.Rows) <= 1 {
		return nil
	}

	if len(t.Rows[0]) < 1 {
		return nil
	}

	contexts := make([]string, 0, len(t.Rows)-1)
	for _, aRow := range t.Rows[1:] {
		var (
			leftSide  = aRow[0]
			rightSide = make([]string, 0, len(aRow)-1)
		)
		for i, aCell := range aRow[1:] {
			var (
				left  = strings.TrimSpace(t.Rows[0][i+1])
				right = strings.TrimSpace(aCell)
			)
			if right == "" {
				continue
			}
			if isNum, ok := isNumber(left); ok && isNum.ValidYear() {
				left = fmt.Sprintf("For year %d", int(isNum.Value))
			}
			rightSide = append(rightSide, fmt.Sprintf("%s: %s", left, right))
		}
		if len(rightSide) == 0 {
			continue
		}
		contexts = append(contexts, fmt.Sprintf("%s: %s", leftSide, strings.Join(rightSide, ", ")))
	}
	return contexts
}

func emptyRow(row []string) bool {
	for _, cell := range row {
		if cell != "" {
			return false
		}
	}
	return true
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

var numRe, _ = regexp.Compile(`^(\d*\.?\d+|\d{1,3}(,\d{3})*(\.\d+)?)$`)

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
