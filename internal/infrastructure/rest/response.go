package rest

import (
	"encoding/json"
	"log"
	"net/http"
)

// errorResponse é o corpo JSON devolvido nas respostas de erro.
type errorResponse struct {
	Error string `json:"error"`
}

// writeJSON escreve responseBody como JSON com o statusCode informado.
func writeJSON(responseWriter http.ResponseWriter, statusCode int, responseBody any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	if err := json.NewEncoder(responseWriter).Encode(responseBody); err != nil {
		log.Println("write json:", err)
	}
}

// writeError escreve message como erro JSON ({"error": "..."}) com o statusCode informado.
func writeError(responseWriter http.ResponseWriter, statusCode int, message string) {
	writeJSON(responseWriter, statusCode, errorResponse{Error: message})
}
