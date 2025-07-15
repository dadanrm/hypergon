package renderer

import (
	"encoding/json"
	"net/http"

	"github.com/dadanrm/hypergon"
)

// JSONRenderer object.
type JSONRenderer struct{}

// JSONRenderer constructor.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// JSON functions returns a json response.
func (r *JSONRenderer) JSON(w http.ResponseWriter, data any) hypergon.HypergonError {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(data); err != nil {
		return hypergon.HttpError(http.StatusInternalServerError, err.Error())
	}

	return nil
}
