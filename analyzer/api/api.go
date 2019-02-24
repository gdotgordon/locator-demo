// Pacakge api is the endpoint implementation for the analyzer service.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gdotgordon/locator-demo/analyzer/receiver"
	"github.com/gdotgordon/locator-demo/analyzer/types"
	"github.com/gorilla/mux"
)

type Api struct {
	receiver *receiver.Receiver
}

func Init(ctx context.Context, r *mux.Router,
	receiver *receiver.Receiver) error {
	ap := Api{receiver: receiver}
	r.HandleFunc("/v1/status", wrapContext(ctx, ap.getStatus)).Methods("GET")
	r.HandleFunc("/v1/statistics", wrapContext(ctx, ap.getStatistics)).Methods("GET")
	r.HandleFunc("/v1/reset", wrapContext(ctx, ap.reset)).Methods("GET")
	return nil
}

// Liveness check
func (a *Api) getStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	sr := types.StatusResponse{Status: "analyzer service up and running"}
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(sr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("JSON encode error"))
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write(b.Bytes())
}

// Gets the statistics accumulated by the Redis event receiver.
func (a *Api) getStatistics(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	resp, err := a.receiver.GetStats()
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("{\"status\": \"retrieving stats, error: %s\"}", err)
		w.Write([]byte(msg))
		return
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(resp)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{\"status\": \"json unmarshal error\"}"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)

	w.Write(buf.Bytes())
}

// Clears the redis database.
func (a *Api) reset(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if err := a.receiver.Reset(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
}

func wrapContext(ctx context.Context, hf http.HandlerFunc) http.HandlerFunc {
	cw := contextWrapper{ctx: ctx, hf: hf}
	return cw.wrap
}

type contextWrapper struct {
	ctx context.Context
	hf  http.HandlerFunc
}

func (cw *contextWrapper) wrap(w http.ResponseWriter, r *http.Request) {
	rc := r.WithContext(cw.ctx)
	cw.hf(w, rc)
}
