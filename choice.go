package main

import (
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type choice struct {
	TimeSlotID int64     `json:"time_slot_id" schema:"time_slot_id"`
	OrderID    int64     `json:"order_id" schema:"order_id"`
	TimeSlot   *timeSlot `json:"time_slot,omitempty"`
	Order      *order    `json:"order,omitempty"`
}

// POST /api/choice
func createNewChoice(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c := choice{}
	if err := ReadRequestBody(r, &c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	slots, err := fetchTimeSlots(`SELECT id, start_time, end_time, provider_id FROM time_slots WHERE id = $1 AND NOT deleted`, c.TimeSlotID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(slots) == 0 {
		http.Error(w, "Invalid slot", 400)
		return
	}

	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, delivery_date, provider_id FROM orders WHERE id = $1 AND NOT deleted`, c.OrderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(orders) == 0 {
		http.Error(w, "Invalid order", 400)
		return
	}

	query := `INSERT INTO choices(time_slot_id, order_id) VALUES($1, $2) RETURNING time_slot_id, order_id`
	var timeSlotID int64
	var orderID int64
	err = dbConn.QueryRow(query, c.TimeSlotID, c.OrderID).Scan(&timeSlotID, &orderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	RenderJSON(w, map[string]int64{"time_slot_id": timeSlotID, "order_id": orderID})
}

// GET /api/choice/:order_id
func getChoicesByOrder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	orderID, _ := strconv.Atoi(ps.ByName("order_id"))

	query := `SELECT time_slot_id, order_id FROM choices WHERE order_id = $1 AND NOT deleted`
	choices, err := fetchChoices(query, orderID)
	if err != nil {

		http.Error(w, err.Error(), 500)
		return
	}

	RenderJSON(w, map[string][]*choice{"choices": choices})
}

func fetchChoices(query string, args ...interface{}) ([]*choice, error) {
	rows, err := dbConn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*choice, 0)
	for rows.Next() {
		c := new(choice)
		err = rows.Scan(&c.TimeSlotID, &c.OrderID)
		if err != nil {
			return nil, err
		}

		slots, err := fetchTimeSlots(`SELECT id, start_time, end_time, provider_id FROM time_slots WHERE id = $1 AND NOT deleted`, c.TimeSlotID)
		if err != nil {
			return nil, err
		}
		c.TimeSlot = slots[0]

		orders, err := fetchOrders(`SELECT id, customer_name, contact_number, delivery_date, provider_id FROM orders WHERE id = $1 AND NOT deleted`, c.OrderID)
		if err != nil {
			return nil, err
		}
		c.Order = orders[0]

		results = append(results, c)
	}

	return results, nil
}
