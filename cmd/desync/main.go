// Command desync declaratively manages deSEC DNS resources.
//
// Usage:
//
//	desync [global flags] <subcommand> [subcommand flags]
//
// Global flags:
//
//	-token TOKEN   deSEC API token (defaults to $DESEC_TOKEN)
//	-f FILE        path to the JSON state file (default: desync.json)
//
// Subcommands:
//
//	init           Bootstrap a state file from the current API contents.
//
// Subcommands:
//
//	plan           Show what changes would be made without applying them.
//	apply          Apply changes (prompts for confirmation; use -auto-approve to skip).
//	tokens list    List all tokens in the account with their IDs and metadata.
//	tokens create  Create a new token and print its secret (shown only once).
//
// The tokens subcommands let you discover token IDs so that you can reference
// them in the tokenPolicies section of your state file.
//
// Example state file (desync.json):
//
//	{
//	  "tokenPolicies": [
//	    {
//	      "tokenId": "3a6b94b5-d20e-40bd-a7cc-521f5c79fab3",
//	      "policies": [
//	        {"domain": null, "subname": null, "type": null, "permWrite": false},
//	        {"domain": "example.com", "subname": null, "type": null, "permWrite": true}
//	      ]
//	    }
//	  ],
//	  "domains": [
//	    {
//	      "name": "example.com",
//	      "rrsets": [
//	        {"subname": "",    "type": "A",    "records": ["1.2.3.4"]},
//	        {"subname": "www", "type": "A",    "records": ["1.2.3.4"]},
//	        {"subname": "www", "type": "AAAA", "ttl": 300, "records": ["2001:db8::1"]}
//	      ]
//	    }
//	  ]
//	}
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/afero"

	"codeberg.org/xchangeee/desync/internal/api"
	"codeberg.org/xchangeee/desync/internal/config"
	"codeberg.org/xchangeee/desync/internal/engine"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	global := flag.NewFlagSet("desync", flag.ContinueOnError)
	global.Usage = printUsage
	token := global.String("token", os.Getenv("DESEC_TOKEN"), "deSEC API token (or set DESEC_TOKEN)")
	file := global.String("f", "desync.json", "path to declarative state file")

	if err := global.Parse(args); err != nil {
		return err
	}

	sub := global.Args()
	if len(sub) == 0 {
		printUsage()
		return fmt.Errorf("subcommand required")
	}

	switch sub[0] {
	case "init":
		return runInit(*token, *file, sub[1:])
	case "plan":
		return runPlan(*token, *file, sub[1:])
	case "apply":
		return runApply(*token, *file, sub[1:])
	case "tokens":
		return runTokens(*token, sub[1:])
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand %q", sub[0])
	}
}

// ---- init ------------------------------------------------------------------

func runInit(token, file string, args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	force := fs.Bool("force", false, "overwrite existing state file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("API token required: use -token or set DESEC_TOKEN")
	}

	if !*force {
		if _, err := os.Stat(file); err == nil {
			return fmt.Errorf("%s already exists; use -force to overwrite", file)
		}
	}

	fmt.Println("Fetching current state from API...")
	client := api.NewClient(token)
	cfg, err := engine.FetchState(client)
	if err != nil {
		return fmt.Errorf("fetching state: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.WriteFile(file, data, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", file, err)
	}

	nRRsets := 0
	for _, d := range cfg.Domains {
		nRRsets += len(d.RRsets)
	}
	fmt.Printf("Wrote %s (%d token policy groups, %d domains, %d rrsets).\n",
		file, len(cfg.TokenPolicies), len(cfg.Domains), nRRsets)
	return nil
}

// ---- plan ------------------------------------------------------------------

func runPlan(token, file string, args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, cfg, err := setup(token, file)
	if err != nil {
		return err
	}

	fmt.Println("Fetching current state...")
	diff, err := engine.Diff(cfg, client)
	if err != nil {
		return fmt.Errorf("computing diff: %w", err)
	}

	fmt.Println()
	engine.PrintPlan(diff)
	return nil
}

// ---- apply -----------------------------------------------------------------

func runApply(token, file string, args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	autoApprove := fs.Bool("auto-approve", false, "skip interactive confirmation prompt")
	if err := fs.Parse(args); err != nil {
		return err
	}

	client, cfg, err := setup(token, file)
	if err != nil {
		return err
	}

	fmt.Println("Fetching current state...")
	diff, err := engine.Diff(cfg, client)
	if err != nil {
		return fmt.Errorf("computing diff: %w", err)
	}

	fmt.Println()
	engine.PrintPlan(diff)

	cr, up, del := diff.Counts()
	if cr+up+del == 0 {
		return nil
	}

	fmt.Println()
	if !*autoApprove {
		fmt.Print("Apply these changes? Type 'yes' to confirm: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if strings.TrimSpace(scanner.Text()) != "yes" {
			fmt.Println("Apply cancelled.")
			return nil
		}
	}

	fmt.Println()
	if err := engine.Apply(diff, cfg, client); err != nil {
		return fmt.Errorf("apply: %w", err)
	}
	fmt.Println("\nApply complete.")
	return nil
}

// ---- tokens ----------------------------------------------------------------

func runTokens(token string, args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: desync tokens <list|create>")
		return fmt.Errorf("tokens subcommand required")
	}
	switch args[0] {
	case "list":
		return runTokensList(token, args[1:])
	case "create":
		return runTokensCreate(token, args[1:])
	default:
		return fmt.Errorf("unknown tokens subcommand %q", args[0])
	}
}

func runTokensList(token string, args []string) error {
	fs := flag.NewFlagSet("tokens list", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("API token required: use -token or set DESEC_TOKEN")
	}
	client := api.NewClient(token)
	tokens, err := client.ListTokens()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCREATED\tLAST USED\tVALID\tPERM_CREATE\tPERM_DELETE\tPERM_MGMT")
	for _, t := range tokens {
		lastUsed := "never"
		if t.LastUsed != nil {
			lastUsed = *t.LastUsed
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%v\t%v\t%v\n",
			t.ID, t.Name, t.Created, lastUsed, t.IsValid,
			t.PermCreateDomain, t.PermDeleteDomain, t.PermManageTokens)
	}
	return w.Flush()
}

func runTokensCreate(token string, args []string) error {
	fs := flag.NewFlagSet("tokens create", flag.ContinueOnError)
	name := fs.String("name", "", "token name")
	permCreate := fs.Bool("perm-create-domain", false, "grant perm_create_domain")
	permDelete := fs.Bool("perm-delete-domain", false, "grant perm_delete_domain")
	permManage := fs.Bool("perm-manage-tokens", false, "grant perm_manage_tokens")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if token == "" {
		return fmt.Errorf("API token required: use -token or set DESEC_TOKEN")
	}
	if *name == "" {
		return fmt.Errorf("-name is required")
	}

	client := api.NewClient(token)
	t, err := client.CreateToken(api.TokenWriteFields{
		Name:             *name,
		PermCreateDomain: *permCreate,
		PermDeleteDomain: *permDelete,
		PermManageTokens: *permManage,
		AllowedSubnets:   []string{"0.0.0.0/0", "::/0"},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Token created.\n")
	fmt.Printf("  ID:     %s\n", t.ID)
	fmt.Printf("  Name:   %s\n", t.Name)
	fmt.Printf("  SECRET: %s\n", t.Secret)
	fmt.Println()
	fmt.Println("The secret is shown only once. Store it securely.")
	fmt.Printf("Reference this token in your state file with tokenId: %q\n", t.ID)
	return nil
}

// ---- shared ----------------------------------------------------------------

func setup(token, file string) (*api.Client, *config.Config, error) {
	if token == "" {
		return nil, nil, fmt.Errorf("API token required: use -token or set DESEC_TOKEN")
	}
	var (
		cfg *config.Config
		err error
	)
	if file == "-" {
		cfg, err = config.LoadReader(os.Stdin, "<stdin>")
	} else {
		cfg, err = config.Load(afero.NewOsFs(), file)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("loading %s: %w", file, err)
	}
	return api.NewClient(token), cfg, nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: desync [flags] <subcommand> [subcommand flags]

Global flags:
  -token TOKEN   deSEC API token (default: $DESEC_TOKEN)
  -f    FILE     state file (default: desync.json)

Subcommands:
  init [-force]              Bootstrap state file from current API contents
  plan                       Show planned changes without applying them
  apply [-auto-approve]      Apply changes (prompts unless -auto-approve)
  tokens list                List all tokens with their IDs
  tokens create -name NAME   Create a new token and print its secret`)
}
