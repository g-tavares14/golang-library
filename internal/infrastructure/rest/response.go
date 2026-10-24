package rest

import (
	"encoding/json"
	"errors"
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

// maxBodyBytes é o tamanho máximo aceito para o corpo de uma requisição.
const maxBodyBytes = 1 << 20 // 1 MiB

// decodeJSON lê o corpo da requisição como JSON em destination. Se o corpo for inválido ou passar de
// maxBodyBytes, escreve o erro (400 ou 413) e devolve false; o handler só precisa retornar.
func decodeJSON(responseWriter http.ResponseWriter, request *http.Request, destination any) bool {
	request.Body = http.MaxBytesReader(responseWriter, request.Body, maxBodyBytes)

	if err := json.NewDecoder(request.Body).Decode(destination); err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			writeError(responseWriter, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeError(responseWriter, http.StatusBadRequest, "invalid request body")
		return false
	}

	return true
}
