package memos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

func NewClient(baseURL string, apiToken string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if parsed.Scheme == "" {
		return nil, fmt.Errorf("base URL must include scheme")
	}

	trimmed := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:  trimmed,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) SearchMemos(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	filter, err := BuildMemoFilter(req)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("pageSize", strconv.Itoa(limit))
	if filter != "" {
		query.Set("filter", filter)
	}
	if req.OrderBy != "" {
		query.Set("orderBy", req.OrderBy)
	}
	if req.ShowDeleted {
		query.Set("showDeleted", "true")
	}
	if req.PageToken != "" {
		query.Set("pageToken", req.PageToken)
	} else if req.Offset > 0 {
		query.Set("pageToken", fmt.Sprintf("offset=%d", req.Offset))
	}

	endpoint, err := c.buildURL("/api/v1/memos", query)
	if err != nil {
		return nil, err
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var response SearchResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return &response, nil
}

func (c *Client) GetMemo(ctx context.Context, uid string) (*Memo, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, fmt.Errorf("memo uid is required")
	}
	endpoint, err := c.buildURL("/api/v1/memos/"+url.PathEscape(uid), nil)
	if err != nil {
		return nil, err
	}
	respBody, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var memo Memo
	if err := json.Unmarshal(respBody, &memo); err != nil {
		return nil, fmt.Errorf("decode memo: %w", err)
	}
	return &memo, nil
}

func (c *Client) CreateMemo(ctx context.Context, req CreateMemoRequest) (*Memo, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, fmt.Errorf("content is required")
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = "PRIVATE"
	}

	normalized, err := normalizeVisibility(visibility)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"state":      "STATE_UNSPECIFIED",
		"content":    req.Content,
		"visibility": normalized,
	}
	if req.CreateTime != nil {
		payload["createTime"] = req.CreateTime.Format(time.RFC3339)
	}
	if req.UpdateTime != nil {
		payload["updateTime"] = req.UpdateTime.Format(time.RFC3339)
	}
	if req.Pinned != nil {
		payload["pinned"] = *req.Pinned
	}

	endpoint, err := c.buildURL("/api/v1/memos", nil)
	if err != nil {
		return nil, err
	}
	respBody, err := c.doRequest(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var memo Memo
	if err := json.Unmarshal(respBody, &memo); err != nil {
		return nil, fmt.Errorf("decode memo: %w", err)
	}
	return &memo, nil
}

func (c *Client) UpdateMemo(ctx context.Context, uid string, req UpdateMemoRequest) (*Memo, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, fmt.Errorf("memo uid is required")
	}

	payload := map[string]any{
		"state": "STATE_UNSPECIFIED",
	}

	if req.Content != nil {
		payload["content"] = *req.Content
	}
	if req.Visibility != nil {
		normalized, err := normalizeVisibility(*req.Visibility)
		if err != nil {
			return nil, err
		}
		payload["visibility"] = normalized
	}
	if req.Pinned != nil {
		payload["pinned"] = *req.Pinned
	}

	if len(payload) == 1 {
		return nil, fmt.Errorf("at least one field must be provided for update")
	}

	endpoint, err := c.buildURL("/api/v1/memos/"+url.PathEscape(uid), nil)
	if err != nil {
		return nil, err
	}
	respBody, err := c.doRequest(ctx, http.MethodPatch, endpoint, payload)
	if err != nil {
		return nil, err
	}
	var memo Memo
	if err := json.Unmarshal(respBody, &memo); err != nil {
		return nil, fmt.Errorf("decode memo: %w", err)
	}
	return &memo, nil
}

func (c *Client) DeleteMemo(ctx context.Context, uid string, force bool) error {
	if strings.TrimSpace(uid) == "" {
		return fmt.Errorf("memo uid is required")
	}

	query := url.Values{}
	if force {
		query.Set("force", "true")
	}

	endpoint, err := c.buildURL("/api/v1/memos/"+url.PathEscape(uid), query)
	if err != nil {
		return err
	}
	_, err = c.doRequest(ctx, http.MethodDelete, endpoint, nil)
	return err
}

func (c *Client) buildURL(path string, query url.Values) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}
	rel, err := url.Parse(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	endpoint := base.ResolveReference(rel)
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint.String(), nil
}

func (c *Client) doRequest(ctx context.Context, method string, endpoint string, body any) ([]byte, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		payload = bytes.NewBuffer(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if strings.TrimSpace(c.apiToken) != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(respBody))}
		if apiErr.Body == "" {
			apiErr.Body = fmt.Sprintf("memos API error: %s", resp.Status)
		}
		return nil, apiErr
	}

	return respBody, nil
}
