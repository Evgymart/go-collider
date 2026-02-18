// Package main provides a database seeding tool for the collider event tracking system.
// It generates test data including users, event types, and events for performance testing.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"collider/internal/config"
	"collider/internal/stores"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	batchSize     = 25000
	numUsers      = 1000
	numEventTypes = 100
	poolSize      = 1000
)

var eventTypes = map[string]struct {
	id   int
	page string
}{
	"usr_reg":     {1, "/reg"},
	"usr_login":   {2, "/login"},
	"usr_logout":  {3, "/logout"},
	"usr_upd":     {4, "/profile"},
	"ord_new":     {5, "/order"},
	"ord_paid":    {6, "/pay"},
	"ord_ship":    {7, "/ship"},
	"ord_dlv":     {8, "/track"},
	"pay_ok":      {9, "/pay/ok"},
	"pay_fail":    {10, "/pay/fail"},
	"pay_refund":  {11, "/refund"},
	"prod_view":   {12, "/prod"},
	"cart_add":    {13, "/cart/add"},
	"cart_rm":     {14, "/cart/rm"},
	"mail_sent":   {15, "/mail/sent"},
	"mail_open":   {16, "/mail/open"},
	"mail_click":  {17, "/mail/click"},
	"ntf_sent":    {18, "/ntf/sent"},
	"ntf_read":    {19, "/ntf/read"},
	"api_req":     {20, "/api/req"},
	"api_resp":    {21, "/api/resp"},
	"api_err":     {22, "/api/err"},
	"pwd_reset":   {23, "/pwd/reset"},
	"pwd_chg":     {24, "/pwd/chg"},
	"2fa_on":      {25, "/2fa/on"},
	"2fa_off":     {26, "/2fa/off"},
	"usr_del":     {27, "/usr/del"},
	"usr_ban":     {28, "/usr/ban"},
	"usr_act":     {29, "/usr/act"},
	"sub_start":   {30, "/sub/start"},
	"sub_end":     {31, "/sub/end"},
	"sub_renew":   {32, "/sub/renew"},
	"inv_send":    {33, "/inv/send"},
	"inv_acc":     {34, "/inv/acc"},
	"feedback":    {35, "/feedback"},
	"avatar":      {36, "/avatar"},
	"prefs":       {37, "/prefs"},
	"mail_ver":    {38, "/verify"},
	"login_fail":  {39, "/login/fail"},
	"prof_view":   {40, "/prof/view"},
	"ntf_prefs":   {41, "/ntf/prefs"},
	"news_sub":    {42, "/news/sub"},
	"ord_cancel":  {43, "/ord/cancel"},
	"ret_req":     {44, "/ret/req"},
	"ret_ok":      {45, "/ret/ok"},
	"ret_no":      {46, "/ret/no"},
	"review":      {47, "/review"},
	"invoice":     {48, "/invoice"},
	"pay_pend":    {49, "/pay/pend"},
	"pay_disp":    {50, "/pay/disp"},
	"pay_settle":  {51, "/pay/settle"},
	"cart_view":   {52, "/cart"},
	"cart_upd":    {53, "/cart/upd"},
	"cart_clear":  {54, "/cart/clear"},
	"chk_start":   {55, "/chk/start"},
	"chk_done":    {56, "/chk/done"},
	"prod_rev":    {57, "/prod/rev"},
	"wish_add":    {58, "/wish/add"},
	"wish_rm":     {59, "/wish/rm"},
	"compare":     {60, "/compare"},
	"share":       {61, "/share"},
	"restock":     {62, "/restock"},
	"stock_low":   {63, "/stock/low"},
	"mail_bounce": {64, "/mail/bounce"},
	"unsub":       {65, "/unsub"},
	"ntf_dismiss": {66, "/ntf/dismiss"},
	"ntf_fail":    {67, "/ntf/fail"},
	"sess_start":  {68, "/sess/start"},
	"sess_exp":    {69, "/sess/exp"},
	"sess_end":    {70, "/sess/end"},
	"adm_login":   {71, "/adm/login"},
	"adm_logout":  {72, "/adm/logout"},
	"adm_usr_upd": {73, "/adm/usr/upd"},
	"adm_usr_del": {74, "/adm/usr/del"},
	"adm_report":  {75, "/adm/report"},
	"adm_cfg":     {76, "/adm/cfg"},
	"file_up":     {77, "/file/up"},
	"file_del":    {78, "/file/del"},
	"file_dl":     {79, "/file/dl"},
	"file_prev":   {80, "/file/prev"},
	"sup_new":     {81, "/sup/new"},
	"sup_close":   {82, "/sup/close"},
	"sup_reopen":  {83, "/sup/reopen"},
	"sup_msg":     {84, "/sup/msg"},
	"sup_rate":    {85, "/sup/rate"},
	"search":      {86, "/search"},
	"filter":      {87, "/filter"},
	"sort":        {88, "/sort"},
	"cfg_upd":     {89, "/cfg"},
	"lang":        {90, "/lang"},
	"tz":          {91, "/tz"},
	"token_gen":   {92, "/token/gen"},
	"token_rev":   {93, "/token/rev"},
	"rate_limit":  {94, "/rate"},
	"cron_start":  {95, "/cron/start"},
	"cron_done":   {96, "/cron/done"},
	"cron_fail":   {97, "/cron/fail"},
	"hook_in":     {98, "/hook/in"},
	"hook_ok":     {99, "/hook/ok"},
	"hook_fail":   {100, "/hook/fail"},
}

var indexes = []string{
	"idx_events_user_timestamp",
	"idx_events_timestamp_desc",
	"idx_events_type_timestamp",
	"idx_events_stats",
	"idx_events_covering",
	"idx_events_metadata_gin",
}

type seedPool struct {
	templates []string
}

func newSeedPool() *seedPool {
	sp := &seedPool{
		templates: make([]string, poolSize),
	}

	typeNames := make([]string, 0, len(eventTypes))
	for name := range eventTypes {
		typeNames = append(typeNames, name)
	}

	startTs := time.Now().Add(-30 * 24 * time.Hour).Unix()
	endTs := time.Now().Unix()
	step := int64((endTs - startTs) / poolSize)

	for i := 0; i < poolSize; i++ {
		userID := (i % numUsers) + 1
		typeIndex := i % len(typeNames)
		typeName := typeNames[typeIndex]
		eventType := eventTypes[typeName]

		ts := startTs + int64(i)*step
		timestamp := time.Unix(ts, 0).Format("2006-01-02 15:04:05")

		metadata, err := json.Marshal(map[string]string{"page": eventType.page})
		if err != nil {
			log.Fatalf("Failed to marshal metadata: %v", err)
		}

		sp.templates[i] = fmt.Sprintf("(%d,%d,%s,%s)", userID, eventType.id, pq.QuoteLiteral(timestamp), pq.QuoteLiteral(string(metadata)))
	}

	return sp
}

func main() {
	prodFlag := flag.Bool("prod", false, "Seed 10M events instead of 100k")
	flag.Parse()

	var totalEvents int
	if *prodFlag {
		totalEvents = 10_000_000
	} else {
		totalEvents = 100_000
	}

	fmt.Printf("Seeding %d events...\n", totalEvents)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := stores.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(runtime.NumCPU() * 2)
	db.SetMaxIdleConns(runtime.NumCPU())

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	startTime := time.Now()

	prepareDB(db)
	cleanDatabase(db)
	seedUsers(db)
	seedEventTypes(db)
	seedEvents(db, totalEvents)
	recreateIndexes(db)

	elapsed := time.Since(startTime)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	var count int
	db.Get(&count, "select count(*) from events")

	fmt.Printf("\nDone.\n")
	fmt.Printf("  Events: %d\n", count)
	fmt.Printf("  Time: %.2fs\n", elapsed.Seconds())
	fmt.Printf("  Memory: %s\n", formatBytes(m.Alloc))
	fmt.Printf("  Workers: %d\n", runtime.NumCPU())
}

func prepareDB(db *sqlx.DB) {
	for _, index := range indexes {
		db.Exec(fmt.Sprintf("drop index if exists %s", pq.QuoteIdentifier(index)))
	}
}

func cleanDatabase(db *sqlx.DB) {
	db.MustExec("truncate table events restart identity cascade")
	db.MustExec("truncate table event_types restart identity cascade")
	db.MustExec("truncate table users restart identity cascade")
}

func seedUsers(db *sqlx.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	values := make([]string, numUsers)
	for i := 1; i <= numUsers; i++ {
		values[i-1] = fmt.Sprintf("(%d,%s)", i, pq.QuoteLiteral(fmt.Sprintf("user%d", i)))
	}

	query := fmt.Sprintf("insert into users (user_id, name) values %s", strings.Join(values, ","))
	db.MustExecContext(ctx, query)

	var maxID int64
	db.GetContext(ctx, &maxID, "select max(user_id) from users")
	db.MustExecContext(ctx, "select setval('users_user_id_seq', $1, true)", maxID)
}

func seedEventTypes(db *sqlx.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	values := make([]string, len(eventTypes))
	i := 0
	for name, et := range eventTypes {
		values[i] = fmt.Sprintf("(%d,%s)", et.id, pq.QuoteLiteral(name))
		i++
	}

	query := fmt.Sprintf("insert into event_types (type_id, name) values %s", strings.Join(values, ","))
	db.MustExecContext(ctx, query)

	var maxID int64
	db.GetContext(ctx, &maxID, "select max(type_id) from event_types")
	db.MustExecContext(ctx, "select setval('event_types_type_id_seq', $1, true)", maxID)
}

func seedEvents(db *sqlx.DB, totalEvents int) {
	pool := newSeedPool()

	numBatches := (totalEvents + batchSize - 1) / batchSize
	maxWorkers := runtime.NumCPU()
	printInterval := numBatches / 10
	if printInterval < 1 {
		printInterval = 1
	}

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var eventID int64 = 1
	var currentBatch int32 = 0
	var mu sync.Mutex

	errChan := make(chan error, numBatches)

	for batch := 0; batch < numBatches; batch++ {
		wg.Add(1)
		sem <- struct{}{}

		thisBatchSize := batchSize
		if batch == numBatches-1 && totalEvents%batchSize != 0 {
			thisBatchSize = totalEvents % batchSize
		}

		go func(batchNum int, size int) {
			defer func() {
				<-sem
				wg.Done()
			}()

			if err := seedBatch(db, pool, batchNum, &eventID, size); err != nil {
				errChan <- fmt.Errorf("batch %d: %w", batchNum, err)
				return
			}
			batchNumDone := atomic.AddInt32(&currentBatch, 1)
			if batchNumDone%int32(printInterval) == 0 || int(batchNumDone) == numBatches {
				mu.Lock()
				fmt.Printf("  Progress: %d/%d batches\n", batchNumDone, numBatches)
				mu.Unlock()
			}
		}(batch, thisBatchSize)
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for err := range errChan {
		if err != nil {
			log.Printf("Error seeding batch: %v", err)
		}
	}
}

func seedBatch(db *sqlx.DB, pool *seedPool, batchNum int, eventID *int64, currentBatchSize int) error {
	values := make([]string, currentBatchSize)

	for i := 0; i < currentBatchSize; i++ {
		id := atomic.AddInt64(eventID, 1) - 1
		index := int(id) % poolSize
		template := pool.templates[index]
		values[i] = fmt.Sprintf("(%d,%s", id, template[1:])
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	query := fmt.Sprintf("insert into events (event_id, user_id, type_id, timestamp, metadata) values %s", strings.Join(values, ","))

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to insert batch: %w", err)
	}

	return nil
}

func recreateIndexes(db *sqlx.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	db.MustExecContext(ctx, fmt.Sprintf("create index concurrently if not exists %s on events (user_id, timestamp desc)", pq.QuoteIdentifier("idx_events_user_timestamp")))
	db.MustExecContext(ctx, fmt.Sprintf("create index concurrently if not exists %s on events (timestamp desc)", pq.QuoteIdentifier("idx_events_timestamp_desc")))
	db.MustExecContext(ctx, fmt.Sprintf("create index concurrently if not exists %s on events (type_id, timestamp desc)", pq.QuoteIdentifier("idx_events_type_timestamp")))
	db.MustExecContext(ctx, fmt.Sprintf("create index concurrently if not exists %s on events (user_id, (metadata->>'page'), type_id)", pq.QuoteIdentifier("idx_events_stats")))
	db.MustExecContext(ctx, fmt.Sprintf("create index concurrently if not exists %s on events (user_id, type_id, timestamp desc) include (event_id, metadata)", pq.QuoteIdentifier("idx_events_covering")))
	db.MustExecContext(ctx, fmt.Sprintf("create index concurrently if not exists %s on events using gin (metadata)", pq.QuoteIdentifier("idx_events_metadata_gin")))
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
