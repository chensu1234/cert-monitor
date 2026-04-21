package checker

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
)

// PrintTable prints results as a human-readable ASCII table
func PrintTable(w io.Writer, results []Result, warnDays, criticalDays int) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HOST\tPORT\tISSUE\tDAYS\tEXPIRES\tISSUER\tCN\tWILDCARD\tSANs")

	for _, r := range results {
		issueStr := r.Issue
		if r.Error != "" {
			issueStr = "error: " + r.Error
		}
		fmt.Fprintf(tw, "%s\t%d\t%s\t%d\t%s\t%s\t%s\t%v\t%d\n",
			r.Host, r.Port, issueStr, r.DaysRemaining,
			r.NotAfter, r.Issuer, r.CommonName,
			r.IsWildcard, r.SANs)
	}
	tw.Flush()
}

// PrintJSON prints results as JSON
func PrintJSON(w io.Writer, results []Result) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(map[string]interface{}{
		"version":  "1.0",
		"count":    len(results),
		"results":  results,
	})
}
