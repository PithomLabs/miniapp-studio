package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PithomLabs/miniapp-studio/secrets"
)

// runDevSecretsValidate parses a dotenv file against the registry and
// audits secret-file permissions. Names and statuses print; values
// never do.
func runDevSecretsValidate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("dev secrets-validate", flag.ContinueOnError)
	envFile := fs.String("env-file", "", "dotenv file to validate (required)")
	checkFiles := fs.String("check-files", "", "comma-separated extra secret files to permission-audit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *envFile == "" {
		return fmt.Errorf("usage: studio dev secrets-validate --env-file PATH [--check-files a,b]")
	}
	reg := secrets.StudioRegistry()
	raw, err := os.ReadFile(*envFile)
	if err != nil {
		return fmt.Errorf("read env file: %w", err)
	}
	parsed, err := secrets.ParseDotEnv(raw, reg)
	if err != nil {
		return err
	}
	problems := 0
	for _, name := range reg.Names() {
		v, _ := reg.Lookup(name)
		status := "unset"
		if _, present := parsed[name]; present {
			status = "dotenv"
		} else if strings.TrimSpace(os.Getenv(name)) != "" {
			status = "environment"
		} else if strings.TrimSpace(os.Getenv(name+"_FILE")) != "" {
			status = name + "_FILE"
		}
		fmt.Fprintf(stdout, "%s scope=%s sensitivity=%s rotation=%s status=%s\n", name, v.Scope, v.Sensitivity, v.Rotation, status)
	}
	if *checkFiles != "" {
		for _, p := range strings.Split(*checkFiles, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if err := secrets.CheckFileMode(p); err != nil {
				fmt.Fprintf(stdout, "%s FAIL: %v\n", p, err)
				problems++
			} else {
				fmt.Fprintf(stdout, "%s ok (owner-only)\n", p)
			}
		}
	}
	if problems > 0 {
		return fmt.Errorf("secrets-validate: %d file permission problem(s)", problems)
	}
	return nil
}

// runDevSecretsPlan shows which registered keys a named environment
// needs and where each value would come from. Local planning only:
// it performs no Fly/network operations.
func runDevSecretsPlan(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("dev secrets-plan", flag.ContinueOnError)
	envFile := fs.String("env-file", "", "dotenv file to plan from (required)")
	envName := fs.String("env", "", "target environment: dev|preview|staging|production (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	switch *envName {
	case "dev", "preview", "staging", "production":
	default:
		return fmt.Errorf("usage: studio dev secrets-plan --env-file PATH --env dev|preview|staging|production")
	}
	reg := secrets.StudioRegistry()
	raw, err := os.ReadFile(*envFile)
	if err != nil {
		return fmt.Errorf("read env file: %w", err)
	}
	parsed, err := secrets.ParseDotEnv(raw, reg)
	if err != nil {
		return err
	}
	inScope := 0
	for _, name := range reg.Names() {
		v, _ := reg.Lookup(name)
		scoped := false
		for _, e := range v.Environments {
			if e == *envName {
				scoped = true
				break
			}
		}
		if !scoped {
			continue
		}
		inScope++
		source := "needs value at deploy"
		if _, present := parsed[name]; present {
			source = "dotenv"
		}
		fmt.Fprintf(stdout, "%s sensitivity=%s rotation=%s source=%s\n", name, v.Sensitivity, v.Rotation, source)
	}
	fmt.Fprintf(stdout, "plan: %d keys in scope for %s (values never shown)\n", inScope, *envName)
	return nil
}
