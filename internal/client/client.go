package client

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	baseURL   = "https://intervals.icu"
	userAgent = "icu/0.1.0"
)

type Client struct {
	BaseURL    string
	APIKey     string
	AthleteID  string
	HTTPClient *http.Client
	Debug      bool
}

func New(apiKey, athleteID string, debug bool) *Client {
	return &Client{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		AthleteID: athleteID,
		Debug:     debug,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) athletePath(path string) string {
	return fmt.Sprintf("/api/v1/athlete/%s%s", c.AthleteID, path)
}

func (c *Client) do(method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.BaseURL + path

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.SetBasicAuth("API_KEY", c.APIKey)
	req.Header.Set("User-Agent", userAgent)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if c.Debug {
		slog.Debug("request", "method", method, "url", url)
	}

	var resp *http.Response
	var doErr error
	for attempt := 0; attempt < 3; attempt++ {
		resp, doErr = c.HTTPClient.Do(req)
		if doErr != nil {
			return nil, fmt.Errorf("request failed: %w", doErr)
		}

		if resp.StatusCode == 429 {
			fmt.Fprintf(os.Stderr, "rate limited, retrying in %ds...\n", (attempt+1)*2)
			resp.Body.Close()
			time.Sleep(time.Duration((attempt+1)*2) * time.Second)
			// rebuild request since body may be consumed
			req, err = http.NewRequest(method, url, body)
			if err != nil {
				return nil, err
			}
			req.SetBasicAuth("API_KEY", c.APIKey)
			req.Header.Set("User-Agent", userAgent)
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}
			continue
		}
		break
	}

	if c.Debug {
		slog.Debug("response", "status", resp.StatusCode)
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, newAPIError(resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	return resp, nil
}

func (c *Client) Get(path string) ([]byte, error) {
	resp, err := c.do("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Post(path string, jsonBody []byte) ([]byte, error) {
	resp, err := c.do("POST", path, bytes.NewReader(jsonBody), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Put(path string, jsonBody []byte) ([]byte, error) {
	resp, err := c.do("PUT", path, bytes.NewReader(jsonBody), "application/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Delete(path string) ([]byte, error) {
	resp, err := c.do("DELETE", path, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Download(path string) (io.ReadCloser, error) {
	resp, err := c.do("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// DownloadWithResponse returns the full *http.Response so callers can inspect
// headers (e.g. Content-Disposition) before reading the body.
func (c *Client) DownloadWithResponse(path string) (*http.Response, error) {
	return c.do("GET", path, nil, "")
}

func (c *Client) Upload(path string, filePath string, extraFields map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return nil, err
	}

	for k, v := range extraFields {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	mw.Close()

	resp, err := c.do("POST", path, &buf, mw.FormDataContentType())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) AthletePath(path string) string {
	return c.athletePath(path)
}
