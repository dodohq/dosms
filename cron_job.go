package main

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/luca-moser/chronos"
)

func initCron(stopSignal <-chan int) {
	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id, retries_count FROM orders WHERE NOT deleted`)
	if err != nil {
		log.Fatal("Failed to query for all orders to initiate cron job:", err.Error())
		return
	}

	for _, o := range orders {
		p, err := fetchProviders(`
			SELECT id, title, contact_number, EXTRACT(HOUR FROM timezone('UTC', reminder_time))
			FROM providers WHERE id = $1 AND NOT deleted LIMIT 1`,
			o.ProviderID,
		)
		if err != nil {
			log.Fatal("Failed to get provider for order", o.ID, ":", err.Error())
			return
		}

		o.Provider = p[0]
	}

	for _, o := range orders {
		go scheduleReminder(o, stopSignal)
	}
}

// GET /api/cron/test?customer_name=&contact_number=
func trialExecutionCron(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	queryVals := r.URL.Query()
	cName := queryVals.Get("customer_name")
	cNumber := queryVals.Get("contact_number")
	if cName == "" || cNumber == "" {
		http.Error(w, "Invalid Customer Info", 400)
		return
	}
	o := &order{
		ID:            1,
		CustomerName:  cName,
		ContactNumber: cNumber,
		DeliveryDate:  time.Now().Add(time.Hour * time.Duration(24)).UTC().Format("2006-01-02"),
		Provider: &provider{
			Title:         "Aramex",
			ContactNumber: "+6587654321",
			ReminderTime:  time.Now().Add(time.Minute * time.Duration(1)).UTC().Format("15:04"),
			Slots: []*timeSlot{
				&timeSlot{StartTime: "13", EndTime: "14"},
				&timeSlot{StartTime: "14", EndTime: "15"},
				&timeSlot{StartTime: "15", EndTime: "16"},
			},
		},
	}

	go scheduleReminder(o, stopSignal)

	RenderJSON(w, o)
}

// GET /api/cron/trigger/:order_id
func trialTriggerReminder(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	orderID, _ := strconv.Atoi(ps.ByName("order_id"))

	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, delivery_date, provider_id, retries_count FROM orders WHERE id = $1 AND NOT deleted`, orderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(orders) <= 0 {
		http.Error(w, "Not Found", 404)
		return
	}
	currOrder := orders[0]

	providers, err := fetchProviders(`
		SELECT id, title, contact_number, EXTRACT(HOUR FROM timezone('UTC', reminder_time))
		FROM providers WHERE id = $1 AND NOT deleted`,
		currOrder.ProviderID,
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
	`, currProvider.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	currProvider.Slots = slots

	currOrder.DeliveryDate = time.Now().Add(time.Hour * time.Duration(24)).UTC().Format("2006-01-02")
	currProvider.ReminderTime = time.Now().Add(time.Minute * time.Duration(1)).UTC().Format("15:04")
	currOrder.Provider = currProvider

	go scheduleReminder(currOrder, stopSignal)

	RenderJSON(w, currOrder)
}

// scheduleReminder schedule new cron job for an order
// to be used in a separate goroutine
// order must have a valid reminder time from its Provider
// that can be converted from string to integer or in "HH:MM" format.
// order.DeliveryDate must be in format of 'YYYY-MM-DD'.
// Both timing is assumed to be in UTC timezonea.
func scheduleReminder(o *order, stopSignal <-chan int) {
	if o.Provider.ReminderTime == "" {
		return
	}

	datetime, err := generateGoDateFromString(o.DeliveryDate, o.Provider.ReminderTime)
	if err != nil {
		log.Fatal("Failed to generate datetime of order", o.ID)
	}

	// send one day before the delivery date
	datetime = datetime.Add(time.Hour * time.Duration(-24))
	if datetime.Before(time.Now()) {
		return
	}
	plan := chronos.NewOnceAtDatePlan(datetime)
	task := chronos.NewScheduledTask(func() {
		sendReminderSms(o)
	}, plan)
	defer task.Stop()

	task.Start()

	<-stopSignal
}

// dateStr in format YYYY-MM-DD
// hourStr is format HH or HH:MM
func generateGoDateFromString(dateStr, hourMinuteStr string) (time.Time, error) {
	dateArr := strings.Split(dateStr, "-")
	year, err := strconv.Atoi(dateArr[0])
	if err != nil {
		return time.Time{}, err
	}
	monthInt, err := strconv.Atoi(dateArr[1])
	if err != nil {
		return time.Time{}, err
	}
	var month time.Month
	switch monthInt {
	case 12:
		month = time.December
	case 11:
		month = time.November
	case 10:
		month = time.October
	case 9:
		month = time.September
	case 8:
		month = time.August
	case 7:
		month = time.July
	case 6:
		month = time.June
	case 5:
		month = time.May
	case 4:
		month = time.April
	case 3:
		month = time.March
	case 2:
		month = time.February
	case 1:
		month = time.January
	}
	if month == 0 {
		return time.Time{}, errors.New("Invalid Month")
	}
	date, err := strconv.Atoi(dateArr[2])
	if err != nil {
		return time.Time{}, err
	}

	hourMinuteArr := strings.Split(hourMinuteStr, ":")

	hour, err := strconv.Atoi(hourMinuteArr[0])
	if err != nil {
		return time.Time{}, err
	}

	minute := 0
	if len(hourMinuteArr) >= 2 {
		minute, err = strconv.Atoi(hourMinuteArr[1])
		if err != nil {
			return time.Time{}, err
		}
	}

	return time.Date(year, month, date, hour, minute, 0, 0, time.UTC), nil
}
