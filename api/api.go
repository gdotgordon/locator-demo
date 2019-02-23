package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gdotgordon/locator-demo/locator"
	"github.com/gdotgordon/locator-demo/store"
	"github.com/gdotgordon/locator-demo/types"
	"github.com/gorilla/mux"
)

type Api struct {
	loc locator.Locator
}

func Init(ctx context.Context, r *mux.Router, store store.Store) error {
	ap := Api{}
	r.HandleFunc("/v1/status", wrapContext(ctx, ap.getStatus)).Methods("GET")
	r.HandleFunc("/v1/lookup", wrapContext(ctx, ap.lookup)).Methods("POST")
	ap.loc = locator.New(30, store)
	return nil
}

func (a *Api) getStatus(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	sr := types.StatusResponse{Status: "service up and running"}
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

func (a *Api) lookup(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !strings.HasPrefix(r.Header.Get("Content-type"), "application/json") {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"Status\": \"Bad Request\"}"))
	}
	var req types.AddressRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"Status\": \"Bad Request\"}"))
		return
	}

	resp, err := a.loc.Locate(r.Context(), req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusBadRequest)
		msg := fmt.Sprintf("{\"status\": \"bad request, error: %s\"}", err)
		w.Write([]byte(msg))
		return
	}
	if resp.Zip == "" {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("{\"status\": \"address not located\"}"))
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
