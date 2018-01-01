package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

type sms struct {
	From string `json:"from,omitempty" schema:"From"`
	To   string `json:"to" schema:"To"`
	Body string `json:"body" schema:"Body"`
}

func sendWithTwilio(toNumber string, body string) (*http.Response, error) {
	urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + os.Getenv("TWILIO_SID") + "/Messages.json"
	msgData := url.Values{}
	msgData.Set("From", os.Getenv("TWILIO_NUMBER"))
	msgData.Set("To", toNumber)
	msgData.Set("Body", body)
	msgDataReader := *strings.NewReader(msgData.Encode())

	req, err := http.NewRequest("POST", urlStr, &msgDataReader)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(os.Getenv("TWILIO_SID"), os.Getenv("TWILIO_TOKEN"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return httpsClient.Do(req)
}

// sendReminderSms send standard reminder sms the day before delivery
// order must have Provider populated.
// order.Provider must have Slots populated.
func sendReminderSms(o *order) (*http.Response, error) {
	bodyStr := "From: " + o.Provider.Title + "\n"
	bodyStr += "Hello " + o.CustomerName + ", your delivery is scheduled to be delivered tomorrow "
	bodyStr += time.Now().Add(time.Hour*time.Duration(24)).Format("Mon 2006 Jan 02") + ". "
	bodyStr += "Please state your available time slots by replying the number beside the time slot. If you’re available for more than one time slot, reply with a space between the numbers. E.g 1 2 4\nIgnore this message if it’s not meant for you.\n\n"

	for idx, s := range o.Provider.Slots {
		bodyStr += strconv.Itoa(idx) + ": " + s.StartTime + ":00" + "-" + s.EndTime + ":00" + "\n"
	}

	return sendWithTwilio(o.ContactNumber, bodyStr)
}

// sendConfirmationSms send standard cofirmation sms after receive slot
// order must have Choices populated.
// order.Choices must have TimeSlot populated
func sendConfirmationSms(o *order) (*http.Response, error) {
	bodyStr := "Thank you " + o.CustomerName + ". The courier will be coming during your available time slots: "
	for i, c := range o.Choices {
		bodyStr += c.TimeSlot.StartTime + ":00" + ":" + c.TimeSlot.EndTime + ":00"
		if i < len(o.Choices)-1 {
			bodyStr += ", "
		}
	}
	bodyStr += ". Do note that delivery might sometimes be off schedule due to unforeseen circumstances. Reply ‘WRONG’ if you would like to change your available time slots. Otherwise, thank you for your time."

	return sendWithTwilio(o.ContactNumber, bodyStr)
}

// sendRetrySms send standard retry sms
// order must have Provider populate
// order.Provider must have Slots populated
func sendRetrySms(o *order, lastChance bool) (*http.Response, error) {
	bodyStr := "Please reply the number that represents your available time slot. If you’re available for more than one time slot, reply with a space between the numbers. E.g 1 2 4\n\n"
	if lastChance {
		bodyStr = "Please confirm your available time slot. There will be no more changes after this. " + bodyStr
	}
	for idx, s := range o.Provider.Slots {
		bodyStr += strconv.Itoa(idx) + ": " + s.StartTime + ":00" + "-" + s.EndTime + ":00" + "\n"
	}

	return sendWithTwilio(o.ContactNumber, bodyStr)
}

// sendMaxExceededSms send standard sms after max retries made
// order must have valid ContactNumber field
func sendMaxExceededSms(o *order) (*http.Response, error) {
	bodyStr := "You have exceeded the number of changes. Please call +6581489408 to confirm your delivery timings. Thank you."
	return sendWithTwilio(o.ContactNumber, bodyStr)
}

func handleChoosingSlots(w http.ResponseWriter, s *sms) {
	choicesIdxArr := strings.Split(strings.TrimSpace(s.Body), " ")
	if len(choicesIdxArr) <= 0 {
		http.Error(w, "No choices made", 400)
		return
	}

	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id, retries_count FROM orders WHERE contact_number = $1 AND NOT deleted`, s.From)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	} else if len(orders) <= 0 {
		http.Error(w, "No Order Found", 404)
		return
	}
	o := orders[0]
	if o.RetriesCount >= 3 {
		sendMaxExceededSms(o)
	}

	slots, err := fetchTimeSlots(`
		SELECT id, EXTRACT(HOUR FROM start_time), EXTRACT(HOUR FROM end_time), provider_id
		FROM time_slots WHERE provider_id = $1 AND NOT deleted ORDER BY start_time ASC
	`, o.ProviderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// clear all previously made choices
	query := `UPDATE choices SET deleted = TRUE WHERE order_id = $1`
	stmt, err := dbConn.Prepare(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_, err = stmt.Exec(o.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	noChoiceMade := true
	for i, slot := range slots {
		for _, idx := range choicesIdxArr {
			if strconv.Itoa(i) == idx {
				noChoiceMade = false
				query := `INSERT INTO choices(time_slot_id, order_id) VALUES($1, $2) RETURNING time_slot_id`
				var timeSlotID int64
				err := dbConn.QueryRow(query, slot.ID, o.ID).Scan(&timeSlotID)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return
				}
				o.Choices = append(o.Choices, &choice{TimeSlot: slot})
			}
		}
	}
	if noChoiceMade {
		http.Error(w, "No choice made", 400)
		return
	}
	query = `UPDATE orders SET retries_count = retries_count + 1 WHERE id = $1`
	stmt, err = dbConn.Prepare(query)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	_, err = stmt.Exec(o.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	sendConfirmationSms(o)
}

func handleRetry(w http.ResponseWriter, s *sms) {
	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id, retries_count FROM orders WHERE contact_number = $1 AND NOT deleted`, s.From)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	} else if len(orders) <= 0 {
		http.Error(w, "No Order Found", 404)
		return
	}
	o := orders[0]
	if o.RetriesCount >= 3 {
		sendMaxExceededSms(o)
	}

	slots, err := fetchTimeSlots(`
		SELECT id, EXTRACT(HOUR FROM start_time), EXTRACT(HOUR FROM end_time), provider_id
		FROM time_slots WHERE provider_id = $1 AND NOT deleted ORDER BY start_time ASC
	`, o.ProviderID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	o.Provider = new(provider)
	o.Provider.Slots = slots

	lastChance := o.RetriesCount == 2

	sendRetrySms(o, lastChance)
}

// POST /api/sms
func sendAnSms(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s := sms{}
	if err := ReadRequestBody(r, &s); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	resp, _ := sendWithTwilio(s.To, s.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		RenderJSON(w, b)
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		http.Error(w, string(b), resp.StatusCode)
	}
}

// POST /api/sms/reply
func respondToSms(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s := sms{}
	if err := ReadRequestBody(r, &s); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if s.Body == "" {
		http.Error(w, "Empty Message", 400)
		return
	}

	choosingSlots, err := regexp.Match("^[\\s\\d]+$", []byte(s.Body))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if choosingSlots {
		handleChoosingSlots(w, &s)
		return
	}

	retrying := strings.ToUpper(s.Body) == "WRONG"
	if retrying {
		handleRetry(w, &s)
		return
	}

	http.Error(w, "Invalid SMS Reply", 400)
}
