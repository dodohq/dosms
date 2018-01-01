package main

import (
	"encoding/csv"
	"net/http"
	"os"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type order struct {
	ID            int64     `json:"id"`
	CustomerName  string    `json:"customer_name" schema:"customer_name"`
	ContactNumber string    `json:"contact_number" schema:"contact_number"`
	DeliveryDate  string    `json:"delivery_date" schema:"delivery_date"`
	ProviderID    int64     `json:"provider_id" schema:"provider_id"`
	RetriesCount  int64     `json:"retries_count"`
	Choices       []*choice `json:"choices,omitempty"`
	Provider      *provider `json:"provider,omitempty"`
}

// POST /api/order
func createNewOrder(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	o := order{}
	if err := ReadRequestBody(r, &o); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	providers, err := fetchProviders(`
		SELECT id, title, contact_number, EXTRACT(HOUR FROM timezone('UTC', reminder_time))
		FROM providers WHERE id = $1 AND NOT deleted`,
		o.ProviderID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(providers) == 0 {
		http.Error(w, "Invalid provider", 400)
		return
	}

	query := `INSERT INTO orders(customer_name, contact_number, delivery_date, provider_id) VALUES($1, $2, $3, $4) RETURNING id`
	var ID int64
	err = dbConn.QueryRow(query, o.CustomerName, o.ContactNumber, o.DeliveryDate, o.ProviderID).Scan(&ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	slots, err := fetchTimeSlots(`
		SELECT id, EXTRACT(HOUR FROM start_time), EXTRACT(HOUR FROM end_time), provider_id
		FROM time_slots WHERE provider_id = $1 AND NOT deleted ORDER BY start_time ASC
	`, o.ProviderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id, retries_count FROM orders WHERE id = $1 AND NOT deleted`, ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(orders) <= 0 {
		http.Error(w, "Fail to insert", 500)
		return
	}

	o = *orders[0]
	// launch reminder
	o.Provider = providers[0]
	o.Provider.Slots = slots
	go scheduleReminder(&o, stopSignal)

	RenderJSON(w, map[string]int64{"id": ID})
}

// POST /api/order/:provider_id/csv_upload
func newOrdersFromCsv(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	providerID, _ := strconv.Atoi(ps.ByName("provider_id"))
	filePath, err := ReadFileUpload(r, "orders_csv")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	fileReader, err := os.Open(filePath)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer fileReader.Close()

	csvReader := csv.NewReader(fileReader)
	records, err := csvReader.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(records) <= 1 {
		http.Error(w, "Empty Record", 400)
		return
	}

	providers, err := fetchProviders(`
		SELECT id, title, contact_number, EXTRACT(HOUR FROM timezone('UTC', reminder_time))
		FROM providers WHERE id = $1 AND NOT deleted`,
		providerID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(providers) == 0 {
		http.Error(w, "Invalid provider", 400)
		return
	}
	currProvider := providers[0]

	slots, err := fetchTimeSlots(`
		SELECT id, EXTRACT(HOUR FROM start_time), EXTRACT(HOUR FROM end_time), provider_id
		FROM time_slots WHERE provider_id = $1 AND NOT deleted ORDER BY start_time ASC
	`, providerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	currProvider.Slots = slots

	orders := []*order{}

	query := "INSERT INTO orders(customer_name, contact_number, delivery_date, provider_id) VALUES"
	queryParams := []interface{}{}
	for i := 1; i < len(records); i++ {
		qI := i - 1
		query += "($" + strconv.Itoa(qI*4+1) + ", $" + strconv.Itoa(qI*4+2) + ", $" + strconv.Itoa(qI*4+3) + ", $" + strconv.Itoa(qI*4+4) + ")"
		if i < len(records)-1 {
			query += ",\n"
		}

		queryParams = append(queryParams, records[i][0], records[i][1], records[i][2], providerID)
		orders = append(orders, &order{
			ContactNumber: records[i][1],
		})
	}

	stmt, err := dbConn.Prepare(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_, err = stmt.Exec(queryParams...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for i, o := range orders {
		newlyInsertedOrders, err := fetchOrders(`
			SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id, retries_count
			FROM orders WHERE contact_number = $1 AND NOT deleted`,
			o.ContactNumber,
		)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if len(newlyInsertedOrders) <= 0 {
			http.Error(w, "Fail to insert", 500)
			return
		}

		orders[i] = newlyInsertedOrders[0]
		orders[i].Provider = currProvider
	}

	for _, o := range orders {
		go scheduleReminder(o, stopSignal)
	}

	RenderJSON(w, &map[string]int{})
}

// GET /api/order/:provider_id
func getOrdersByProvider(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	providerID, _ := strconv.Atoi(ps.ByName("provider_id"))
	query := `SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id, retries_count FROM orders WHERE provider_id = $1 AND NOT deleted`
	orders, err := fetchOrders(query, providerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	providers, err := fetchProviders(`SELECT id, title, contact_number, reminder_time FROM providers WHERE id = $1 AND NOT deleted`, providerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(providers) == 0 {
		http.Error(w, "Not Found", 404)
		return
	}

	for _, o := range orders {
		query := `SELECT time_slot_id, order_id FROM choices WHERE order_id = $1`
		choices, err := fetchChoices(query, o.ID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		o.Choices = choices
	}

	RenderJSON(w, map[string][]*order{"orders": orders})
}

// DELETE /api/order/:id
func deleteOrder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ID, _ := strconv.Atoi(ps.ByName("id"))
	query := `UPDATE orders SET deleted = TRUE WHERE id = $1`
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

func fetchOrders(query string, args ...interface{}) ([]*order, error) {
	rows, err := dbConn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]*order, 0)
	for rows.Next() {
		o := new(order)
		err = rows.Scan(&o.ID, &o.CustomerName, &o.ContactNumber, &o.DeliveryDate, &o.ProviderID, &o.RetriesCount)
		if err != nil {
			return nil, err
		}

		results = append(results, o)
	}

	return results, nil
}
