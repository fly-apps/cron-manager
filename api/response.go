package api

import (
	"encoding/json"
	"net/http"

	"log"
)

type errRes struct {
	Error string `json:"error"`
}

type Response struct {
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func renderJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("failed to write json response: %s", err)
	}
}

func renderErr(w http.ResponseWriter, err error) {
	renderJSON(w, errRes{Error: err.Error()}, http.StatusInternalServerError)
}
