package output

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/nbw/firehose/internal/client"
)

func tapsTable(w io.Writer, ts []client.Tap) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tNAME\tTOKEN PREFIX\tRULES\tCREATED")
	for _, t := range ts {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\n",
			t.ID, ellipsize(t.Name, 32), t.TokenPrefix, t.RulesCount, t.CreatedAt)
	}
	tw.Flush()
}

func rulesTable(w io.Writer, rs []client.Rule) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tVALUE\tTAG\tNSFW\tQUALITY")
	for _, r := range rs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			r.ID, ellipsize(r.Value, 60), r.Tag, fmtBoolPtr(r.NSFW), fmtBoolPtr(r.Quality))
	}
	tw.Flush()
}

func ellipsize(s string, max int) string {
	if max <= 1 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max-1]) + "…"
}

func fmtBoolPtr(b *bool) string {
	if b == nil {
		return "-"
	}
	if *b {
		return "true"
	}
	return "false"
}
