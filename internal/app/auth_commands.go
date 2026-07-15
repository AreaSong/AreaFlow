package app

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/areasong/areaflow/internal/auth"
	"github.com/areasong/areaflow/internal/config"
	"github.com/areasong/areaflow/internal/db"
)

func (c command) runAuth(ctx context.Context, args []string) error {
	if len(args) < 2 || args[0] != "token" {
		return fmt.Errorf("usage: areaflow auth token <create|list|revoke>")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	service := auth.NewService(pool).WithTokenMaxTTL(cfg.Auth.TokenMaxTTL)

	switch args[1] {
	case "create":
		flags := flag.NewFlagSet("auth token create", flag.ContinueOnError)
		flags.SetOutput(c.stderr)
		actor := flags.String("actor", "", "audit actor")
		reason := flags.String("reason", "", "creation reason")
		expiresAtValue := flags.String("expires-at", "", "RFC3339 expiration")
		jsonOutput := flags.Bool("json", false, "JSON output")
		var projects stringListFlag
		var capabilities stringListFlag
		flags.Var(&projects, "project", "allowed project key; repeatable, * for all")
		flags.Var(&capabilities, "capability", "allowed capability; repeatable")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		var expiresAt *time.Time
		if strings.TrimSpace(*expiresAtValue) != "" {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*expiresAtValue))
			if err != nil {
				return fmt.Errorf("parse expires-at: %w", err)
			}
			expiresAt = &parsed
		} else {
			defaultExpiry := time.Now().UTC().Add(30 * 24 * time.Hour)
			expiresAt = &defaultExpiry
		}
		created, err := service.CreateToken(ctx, auth.CreateTokenOptions{
			Actor: *actor, CreatedBy: *actor, Reason: *reason, Projects: projects, Capabilities: capabilities, ExpiresAt: expiresAt,
		})
		if err != nil {
			return err
		}
		if *jsonOutput {
			return c.printJSON(map[string]any{
				"token": created.Token, "token_key": created.Record.TokenKey, "actor": created.Record.Actor,
				"projects": created.Record.Projects, "capabilities": created.Record.Capabilities,
				"expires_at": created.Record.ExpiresAt, "created_at": created.Record.CreatedAt,
			})
		}
		fmt.Fprintf(c.stdout, "token %s\n", created.Token)
		fmt.Fprintf(c.stdout, "token_key %s\n", created.Record.TokenKey)
		fmt.Fprintln(c.stdout, "The token is shown once; store it in a secure local credential store.")
		return nil
	case "list":
		jsonOutput, err := outputJSON(args[2:])
		if err != nil {
			return err
		}
		records, err := service.ListTokens(ctx)
		if err != nil {
			return err
		}
		if jsonOutput {
			return c.printJSON(records)
		}
		for _, record := range records {
			fmt.Fprintf(c.stdout, "%s\t%s\t%s\t%s\n", record.TokenKey, record.Status, record.Actor, strings.Join(record.Projects, ","))
		}
		return nil
	case "revoke":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow auth token revoke <token-key> --actor ACTOR --reason TEXT")
		}
		flags := flag.NewFlagSet("auth token revoke", flag.ContinueOnError)
		flags.SetOutput(c.stderr)
		actor := flags.String("actor", "", "audit actor")
		reason := flags.String("reason", "", "revocation reason")
		if err := flags.Parse(args[3:]); err != nil {
			return err
		}
		if err := service.RevokeToken(ctx, args[2], *actor, *reason); err != nil {
			return err
		}
		fmt.Fprintf(c.stdout, "revoked %s\n", args[2])
		return nil
	default:
		return fmt.Errorf("unknown auth token command %q", args[1])
	}
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
