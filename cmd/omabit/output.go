package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"

	"git.thechristophgroup.org/thelifeandtimes/omabit/internal/omabittheme"
)

func safe(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, s)
}
func readable(w io.Writer, value any) bool {
	table := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	defer table.Flush()
	switch v := value.(type) {
	case omabittheme.Palette:
		fmt.Fprintf(table, "Theme: %s (dark=%t)\nPrimary: %s\nSecondary: %s\nTertiary: %s\nBackground: %s\nSurface: %s\n", safe(v.Name), v.Dark, v.Primary, v.Secondary, v.Tertiary, v.Background, v.Surface)
	case omabittheme.Preference:
		fmt.Fprintf(table, "Ship: %s / %s\nAutomatic: %t\nPending: %t\nLast theme: %s\nLast published: %s\n", safe(v.Name), safe(v.Ship), v.Automatic, v.Pending, safe(v.LastTheme), safe(v.LastPublished))
		if v.LastError != "" {
			fmt.Fprintln(table, safe(v.LastError))
		}
	case omabittheme.State:
		fmt.Fprintln(table, "CONNECTION\tSHIP\tAUTOMATIC\tPENDING\tLAST THEME\tERROR")
		ids := []string{}
		for id := range v.Ships {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			p := v.Ships[id]
			fmt.Fprintf(table, "%s\t%s\t%t\t%t\t%s\t%s\n", safe(p.Name), safe(p.Ship), p.Automatic, p.Pending, safe(p.LastTheme), safe(p.LastError))
		}
		if v.Migration != nil {
			fmt.Fprintf(table, "Migration complete: %t\nOriginal source digest: %s\n", v.Migration.Complete, v.Migration.SourceDigest)
		}
	case omabittheme.MigrationPlan:
		fmt.Fprintf(table, "Source digest: %s\nLegacy ships: %d\n", v.Digest, len(v.Ships))
		for _, p := range v.Ships {
			fmt.Fprintf(table, "%s\t%s\tautomatic=%t\tpending=%t\n", safe(p.Ship), safe(p.URL), p.Automatic, p.Pending)
		}
		for _, issue := range v.Issues {
			fmt.Fprintln(table, "Blocked:", safe(issue))
		}
	default:
		return false
	}
	return true
}
