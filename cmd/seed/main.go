package main

import (
	"collider/internal/stores"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var eventTypes = []string{
	"user.registered",
	"user.login",
	"user.logout",
	"user.updated",
	"order.created",
	"order.paid",
	"order.shipped",
	"order.delivered",
	"payment.processed",
	"payment.failed",
	"payment.refunded",
	"product.viewed",
	"product.added_to_cart",
	"product.removed_from_cart",
	"email.sent",
	"email.opened",
	"email.clicked",
	"notification.sent",
	"notification.read",
	"api.request",
	"api.response",
	"api.error",
	"user.password_reset_requested",
	"user.password_changed",
	"user.two_factor_enabled",
	"user.two_factor_disabled",
	"user.deleted",
	"user.suspended",
	"user.reactivated",
	"user.subscription_started",
	"user.subscription_cancelled",
	"user.subscription_renewed",
	"user.invited",
	"user.invite_accepted",
	"user.feedback_submitted",
	"user.avatar_uploaded",
	"user.preferences_updated",
	"user.email_verified",
	"user.login_failed",
	"user.profile_viewed",
	"user.notification_preferences_updated",
	"user.newsletter_subscribed",
	"order.cancelled",
	"order.return_requested",
	"order.return_approved",
	"order.return_rejected",
	"order.review_submitted",
	"order.invoice_generated",
	"payment.pending",
	"payment.disputed",
	"payment.settled",
	"cart.viewed",
	"cart.updated",
	"cart.cleared",
	"checkout.started",
	"checkout.completed",
	"product.review_submitted",
	"product.wishlisted",
	"product.unwishlisted",
	"product.compared",
	"product.shared",
	"product.restock_requested",
	"product.stock_low",
	"email.bounced",
	"email.unsubscribed",
	"notification.dismissed",
	"notification.failed",
	"session.started",
	"session.expired",
	"session.terminated",
	"admin.login",
	"admin.logout",
	"admin.updated_user",
	"admin.deleted_user",
	"admin.generated_report",
	"admin.settings_updated",
	"file.uploaded",
	"file.deleted",
	"file.downloaded",
	"file.previewed",
	"support.ticket_created",
	"support.ticket_closed",
	"support.ticket_reopened",
	"support.message_sent",
	"support.rating_submitted",
	"search.performed",
	"search.filtered",
	"search.sorted",
	"settings.updated",
	"language.changed",
	"timezone.changed",
	"api.token_generated",
	"api.token_revoked",
	"api.rate_limited",
	"cron.job_started",
	"cron.job_finished",
	"cron.job_failed",
	"webhook.received",
	"webhook.verified",
	"webhook.failed",
}

func main() {
	start := time.Now()

	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := stores.Connect(databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Starting seed...")

	if err := seedEventTypes(db); err != nil {
		log.Fatal(err)
	}

	if err := seedUsers(db); err != nil {
		log.Fatal(err)
	}

	eventCount := int64(100000)
	isProd := false
	if len(os.Args) > 1 && os.Args[1] == "--prod" {
		eventCount = 10000000
		isProd = true
	}

	if err := seedEventsParallel(databaseUrl, eventCount, isProd); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nSeeded successfully in %.2f seconds\n", time.Since(start).Seconds())
}

func seedEventTypes(db *sqlx.DB) error {
	fmt.Println("Seeding event types...")

	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Preparex(`insert into event_types (name) values ($1) on conflict (name) do nothing`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, eventType := range eventTypes {
		if _, err := stmt.Exec(eventType); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("Seeded %d event types\n", len(eventTypes))
	return nil
}

func seedUsers(db *sqlx.DB) error {
	fmt.Println("Seeding users...")

	query := `
		insert into users (name)
		select 'user_' || md5(random()::text || clock_timestamp()::text)
		from generate_series(1, 1000)
		on conflict do nothing
	`

	start := time.Now()
	result, err := db.Exec(query)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Seeded %d users in %.2f seconds\n", rowsAffected, time.Since(start).Seconds())

	return nil
}

func seedEventsParallel(databaseUrl string, count int64, isProd bool) error {
	fmt.Printf("Seeding %d events using parallel inserts...\n", count)

	batchSize := int64(100000)
	if bs := os.Getenv("SEED_BATCH_SIZE"); bs != "" {
		fmt.Printf("Using batch size from SEED_BATCH_SIZE: %s\n", bs)
		fmt.Sscanf(bs, "%d", &batchSize)
	}

	numWorkers := 4
	if w := os.Getenv("SEED_WORKERS"); w != "" {
		fmt.Printf("Using %s workers (from SEED_WORKERS)\n", w)
		fmt.Sscanf(w, "%d", &numWorkers)
	} else {
		fmt.Printf("Using %d workers (set SEED_WORKERS to override)\n", numWorkers)
	}

	maxOpenConns := numWorkers * 2
	if maxOpenConns < 20 {
		maxOpenConns = 20
	}

	skipIndexes := isProd

	batches := (count + batchSize - 1) / batchSize

	var inserted int64
	start := time.Now()
	lastUpdate := start

	if skipIndexes {
		fmt.Println("Dropping indexes before seeding...")
		if err := dropIndexes(databaseUrl); err != nil {
			return fmt.Errorf("failed to drop indexes: %w", err)
		}
		fmt.Println("Indexes dropped successfully")
	}

	var wg sync.WaitGroup
	errCh := make(chan error, batches)

	insertQuery := `
		with user_ids as (
			select array_agg(user_id) as ids from users
		),
		type_ids as (
			select array_agg(type_id) as ids from event_types
		)
		insert into events (user_id, type_id, metadata, timestamp)
		select
			user_ids.ids[1 + floor(random() * array_length(user_ids.ids, 1))::integer],
			type_ids.ids[1 + floor(random() * array_length(type_ids.ids, 1))::integer],
			jsonb_build_object(
				'page',
				(array['/home', '/about', '/products', '/contact', '/login', '/checkout', '/profile', '/search'])[floor(random() * 8)::int + 1],
				'referrer',
				(array['https://google.com', 'https://twitter.com', 'https://facebook.com', 'direct', null])[floor(random() * 5)::int + 1],
				'session_id',
				md5(random()::text)
			),
			(now() at time zone 'Europe/Moscow') - (random() * interval '365 days')
		from generate_series(1, $1), user_ids, type_ids
	`

	sem := make(chan struct{}, numWorkers)

	for i := int64(0); i < batches; i++ {
		currentBatchSize := batchSize
		if i == batches-1 && count%batchSize != 0 {
			currentBatchSize = count % batchSize
		}

		wg.Add(1)

		go func(batchNum int64, size int64) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			db, err := stores.ConnectWithPool(databaseUrl, maxOpenConns, maxOpenConns/2)
			if err != nil {
				errCh <- err
				return
			}
			defer db.Close()

			tx, err := db.Beginx()
			if err != nil {
				errCh <- err
				return
			}
			defer tx.Rollback()

			optimizations := []string{
				"set local synchronous_commit = off",
				"set local work_mem = '256MB'",
				"set local maintenance_work_mem = '512MB'",
			}

			for _, opt := range optimizations {
				tx.MustExec(opt)
			}

			result, err := tx.Exec(insertQuery, size)
			if err != nil {
				errCh <- err
				return
			}

			if err := tx.Commit(); err != nil {
				errCh <- err
				return
			}

			rows, _ := result.RowsAffected()
			newInserted := atomic.AddInt64(&inserted, rows)

			if time.Since(lastUpdate) > 500*time.Millisecond {
				elapsed := time.Since(start).Seconds()
				rate := float64(newInserted) / elapsed
				remaining := float64(count-newInserted) / rate
				fmt.Printf("\rProgress: %d/%d (%.1f%%) - Rate: %.0f/s - ETA: %.0fs",
					newInserted, count, float64(newInserted)/float64(count)*100, rate, remaining)
				lastUpdate = time.Now()
			}
		}(i, currentBatchSize)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		if err != nil {
			if skipIndexes {
				fmt.Println("\nAttempting to recreate indexes after error...")
				recreateIndexes(databaseUrl)
			}
			return err
		}
	}

	elapsed := time.Since(start).Seconds()
	rate := float64(inserted) / elapsed
	fmt.Printf("\nSeeded %d events in %.2f seconds (%.0f events/sec)\n", inserted, elapsed, rate)

	if skipIndexes {
		fmt.Println("\nRecreating indexes...")
		indexStart := time.Now()
		if err := recreateIndexes(databaseUrl); err != nil {
			return fmt.Errorf("failed to recreate indexes: %w", err)
		}
		fmt.Printf("Indexes recreated in %.2f seconds\n", time.Since(indexStart).Seconds())
	}

	return nil
}

func dropIndexes(databaseUrl string) error {
	indexes := []string{
		"idx_events_user_timestamp",
		"idx_events_timestamp_desc",
		"idx_events_type_timestamp",
		"idx_events_stats",
		"idx_events_covering",
		"idx_events_metadata_gin",
	}

	db, err := stores.Connect(databaseUrl)
	if err != nil {
		return err
	}
	defer db.Close()

	for _, idx := range indexes {
		query := fmt.Sprintf("drop index if exists %s", idx)
		if _, err := db.Exec(query); err != nil {
			fmt.Printf("Warning: failed to drop index %s: %v\n", idx, err)
		}
	}

	return nil
}

func recreateIndexes(databaseUrl string) error {
	indexes := []struct {
		name string
		sql  string
	}{
		{
			"idx_events_user_timestamp",
			"create index idx_events_user_timestamp on events (user_id, \"timestamp\" desc)",
		},
		{
			"idx_events_timestamp_desc",
			"create index idx_events_timestamp_desc on events (\"timestamp\" desc)",
		},
		{
			"idx_events_type_timestamp",
			"create index idx_events_type_timestamp on events (type_id, \"timestamp\" desc)",
		},
		{
			"idx_events_stats",
			"create index idx_events_stats on events (user_id, (metadata->>'page'), type_id)",
		},
		{
			"idx_events_covering",
			"create index idx_events_covering on events (user_id, type_id, \"timestamp\" desc) include (event_id, metadata)",
		},
		{
			"idx_events_metadata_gin",
			"create index idx_events_metadata_gin on events using gin (metadata)",
		},
	}

	db, err := stores.Connect(databaseUrl)
	if err != nil {
		return err
	}
	defer db.Close()

	db.MustExec("set local maintenance_work_mem = '1GB'")

	for _, idx := range indexes {
		start := time.Now()
		if _, err := db.Exec(idx.sql); err != nil {
			fmt.Printf("Warning: failed to create index %s: %v\n", idx.name, err)
		} else {
			fmt.Printf("  Created %s in %.2fs\n", idx.name, time.Since(start).Seconds())
		}
	}

	return nil
}
