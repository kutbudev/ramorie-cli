package commands

import "strings"

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// extractProjectFlag rescues a -p/--project value that urfave/cli swallowed
// into the positional arguments. urfave/cli v2 stops parsing flags after the
// first positional, so the natural `remember "my note" -p myproject` ends up
// with "-p myproject" as positional content and an unset project flag. This
// scans the positionals, pulls out the first project flag it finds (in any of
// the -p val, --project val, -p=val, --project=val, -pval forms), and returns
// the value plus the remaining args. If none is present it returns ("", args).
func extractProjectFlag(args []string) (project string, rest []string) {
	rest = make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case project == "" && (a == "-p" || a == "--project"):
			if i+1 < len(args) {
				project = args[i+1]
				i++ // consume the value too
			}
			continue
		case project == "" && strings.HasPrefix(a, "--project="):
			project = strings.TrimPrefix(a, "--project=")
			continue
		case project == "" && strings.HasPrefix(a, "-p="):
			project = strings.TrimPrefix(a, "-p=")
			continue
		case project == "" && len(a) > 2 && strings.HasPrefix(a, "-p") && !strings.HasPrefix(a, "--"):
			project = a[2:] // -pmyproject
			continue
		}
		rest = append(rest, a)
	}
	return project, rest
}
