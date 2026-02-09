package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nfedorov/port_server/internal/model"
	"github.com/nfedorov/port_server/internal/store"
)

type Client struct {
	base   string
	client *http.Client
}

func New(addr string) *Client {
	return &Client{
		base:   "http://" + addr,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Health() error {
	resp, err := c.client.Get(c.base + "/healthz")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) Allocate(req model.AllocateRequest) (*model.Allocation, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	resp, err := c.client.Post(c.base+"/v1/allocations", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var errResp model.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("decode conflict response: %w", err)
		}
		if errResp.Error == "service already allocated" {
			return errResp.Holder, store.ErrServiceAllocated
		}
		if errResp.Error == "port in use on system" {
			return nil, store.ErrPortBusy
		}
		return errResp.Holder, store.ErrPortTaken
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, readError(resp)
	}

	var alloc model.Allocation
	if err := json.NewDecoder(resp.Body).Decode(&alloc); err != nil {
		return nil, fmt.Errorf("decode allocation: %w", err)
	}
	return &alloc, nil
}

func (c *Client) List(f store.Filter) ([]model.Allocation, error) {
	u, _ := url.Parse(c.base + "/v1/allocations")
	q := u.Query()
	if f.App != "" {
		q.Set("app", f.App)
	}
	if f.Instance != "" {
		q.Set("instance", f.Instance)
	}
	if f.Service != "" {
		q.Set("service", f.Service)
	}
	u.RawQuery = q.Encode()

	resp, err := c.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, readError(resp)
	}

	var allocs []model.Allocation
	if err := json.NewDecoder(resp.Body).Decode(&allocs); err != nil {
		return nil, fmt.Errorf("decode allocations: %w", err)
	}
	return allocs, nil
}

func (c *Client) ReleaseByID(id int64) error {
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/v1/allocations/%d", c.base, id), nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return store.ErrNotFound
	}
	if resp.StatusCode != 200 {
		return readError(resp)
	}
	return nil
}

func (c *Client) ReleaseByFilter(rel model.ReleaseRequest) (int64, error) {
	body, err := json.Marshal(rel)
	if err != nil {
		return 0, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("DELETE", c.base+"/v1/allocations", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, readError(resp)
	}

	var result map[string]int64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return result["deleted"], nil
}

func (c *Client) CheckPort(port int) (*model.PortStatus, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/v1/ports/%d", c.base, port))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, readError(resp)
	}

	var status model.PortStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode port status: %w", err)
	}
	return &status, nil
}

func readError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(data))
}
