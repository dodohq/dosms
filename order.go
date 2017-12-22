package main

import (
	"net/http"
	"strconv"

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

	RenderJSON(w, map[string]int64{"id": ID})
}

// GET /api/order/:provider_id
func getOrdersByProvider(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	providerID, _ := strconv.Atoi(ps.ByName("provider_id"))
	query := `SELECT id, customer_name, contact_number, delivery_date, provider_id FROM orders WHERE provider_id = $1 AND NOT deleted`
	orders, err := fetchOrders(query, providerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	RenderJSON(w, map[string][]*order{"orders": orders})
}

func fetchOrders(query string, args ...interface{}) ([]*order, error) {
	rows, err := dbConn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*order, 0)
	for rows.Next() {
		o := new(order)
		err = rows.Scan(&o.ID, &o.CustomerName, &o.ContactNumber, &o.DeliveryDate, &o.ProviderID)
		if err != nil {
			return nil, err
		}

		results = append(results, o)
	}

	return results, nil
}
