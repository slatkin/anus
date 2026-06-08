package miniflux

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(serverURL, apiKey string, allowInvalidCerts bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: allowInvalidCerts},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}
	return &Client{
		baseURL:    serverURL,
		apiKey:     apiKey,
		httpClient: client,
	}
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, path)

	var reqBody io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Auth-Token", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "anus-go/0.1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("api error: %s (status: %d)", string(respBytes), resp.StatusCode)
	}

	return respBytes, nil
}

// fixProxyURLs rewrites image proxy URLs that Miniflux generates with
// "http://localhost" when its BASE_URL env var is not configured. We replace
// the localhost origin with the actual server URL so images load correctly.
func (c *Client) fixProxyURLs(entries []FeedEntry) {
	for i := range entries {
		entries[i].Content = strings.ReplaceAll(
			entries[i].Content,
			"http://localhost/proxy/",
			c.baseURL+"/proxy/",
		)
	}
}

func (c *Client) GetUnreadEntries(limit, offset int) ([]FeedEntry, int, error) {
	path := fmt.Sprintf("/v1/entries?status=unread&order=published_at&direction=desc&limit=%d&offset=%d", limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}

	var result FeedEntriesResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, 0, err
	}
	c.fixProxyURLs(result.Entries)
	return result.Entries, result.Total, nil
}

func (c *Client) GetReadEntries(since time.Time, limit, offset int) ([]FeedEntry, int, error) {
	path := fmt.Sprintf("/v1/entries?status=read&after=%d&order=published_at&direction=desc&limit=%d&offset=%d",
		since.Unix(), limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}
	var result FeedEntriesResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, 0, err
	}
	c.fixProxyURLs(result.Entries)
	return result.Entries, result.Total, nil
}

func (c *Client) GetStarredEntries(limit, offset int) ([]FeedEntry, int, error) {
	path := fmt.Sprintf("/v1/entries?starred=true&order=published_at&direction=desc&limit=%d&offset=%d", limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}

	var result FeedEntriesResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, 0, err
	}
	c.fixProxyURLs(result.Entries)
	return result.Entries, result.Total, nil
}

func (c *Client) ChangeEntryReadStatus(entryIDs []int, status ReadStatus) error {
	req := UpdateEntriesRequest{
		Status:   string(status),
		EntryIDs: entryIDs,
	}
	_, err := c.doRequest("PUT", "/v1/entries", req)
	return err
}

func (c *Client) ToggleStarred(entryID int) error {
	path := fmt.Sprintf("/v1/entries/%d/bookmark", entryID)
	_, err := c.doRequest("PUT", path, nil)
	return err
}

func (c *Client) SaveEntry(entryID int) error {
	path := fmt.Sprintf("/v1/entries/%d/save", entryID)
	// Original Rust used POST for save
	_, err := c.doRequest("POST", path, nil)
	return err
}

func (c *Client) MarkAllAsRead(entryIDs []int) error {
	return c.ChangeEntryReadStatus(entryIDs, ReadStatusRead)
}

func (c *Client) RefreshAllFeeds() error {
	_, err := c.doRequest("PUT", "/v1/feeds/refresh", nil)
	return err
}

func (c *Client) SearchEntries(query string, limit, offset int) ([]FeedEntry, int, error) {
	path := fmt.Sprintf("/v1/entries?search=%s&order=published_at&direction=desc&limit=%d&offset=%d",
		url.QueryEscape(query), limit, offset)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, 0, err
	}
	var result FeedEntriesResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, 0, err
	}
	c.fixProxyURLs(result.Entries)
	return result.Entries, result.Total, nil
}

func (c *Client) FetchOriginalContent(entryID int) (string, error) {
	path := fmt.Sprintf("/v1/entries/%d/fetch-content", entryID)
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return "", err
	}
	var result OriginalContentResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}
