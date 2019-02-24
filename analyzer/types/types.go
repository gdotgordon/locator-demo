package types

import "time"

const (
	KeyPrefix  = "locator:"
	LatencyKey = KeyPrefix + "latency"
	SuccessKey = KeyPrefix + "success"
	ErrorKey   = KeyPrefix + "error"
)

type StatusResponse struct {
	Status string `json:"status"`
}

type StatsResponse struct {
	Success int64         `json:"success"`
	Failure int64         `json:"failure"`
	Latency time.Duration `json:"latency"`
}
