package handlers

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"

	"github.com/labstack/echo/v4"
)

// ErrOTLPBodyTooLarge indicates the request (or its decompressed form)
// exceeds the per-handler byte cap. Callers map this to HTTP 413.
var ErrOTLPBodyTooLarge = errors.New("otlp body too large")

// ErrOTLPBodyRead indicates a read or decode failure on the request
// body. Callers map this to HTTP 400.
var ErrOTLPBodyRead = errors.New("otlp body read failed")

// readOTLPBody returns the raw OTLP request bytes, transparently
// decompressing gzip-encoded payloads (Content-Encoding: gzip).
//
// The limit applies to the *decompressed* body so that a small
// compressed payload that would expand to gigabytes still gets
// rejected. The wire-side LimitReader uses the same cap, so a peer
// that sends an oversized raw body never reaches the gzip stage.
func readOTLPBody(c echo.Context, maxBytes int64) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(c.Request().Body, maxBytes+1))
	if err != nil {
		return nil, ErrOTLPBodyRead
	}
	if int64(len(raw)) > maxBytes {
		return nil, ErrOTLPBodyTooLarge
	}

	if c.Request().Header.Get("Content-Encoding") != "gzip" {
		return raw, nil
	}

	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrOTLPBodyRead
	}
	defer gz.Close()

	body, err := io.ReadAll(io.LimitReader(gz, maxBytes+1))
	if err != nil {
		return nil, ErrOTLPBodyRead
	}
	if int64(len(body)) > maxBytes {
		return nil, ErrOTLPBodyTooLarge
	}
	return body, nil
}
