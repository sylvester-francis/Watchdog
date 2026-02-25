package handlers

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

// maxPageSize is the maximum number of items a list endpoint will return in a
// single response (H-020). Any client-supplied per_page / limit value is
// clamped to this ceiling.
const maxPageSize = 100

// defaultPageSize is used when the client omits a per_page / limit param.
const defaultPageSize = 50

// hardQueryLimit is the absolute maximum number of rows any single list query
// may return, even for endpoints that don't expose pagination to the client.
// This prevents runaway queries on large datasets.
const hardQueryLimit = 1000

// clampPageSize reads a "per_page" or "limit" query param from the request and
// clamps it to [1, maxPageSize]. Returns defaultPageSize when the param is
// absent or invalid.
func clampPageSize(c echo.Context) int {
	raw := c.QueryParam("per_page")
	if raw == "" {
		raw = c.QueryParam("limit")
	}
	if raw == "" {
		return defaultPageSize
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return defaultPageSize
	}
	if n > maxPageSize {
		return maxPageSize
	}
	return n
}
