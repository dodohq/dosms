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

// GET /api/provider
func getAllProviders(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := `SELECT id, title, contact_number FROM providers WHERE deleted <> TRUE`
	providers, err := fetch(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	response, _ := json.Marshal(&map[string][]*provider{"providers": providers})
	RenderJSON(w, response)
}

func fetch(query string, args ...interface{}) ([]*provider, error) {
	rows, err := dbConn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*provider, 0)
	for rows.Next() {
		t := new(provider)
		err = rows.Scan(&t.ID, &t.Title, &t.ContactNumber)
		if err != nil {
			return nil, err
		}
		results = append(results, t)
	}

	return results, nil
}
