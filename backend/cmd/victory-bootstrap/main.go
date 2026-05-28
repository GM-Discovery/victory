package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"victory/backend/internal/db"
	"victory/backend/internal/identity"
)

func main() {
	log.SetFlags(0)

	if len(os.Args) < 2 {
		fatalUsage()
	}

	var err error
	switch os.Args[1] {
	case "producer":
		err = runProducer(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}

	if err != nil {
		log.Fatalf("victory-bootstrap: %v", err)
	}
}

func runProducer(args []string) error {
	fs := flag.NewFlagSet("producer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		discordUserID = fs.String("discord-user-id", "", "Discord user ID linked to the Victory user")
		userID        = fs.String("user-id", "", "Victory user ID")
		handle        = fs.String("handle", "", "Victory handle")
		locationSlug  = fs.String("location", "amurray-family", "Location slug for the producer grant")
	)

	if err := fs.Parse(args); err != nil {
		return err
	}

	selectorCount := 0
	if strings.TrimSpace(*discordUserID) != "" {
		selectorCount++
	}
	if strings.TrimSpace(*userID) != "" {
		selectorCount++
	}
	if strings.TrimSpace(*handle) != "" {
		selectorCount++
	}
	if selectorCount == 0 {
		return fmt.Errorf("one of --discord-user-id, --user-id, or --handle is required")
	}
	if selectorCount > 1 {
		return fmt.Errorf("only one of --discord-user-id, --user-id, or --handle may be set")
	}

	databaseURL := getenv("DATABASE_URL", "postgres://victory:REDACTED@127.0.0.1:5432/victory?sslmode=disable")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.NewPool(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("db connect failed: %w", err)
	}
	defer pool.Close()

	result, err := identity.BootstrapProducer(ctx, pool, identity.BootstrapProducerInput{
		DiscordUserID: strings.TrimSpace(*discordUserID),
		UserID:        strings.TrimSpace(*userID),
		Handle:        strings.TrimSpace(*handle),
		LocationSlug:  strings.TrimSpace(*locationSlug),
	})
	if err != nil {
		switch err {
		case identity.ErrBootstrapLocationNotFound:
			return fmt.Errorf("location %q not found", strings.TrimSpace(*locationSlug))
		case identity.ErrBootstrapUserNotFound:
			if strings.TrimSpace(*discordUserID) != "" {
				return fmt.Errorf("no Victory user linked to Discord user ID %q", strings.TrimSpace(*discordUserID))
			}
			if strings.TrimSpace(*userID) != "" {
				return fmt.Errorf("Victory user ID %q not found", strings.TrimSpace(*userID))
			}
			return fmt.Errorf("Victory handle %q not found", strings.TrimSpace(*handle))
		case identity.ErrBootstrapTargetRequired:
			return fmt.Errorf("a user selector is required")
		default:
			return err
		}
	}

	label := result.User.Handle
	if label == "" {
		label = result.User.DisplayName
	}
	if label == "" {
		label = result.User.ID
	}

	fmt.Println("Producer grant complete.")
	fmt.Printf("Victory user: %s (%s)\n", result.User.ID, label)
	fmt.Printf("Location: %s (%s)\n", result.Location.Name, result.Location.Slug)
	fmt.Println("Role: producer")
	fmt.Printf("Already existed: %t\n", result.AlreadyExisted)
	return nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalUsage() {
	printUsage()
	os.Exit(2)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/victory-bootstrap producer --discord-user-id <discord_user_id>")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/victory-bootstrap producer --user-id <victory_user_id>")
	fmt.Fprintln(os.Stderr, "  go run ./cmd/victory-bootstrap producer --handle <victory_handle>")
}
