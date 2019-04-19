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

	"github.com/gdotgordon/locator-demo/locator/locking"
	"github.com/gdotgordon/locator-demo/locator/store"
	"github.com/gdotgordon/locator-demo/locator/types"
	"github.com/tidwall/gjson"
)

// Geolocator has a single method, namely to look up the x,y for the address.
type Geolocator interface {
	Locate(context.Context, types.AddressRequest) (*types.AddressResponse, error)
}

const (
	// CensusURL is the location of the geolocator service
	CensusURL = "https://geocoding.geo.census.gov/geocoder/locations/address?"

	// CensusStdPrm is the query param string for the URL
	CensusStdPrm = "&benchmark=9&format=json"
)

// CensusGeolocator uses the free service at the US Census bureau.
type CensusGeolocator struct {
	client     *http.Client
	store      store.Store
	useLocking bool
}

// New creates a new CensusGeolocator
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

// Locate does a geolocation lookup.  At the end (in defer) gather up some
// stats and store them in redis, triggering key events.
func (cl *CensusGeolocator) Locate(ctx context.Context,
	reqAddr types.AddressRequest) (*types.AddressResponse, error) {
	start := time.Now()
	var err error

	// Here we invoke the function that sets the redis keys that
	// will trigger notifications in the analyzer.
	defer func() {
		serr := err
		cl.sendStats(start, serr)
	}()

	if reqAddr.StructureNumber == "" || reqAddr.Street == "" {
		log.Printf("request invalid: missing unit number")
		err = errors.New("Structure number and Street are required")
		return nil, err
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
	reqURL := CensusURL + qs
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		log.Printf("error creating request '%s': %v\n", reqURL, err)
		return nil, err
	}
	req.Header.Add("Content-type", "application/json")
	req = req.WithContext(ctx)
	resp, err := cl.client.Do(req)
	if err != nil {
		log.Printf("error opening '%s': %v\n", reqURL, err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("location lookup failed '%s': %v\n", reqURL, err)
		err = fmt.Errorf("HTTP status %d : %s", resp.StatusCode,
			http.StatusText(resp.StatusCode))
		return nil, err
	}
	ct := resp.Header.Get("Content-type")
	if !strings.HasPrefix(ct, "application/json") {
		err = fmt.Errorf("Unexpected content type '%s,", ct)
		return nil, err
	}

	var ar types.AddressResponse
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("Error reading repsonse '%v,", err)
		return nil, err
	}

	// The gjson package turns out to be far less cumbersome in extracting
	// fields from a complex JSON object, compared to encoding/json.
	js := string(b)
	if !gjson.Valid(js) {
		err = fmt.Errorf("Invalid JSON")
		return nil, err
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

// Here is where all the redis keys are set.  The store object (cl.store)
// is the encapsulation of the actual redis calls (see store/store.go).
//
// Note, the object locking, discussed in the writeup, is not enabled
// here, due to the weakness of the Redis-suggested algorithm, plus it
// is not needed given the semantics of the parameters in terms of
// the order thy are received.  But you may enable it, and it will
// work fine for reasonably small numbers of concurrent requests.
func (cl *CensusGeolocator) sendStats(start time.Time, gerr error) {
	var err error
	var lock *locking.Lock
	if cl.useLocking {
		lock, err = cl.store.AcquireLock()
		if err != nil {
			log.Printf("error acquiring lock, stats not saved: %v", err)
			return
		}
	}

	// Store the parameters of interest.
	if err = cl.store.StoreLatency(time.Now().Sub(start)); err != nil {
		log.Printf("error storing latency, skipped: %v", err)
	}

	if gerr != nil {
		if err = cl.store.AddError(); err != nil {
			log.Printf("error storing error, skipped: %v", err)
		}
	} else {
		if err = cl.store.AddSuccess(); err != nil {
			log.Printf("error storing error, skipped: %v", err)
		}
	}

	// Meh.
	if cl.useLocking {
		if err = cl.store.Unlock(lock); err != nil {
			log.Printf("error unlocking, stats not saved: %v", err)
		}
	}
}
