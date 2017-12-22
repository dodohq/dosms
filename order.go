package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type order struct {
	ID            int64  `json:"id"`
	CustomerName  string `json:"customer_name" schema:"customer_name"`
	ContactNumber string `json:"contact_number" schema:"contact_number"`
	DeliveryDate  string `json:"delivery_date" schema:"delivery_date"`
	ProviderID    int64  `json:"provider_id" schema:"provider_id"`
}

// POST /api/order
func createNewOrder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	o := order{}
	if err := ReadRequestBody(r, &o); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	query := `INSERT INTO orders(customer_name, contact_number, delivery_date, provider_id) VALUES($1, $2, $3, $4) RETURNING id`
	var ID int64
	err := dbConn.QueryRow(query, o.CustomerName, o.ContactNumber, o.DeliveryDate, o.ProviderID).Scan(&ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	RenderJSON(w, &map[string]int64{"id": ID})
}
