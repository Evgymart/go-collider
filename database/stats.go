package database

import (
	"collider/models"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type StatsStore struct {
	db *sqlx.DB
}

func NewStatsStore(db *sqlx.DB) *StatsStore {
	return &StatsStore{db: db}
}

func (s StatsStore) GetStats(data models.GetStatsData) (*models.Stats, error) {
	var stats models.Stats
	query := `
        with page_stats as (
            select
                metadata->>'page' as page,
                count(*) as page_events,
                count(distinct user_id) as page_users
            from events
            where metadata->>'page' is not null
                and (timestamp >= $1::timestamp or $1::timestamp is null)
                and (timestamp < $2::timestamp or $2::timestamp is null)
                and (type_id = $3::uuid or $3::uuid is null)
            group by metadata->>'page'
        ),
        overall_stats as (
            select
                count(*) as total_events,
                count(distinct user_id) as unique_users
            from events
            where (timestamp >= $1::timestamp or $1::timestamp is null)
                and (timestamp < $2::timestamp or $2::timestamp is null)
                and (type_id = $3::uuid or $3::uuid is null)
        ),
        top_pages as (
            select page, page_events
            from page_stats
            order by page_events desc
            limit 10
        )
        select
            total_events,
            unique_users,
            coalesce(
                (select json_object_agg(page, page_events) from top_pages),
                '{}'::json
            ) as top_pages
        from overall_stats
    `

	err := s.db.QueryRowx(query, data.From, data.To, data.TypeID).StructScan(&stats)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}
