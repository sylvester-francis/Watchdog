package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// APIClient wraps HTTP calls to the WatchDog hub API.
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func newClient(cfg *CLIConfig) *APIClient {
	return &APIClient{
		baseURL: cfg.HubURL + "/api/v1",
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *APIClient) get(path string) ([]byte, int, error) {
	return c.do(http.MethodGet, path, nil)
}

func (c *APIClient) post(path string, body any) ([]byte, int, error) {
	return c.do(http.MethodPost, path, body)
}

func (c *APIClient) put(path string, body any) ([]byte, int, error) {
	return c.do(http.MethodPut, path, body)
}

func (c *APIClient) delete(path string) ([]byte, int, error) {
	return c.do(http.MethodDelete, path, nil)
}

func (c *APIClient) do(method, path string, body any) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return respBody, resp.StatusCode, nil
}
