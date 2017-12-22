package main

import (
	"net/http"

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

	query := `INSERT INTO choices(time_slot_id, order_id) VALUES($1, $2) RETURNING time_slot_id, order_id`
	var timeSlotID int64
	var orderID int64
	err := dbConn.QueryRow(query, c.TimeSlotID, c.OrderID).Scan(&timeSlotID, &orderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	RenderJSON(w, map[string]int64{"time_slot_id": timeSlotID, "order_id": orderID})
}
