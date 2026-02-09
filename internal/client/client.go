package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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
		client: &http.Client{},
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
	body, _ := json.Marshal(req)
	resp, err := c.client.Post(c.base+"/v1/allocations", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		var errResp model.ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error == "service already allocated" {
			return errResp.Holder, store.ErrServiceAllocated
		}
		return errResp.Holder, store.ErrPortTaken
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, readError(resp)
	}

	var alloc model.Allocation
	json.NewDecoder(resp.Body).Decode(&alloc)
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
	json.NewDecoder(resp.Body).Decode(&allocs)
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
	body, _ := json.Marshal(rel)
	req, _ := http.NewRequest("DELETE", c.base+"/v1/allocations", bytes.NewReader(body))
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
	json.NewDecoder(resp.Body).Decode(&result)
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
	json.NewDecoder(resp.Body).Decode(&status)
	return &status, nil
}

func readError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(data))
}
