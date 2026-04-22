package engine

import "fmt"

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorDim    = "\033[2m"
)

// PrintPlan writes a terraform-style plan to stdout.
func PrintPlan(d *DiffResult) {
	hasChanges := false

	if len(d.TokenPolicies) > 0 {
		fmt.Println("Token Policies:")
		for _, tp := range d.TokenPolicies {
			if len(tp.Policies) == 0 {
				fmt.Printf("  %s(no changes) token %s%s\n", colorDim, tp.TokenID, colorReset)
				continue
			}
			hasChanges = true
			fmt.Printf("  Token %s:\n", tp.TokenID)
			for _, p := range tp.Policies {
				printPolicyLine("    ", p)
			}
		}
		fmt.Println()
	}

	if len(d.Domains) > 0 {
		fmt.Println("RRsets:")
		for _, dd := range d.Domains {
			fmt.Printf("  Domain %q:\n", dd.Domain)
			for _, r := range dd.Changes {
				hasChanges = true
				subLabel := r.Subname
				if subLabel == "" {
					subLabel = "@"
				}
				switch r.Kind {
				case ChangeCreate:
					fmt.Printf("    %s+ create  %-24s %s%s\n", colorGreen, subLabel, r.Type, colorReset)
					if r.Desired != nil {
						fmt.Printf("              ttl=%d records=%s\n", r.Desired.EffectiveTTL(), joinStrs(r.Desired.Records))
					}
				case ChangeUpdate:
					fmt.Printf("    %s~ update  %-24s %s%s\n", colorYellow, subLabel, r.Type, colorReset)
					for _, fd := range r.Diffs {
						fmt.Printf("        %s~ %s: %s -> %s%s\n", colorYellow, fd.Field, fd.Current, fd.Desired, colorReset)
					}
				case ChangeDelete:
					fmt.Printf("    %s- delete  %-24s %s%s\n", colorRed, subLabel, r.Type, colorReset)
				}
			}
		}
		fmt.Println()
	}

	cr, up, del := d.Counts()
	if !hasChanges {
		fmt.Println("No changes. Infrastructure is up-to-date.")
		return
	}
	fmt.Printf("Plan: %d to create, %d to update, %d to delete.\n", cr, up, del)
}

func printPolicyLine(indent string, p PolicyDiff) {
	label := fmt.Sprintf("{domain=%s subname=%s type=%s}", ptrStr(p.Domain), ptrStr(p.Subname), ptrStr(p.Type))
	switch p.Kind {
	case ChangeCreate:
		fmt.Printf("%s%s+ create policy %s perm_write=%s%s\n",
			indent, colorGreen, label, p.Diffs[0].Desired, colorReset)
	case ChangeUpdate:
		fmt.Printf("%s%s~ update policy %s%s\n", indent, colorYellow, label, colorReset)
		for _, fd := range p.Diffs {
			fmt.Printf("%s    %s~ %s: %s -> %s%s\n", indent, colorYellow, fd.Field, fd.Current, fd.Desired, colorReset)
		}
	case ChangeDelete:
		fmt.Printf("%s%s- delete policy %s%s\n", indent, colorRed, label, colorReset)
	}
}
