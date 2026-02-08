package model

import "time"

type Allocation struct {
	ID        int64     `json:"id"`
	App       string    `json:"app"`
	Instance  string    `json:"instance"`
	Service   string    `json:"service"`
	Port      int       `json:"port"`
	CreatedAt time.Time `json:"created_at"`
}

type AllocateRequest struct {
	App      string `json:"app"`
	Instance string `json:"instance"`
	Service  string `json:"service"`
	Port     int    `json:"port,omitempty"`
}

type ReleaseRequest struct {
	App      string `json:"app,omitempty"`
	Instance string `json:"instance,omitempty"`
	Service  string `json:"service,omitempty"`
	Port     int    `json:"port,omitempty"`
}

type PortStatus struct {
	Port      int         `json:"port"`
	Available bool        `json:"available"`
	Holder    *Allocation `json:"holder,omitempty"`
}

type ErrorResponse struct {
	Error  string      `json:"error"`
	Holder *Allocation `json:"holder,omitempty"`
}
