package main

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type provider struct {
	ID            int64  `json:"id"`
	Title         string `json:"title" schema:"title"`
	ContactNumber string `json:"contact_number" schema:"contact_number"`
}

// POST /api/provider
func createNewProvider(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	p := provider{}
	if err := ReadRequestBody(r, &p); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	query := `INSERT INTO providers(title, contact_number) VALUES($1, $2) RETURNING id`
	var id int64
	err := dbConn.QueryRow(query, p.Title, p.ContactNumber).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	response, _ := json.Marshal(&map[string]int64{"id": id})
	RenderJSON(w, response)
}
