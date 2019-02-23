package types

const (
	KeyPrefix  = "locator:"
	LatencyKey = KeyPrefix + "latency"
)

type StatusResponse struct {
	Status string `json:"status"`
}

type AddressRequest struct {
	StructureNumber string `json:"struct_number"`
	Street          string `json:"street"`
	City            string `json:"city,omitempty"`
	State           string `json:"state,omitempty"`
	Zip             string `json:"zip,omitempty"`
}

type Coords struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type AddressResponse struct {
	Zip         string `json:"zip"`
	Coordinates Coords `json:"coordinates"`
}
