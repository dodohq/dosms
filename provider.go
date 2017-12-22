package main

import (
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type provider struct {
	ID            int64       `json:"id"`
	Title         string      `json:"title" schema:"title"`
	ContactNumber string      `json:"contact_number" schema:"contact_number"`
	Slots         []*timeSlot `json:"slots"`
	Orders        []*order    `json:"orders"`
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

	RenderJSON(w, map[string]int64{"id": id})
}

// GET /api/provider
func getAllProviders(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	query := `SELECT id, title, contact_number FROM providers WHERE NOT deleted`
	providers, err := fetchProviders(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	for _, p := range providers {
		query := `SELECT id, start_time, end_time, provider_id FROM time_slots WHERE provider_id = $1 AND NOT deleted`
		slots, err := fetchTimeSlots(query, p.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		p.Slots = slots

		query = `SELECT id, customer_name, contact_number, delivery_date, provider_id FROM orders WHERE provider_id = $1 AND NOT deleted`
		orders, err := fetchOrders(query, p.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		p.Orders = orders
	}

	RenderJSON(w, map[string][]*provider{"providers": providers})
}

// GET /api/provider/:id
func getProviderByID(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	query := `SELECT id, title, contact_number FROM providers WHERE id = $1 AND NOT deleted LIMIT 1`
	providers, err := fetchProviders(query, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(providers) == 0 {
		http.Error(w, "Not Found", 404)
		return
	}

	p := providers[0]
	query = `SELECT id, start_time, end_time, provider_id FROM time_slots WHERE provider_id = $1 AND NOT deleted`
	slots, err := fetchTimeSlots(query, p.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	p.Slots = slots

	RenderJSON(w, p)
}

// DELETE /api/provider/:id
func deleteProvider(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, _ := strconv.Atoi(ps.ByName("id"))
	query := `UPDATE providers SET deleted = TRUE WHERE id = $1`
	stmt, err := dbConn.Prepare(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	res, err := stmt.Exec(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if rowsAffected <= 0 {
		http.Error(w, "Not Found", 404)
		return
	}

	RenderJSON(w, map[string]string{})
}

func fetchProviders(query string, args ...interface{}) ([]*provider, error) {
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
