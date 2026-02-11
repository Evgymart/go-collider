package pagination

import (
	"net/http"
	"strconv"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

type Params struct {
	Page  uint
	Limit uint
}

func ParseFromRequest(r *http.Request) Params {
	query := r.URL.Query()

	page := DefaultPage
	if p, err := strconv.Atoi(query.Get("page")); err == nil && p >= 1 {
		page = p
	}

	limit := DefaultLimit
	if l, err := strconv.Atoi(query.Get("limit")); err == nil && l >= 1 {
		limit = l
		if limit > MaxLimit {
			limit = MaxLimit
		}
	}

	return Params{Page: uint(page), Limit: uint(limit)}
}
