package geolocator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gdotgordon/locator-demo/locator/types"
)

type NoOpStore struct {
}

func (nos NoOpStore) StoreLatency(d time.Duration) error {
	fmt.Printf("Storing duration: %s\n", d.String())
	return nil
}

func (nos NoOpStore) AddSuccess() error {
	return nil
}

func (nos NoOpStore) AddError() error {
	return nil
}

func (nos NoOpStore) ClearDatabase() error {
	return nil
}

func TestLookup(t *testing.T) {
	l := New(30, NoOpStore{})

	for _, test := range []struct {
		rq types.AddressRequest
		rs types.AddressResponse
		e  string
	}{
		{
			rq: types.AddressRequest{StructureNumber: "4600", Street: "Silver Hill Rd",
				City: "Suitland", State: "MD", Zip: "20746"},
			rs: types.AddressResponse{Zip: "20746",
				Coordinates: types.Coords{X: -76.92691, Y: 38.846542}},
		},
		{
			rq: types.AddressRequest{StructureNumber: "1500", Street: "Red Rover St",
				City: "Austin", State: "TX"},
			rs: types.AddressResponse{Zip: "78701",
				Coordinates: types.Coords{X: -97.734770, Y: 30.275732}},
		},
		{
			rq: types.AddressRequest{StructureNumber: "46", Street: "Blue Bayou Ln",
				City: "San Ramon", State: "CA"},
			rs: types.AddressResponse{},
		},
		{
			rq: types.AddressRequest{Street: "Blue Bayou Ln",
				City: "San Ramon", State: "CA"},
			rs: types.AddressResponse{},
			e:  "Structure number and Street are required",
		},
	} {
		resp, err := l.Locate(context.Background(), test.rq)
		if test.e != "" {
			if err == nil {
				t.Fatalf("Did not get expected error: %s", test.e)
			} else if test.e != err.Error() {
				t.Fatalf("Expected error '%s'', got '%s'", test.e, err.Error())
			}
			continue
		}
		if test.e == "" && err != nil {
			t.Fatalf("Got unexpected error: %v", err)
		}
		if resp.Zip != test.rs.Zip {
			t.Fatalf("Expected zip '%s'', got '%s'", test.rs.Zip, resp.Zip)
		}
		if resp.Coordinates.X != test.rs.Coordinates.X {
			t.Fatalf("Expected X %f, got %f", test.rs.Coordinates.X, resp.Coordinates.X)
		}
		if resp.Coordinates.Y != test.rs.Coordinates.Y {
			t.Fatalf("Expected Y %f, got %f", test.rs.Coordinates.Y, resp.Coordinates.Y)
		}
	}
}
