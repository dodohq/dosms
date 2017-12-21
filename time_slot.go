package main

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type timeSlot struct {
	ID         int64  `json:"id"`
	StartTime  string `json:"start_time" schema:"start_time"`
	EndTime    string `json:"end_time" schema:"end_time"`
	ProviderID int64  `json:"provider_id" schema:"provider_id"`
}

// POST /api/time_slot
func createNewTimeSlot(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s := timeSlot{}
	if err := ReadRequestBody(r, &s); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if s.StartTime >= s.EndTime {
		http.Error(w, "Invalid Time Slot", 400)
		return
	}

	query := `INSERT INTO time_slots(start_time, end_time, provider_id) VALUES($1, $2, $3) RETURNING id`
	var id int64
	err := dbConn.QueryRow(query, s.StartTime, s.EndTime, s.ProviderID).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	response, _ := json.Marshal(&map[string]int64{"id": id})
	RenderJSON(w, response)
}
