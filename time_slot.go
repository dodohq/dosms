package main

import (
	"net/http"
	"strconv"

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

	RenderJSON(w, map[string]int64{"id": id})
}

// GET /api/time_slot/:provider_id
func getTimeSlotsByProvider(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	providerID, _ := strconv.Atoi(ps.ByName("provider_id"))
	query := `SELECT id, start_time, end_time, provider_id FROM time_slots WHERE provider_id = $1 AND NOT deleted`
	slots, err := fetchTimeSlots(query, providerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	RenderJSON(w, map[string][]*timeSlot{"slots": slots})
}

// DELETE /api/time_slot/:id
func deleteTimeSlot(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ID, _ := strconv.Atoi(ps.ByName("id"))
	query := `UPDATE time_slots SET deleted = TRUE WHERE id = $1`
	stmt, err := dbConn.Prepare(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	res, err := stmt.Exec(ID)
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

func fetchTimeSlots(query string, args ...interface{}) ([]*timeSlot, error) {
	rows, err := dbConn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*timeSlot, 0)
	for rows.Next() {
		s := new(timeSlot)
		err = rows.Scan(&s.ID, &s.StartTime, &s.EndTime, &s.ProviderID)
		if err != nil {
			return nil, err
		}

		results = append(results, s)
	}

	return results, nil
}
