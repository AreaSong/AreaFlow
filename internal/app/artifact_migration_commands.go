package app

import (
	"context"
	"flag"
	"fmt"
	"strconv"
	"time"

	"github.com/areasong/areaflow/internal/config"
	"github.com/areasong/areaflow/internal/db"
	"github.com/areasong/areaflow/internal/project"
)

func (c command) runArtifactMigration(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: areaflow artifact migration <inventory|copy|activate|complete-observation> <project>")
	}
	cfg := config.FromEnv()
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := project.NewStore(pool)
	record, err := store.GetByKey(ctx, args[1])
	if err != nil {
		return err
	}

	switch args[0] {
	case "inventory":
		flags := flag.NewFlagSet("artifact migration inventory", flag.ContinueOnError)
		flags.SetOutput(c.stderr)
		sourceBackend := flags.String("source-backend", "local", "source artifact backend")
		targetBackend := flags.String("target-backend", "s3", "target artifact backend")
		jsonOutput := flags.Bool("json", false, "JSON output")
		if err := flags.Parse(args[2:]); err != nil {
			return err
		}
		inventory, err := store.ArtifactMigrationInventory(ctx, record, *sourceBackend, *targetBackend)
		if err != nil {
			return err
		}
		if *jsonOutput {
			return c.printJSON(inventory)
		}
		fmt.Fprintf(c.stdout, "artifact migration inventory: project=%s pending=%d verified=%d activated=%d observing=%d stable=%d\n", record.Key, inventory.Pending, inventory.Verified, inventory.Activated, inventory.Observing, inventory.Stable)
		return nil
	case "copy":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow artifact migration copy <project> <artifact-id> --target-backend s3 --target-root PREFIX --actor ACTOR --reason TEXT")
		}
		artifactID, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil || artifactID <= 0 {
			return fmt.Errorf("artifact id must be a positive integer")
		}
		flags := flag.NewFlagSet("artifact migration copy", flag.ContinueOnError)
		flags.SetOutput(c.stderr)
		targetBackend := flags.String("target-backend", "s3", "target artifact backend")
		targetRoot := flags.String("target-root", "", "target prefix or local root")
		actor := flags.String("actor", "", "audit actor")
		reason := flags.String("reason", "", "migration reason")
		jsonOutput := flags.Bool("json", false, "JSON output")
		if err := flags.Parse(args[3:]); err != nil {
			return err
		}
		location, err := store.CopyArtifactToBackend(ctx, record, artifactID, project.CopyArtifactOptions{
			TargetBackend: *targetBackend, TargetRoot: *targetRoot, Actor: *actor, Reason: *reason,
		})
		if err != nil {
			return err
		}
		if *jsonOutput {
			return c.printJSON(location)
		}
		fmt.Fprintf(c.stdout, "artifact migration copied: artifact=%d location=%d backend=%s verified=%t\n", artifactID, location.ID, location.StorageBackend, location.VerifiedAt != nil)
		return nil
	case "activate":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow artifact migration activate <project> <artifact-id> --location-id ID --observe-until RFC3339 --actor ACTOR --reason TEXT")
		}
		artifactID, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil || artifactID <= 0 {
			return fmt.Errorf("artifact id must be a positive integer")
		}
		flags := flag.NewFlagSet("artifact migration activate", flag.ContinueOnError)
		flags.SetOutput(c.stderr)
		locationID := flags.Int64("location-id", 0, "verified target location id")
		observeUntil := flags.String("observe-until", "", "RFC3339 observation deadline")
		actor := flags.String("actor", "", "audit actor")
		reason := flags.String("reason", "", "activation reason")
		jsonOutput := flags.Bool("json", false, "JSON output")
		if err := flags.Parse(args[3:]); err != nil {
			return err
		}
		deadline, err := time.Parse(time.RFC3339, *observeUntil)
		if err != nil {
			return fmt.Errorf("observe-until must be RFC3339: %w", err)
		}
		artifactRecord, err := store.ActivateArtifactLocation(ctx, record, artifactID, project.ActivateArtifactOptions{
			TargetLocationID: *locationID, ObservationUntil: deadline, Actor: *actor, Reason: *reason,
		})
		if err != nil {
			return err
		}
		if *jsonOutput {
			return c.printJSON(artifactRecord)
		}
		fmt.Fprintf(c.stdout, "artifact migration activated: artifact=%d backend=%s observation_until=%s\n", artifactID, artifactRecord.StorageBackend, deadline.UTC().Format(time.RFC3339))
		return nil
	case "complete-observation":
		if len(args) < 3 {
			return fmt.Errorf("usage: areaflow artifact migration complete-observation <project> <artifact-id> --actor ACTOR --reason TEXT")
		}
		artifactID, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil || artifactID <= 0 {
			return fmt.Errorf("artifact id must be a positive integer")
		}
		flags := flag.NewFlagSet("artifact migration complete-observation", flag.ContinueOnError)
		flags.SetOutput(c.stderr)
		actor := flags.String("actor", "", "audit actor")
		reason := flags.String("reason", "", "observation completion reason")
		jsonOutput := flags.Bool("json", false, "JSON output")
		if err := flags.Parse(args[3:]); err != nil {
			return err
		}
		artifactRecord, err := store.CompleteArtifactObservation(ctx, record, artifactID, project.CompleteArtifactObservationOptions{Actor: *actor, Reason: *reason})
		if err != nil {
			return err
		}
		if *jsonOutput {
			return c.printJSON(artifactRecord)
		}
		fmt.Fprintf(c.stdout, "artifact migration observation complete: artifact=%d backend=%s status=stable\n", artifactID, artifactRecord.StorageBackend)
		return nil
	default:
		return fmt.Errorf("unknown artifact migration command %q", args[0])
	}
}
