package store

import (
	"fmt"
	"strings"
)

func toPostgresParams(sql string) string {
	if sql == "" {
		return ""
	}

	c := 0

	var builder strings.Builder

	for _, b := range sql {
		// check for placeholder
		if b == '?' {
			fmt.Fprintf(&builder, "$%d", c+1)
			c += 1
			continue
		}
		builder.WriteRune(b)
	}

	return builder.String()
}
