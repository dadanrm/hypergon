package renderer

import (
	"encoding/json"
	"log"
	"net/http"
)

// JSONRenderer object.
type JSONRenderer struct{}

// JSONRenderer constructor.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// JSON functions returns a json response.
func (r *JSONRenderer) JSON(w http.ResponseWriter, data any) error {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("Error rendering template:", err)
		return err
	}

	return nil
}
