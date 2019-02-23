// Package geolocator implements the geocoding lookups.  It reords statistics
// such as latency and stores them in the store (which at runtime in configured
// to Redis), and these will be received as events by the Analyzer.  It also
// defines the generic Geolocator interface, and the implmentaiton we are using
// here, the one from the US Census Bureau.
package geolocator

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gdotgordon/locator-demo/store"
	"github.com/gdotgordon/locator-demo/types"
	"github.com/tidwall/gjson"
)

type Geolocator interface {
	Locate(context.Context, types.AddressRequest) (*types.AddressResponse, error)
}

const (
	CensusURL    = "https://geocoding.geo.census.gov/geocoder/locations/address?"
	CensusStdPrm = "&benchmark=9&format=json"
)

type CensusGeolocator struct {
	client *http.Client
	store  store.Store
}

func New(connTimeout int, store store.Store) Geolocator {
	// The one client is thread safe for use by the scanners.
	// Postman seems to complain about certificates, but no one else!
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	if connTimeout > 0 {
		client.Timeout = time.Duration(connTimeout) * time.Second
	}
	return &CensusGeolocator{client: client, store: store}
}

func (cl *CensusGeolocator) Locate(ctx context.Context,
	reqAddr types.AddressRequest) (*types.AddressResponse, error) {
	start := time.Now()
	defer func() {
		cl.store.StoreLatency(time.Now().Sub(start))
	}()
	if reqAddr.StructureNumber == "" || reqAddr.Street == "" {
		return nil, errors.New("Structure number and Street are required")
	}

	// Set up the request URL based on the request objects passed in.
	var buf bytes.Buffer
	buf.WriteString("street=")
	buf.WriteString(reqAddr.StructureNumber)
	buf.WriteByte(' ')
	buf.WriteString(reqAddr.Street)
	if reqAddr.City != "" {
		buf.WriteString("&city=")
		buf.WriteString(reqAddr.City)
	}
	if reqAddr.State != "" {
		buf.WriteString("&state=")
		buf.WriteString(reqAddr.State)
	}
	if reqAddr.Zip != "" {
		buf.WriteString("&zip=")
		buf.WriteString(reqAddr.Zip)
	}
	buf.WriteString(CensusStdPrm)

	// Bleh: the web site doesn't like Go's escaped '='
	// qs = url.QueryEscape(buf.String())
	qs := strings.Replace(buf.String(), " ", "+", -1)
	reqUrl := CensusURL + qs
	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	if err != nil {
		log.Printf("error creating request '%s': %v\n", reqUrl, err)
		return nil, err
	}
	req.Header.Add("Content-type", "application/json")
	req = req.WithContext(ctx)
	resp, err := cl.client.Do(req)
	if err != nil {
		log.Printf("error opening '%s': %v\n", reqUrl, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("location lookup failed '%s': %v\n", reqUrl, err)
		return nil, fmt.Errorf("HTTP status %d : %s", resp.StatusCode,
			http.StatusText(resp.StatusCode))
	}
	ct := resp.Header.Get("Content-type")
	if !strings.HasPrefix(ct, "application/json") {
		return nil, fmt.Errorf("Unexpected content type '%s,", ct)
	}

	var ar types.AddressResponse
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading repsonse '%v,", err)
	}

	// The gjson package turns out to be far less cumbersome in extracting
	// fields from a complex JSON object, compared to encoding/json.
	js := string(b)
	if !gjson.Valid(js) {
		return nil, fmt.Errorf("Invalid JSON")
	}

	// If no matches, just return an empty struct, which sifgnifies "not found".
	// This is not a system malfucntion (and we get HTTP 200), so no error.
	rj := gjson.Get(js, "result.addressMatches.#")
	if rj.Int() == 0 {
		return &ar, nil
	}

	// Use the first match.
	rj = gjson.Get(js, "result.addressMatches.0.addressComponents.zip")
	ar.Zip = rj.String()
	rj = gjson.Get(js, "result.addressMatches.0.coordinates")
	m := rj.Map()
	ar.Coordinates.X = m["x"].Float()
	ar.Coordinates.Y = m["y"].Float()
	return &ar, nil
}
