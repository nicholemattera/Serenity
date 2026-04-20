package repository

import "fmt"

type Pagination struct {
	Limit  int
	Offset int
}

type Page[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// paginateQuery appends LIMIT/OFFSET placeholders to query when p is non-nil.
// args must contain all existing query arguments so placeholder numbers are correct.
func paginateQuery(query string, args []any, p *Pagination) (string, []any) {
	if p == nil {
		return query, args
	}
	n := len(args) + 1
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", n, n+1)
	return query, append(args, p.Limit, p.Offset)
}

// pageResult constructs a Page, setting Limit/Offset only when p is non-nil.
func pageResult[T any](data []T, total int, p *Pagination) *Page[T] {
	page := &Page[T]{Data: data, Total: total}
	if p != nil {
		page.Limit = p.Limit
		page.Offset = p.Offset
	}
	return page
}
