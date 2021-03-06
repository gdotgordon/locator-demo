// Package types describes the constants and data types used
// for requests and event processing.
package types

const (
	KeyPrefix  = "locator:"
	LatencyKey = KeyPrefix + "latency"
	SuccessKey = KeyPrefix + "success"
	ErrorKey   = KeyPrefix + "error"
)

// StatusResponse is the response to astatus check (ping).
type StatusResponse struct {
	Status string `json:"status"`
}

// StatsResponse is the response to a call to get accumulated statistics.
type StatsResponse struct {
	Success      int64  `json:"success"`
	Error        int64  `json:"failure"`
	LatencyCount int64  `json:"latency_events"`
	Latency      string `json:"latency"`
}
