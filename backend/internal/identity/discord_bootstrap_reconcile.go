package identity

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ReconcileDiscordBootstrap(ctx context.Context, pool *pgxpool.Pool, cfg DiscordServerLinkConfig) error {
	runtimeCfg, err := resolveDiscordServerLinkRuntimeConfig(ctx, pool, cfg)
	if err != nil {
		return err
	}
	if !DiscordServerLinkConfigured(runtimeCfg) {
		return nil
	}

	location, err := resolveProducerOfficeLocation(ctx, pool)
	if err != nil {
		return err
	}

	linkRecord, err := loadDiscordServerLinkRecord(ctx, pool, location.ID)
	if err != nil {
		return err
	}
	if !linkRecord.Active || strings.TrimSpace(linkRecord.DiscordGuildID) == "" {
		return nil
	}

	if _, err := reconcileDiscordChannelMappings(ctx, pool, runtimeCfg, location.ID, linkRecord.DiscordGuildID, ""); err != nil {
		log.Printf("discord bootstrap channel reconcile failed: %v", err)
	}

	if DiscordMicCommandConfigured(runtimeCfg) {
		if registered, _, err := ensureDiscordMicCommand(ctx, runtimeCfg, linkRecord.DiscordGuildID); err != nil {
			log.Printf("discord mic command reconcile failed: %v", err)
		} else if registered {
			log.Printf("discord mic command reconciled for guild %s", strings.TrimSpace(linkRecord.DiscordGuildID))
		}
	}

	return nil
}

func scheduleDiscordBootstrapReconcile(pool *pgxpool.Pool, cfg DiscordServerLinkConfig) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := ReconcileDiscordBootstrap(ctx, pool, cfg); err != nil {
			log.Printf("discord bootstrap reconcile failed: %v", err)
		}
	}()
}
