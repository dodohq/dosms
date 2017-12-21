package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

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

	client := &http.Client{}
	req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
	req.SetBasicAuth(os.Getenv("TWILIO_SID"), os.Getenv("TWILIO_TOKEN"))
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return client.Do(req)
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
		fmt.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}

	resp, _ := sendWithTwilio(s.From, "I got your message: "+s.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		b, _ := ioutil.ReadAll(resp.Body)
		RenderJSON(w, b)
	} else {
		b, _ := ioutil.ReadAll(resp.Body)
		http.Error(w, string(b), resp.StatusCode)
	}
}
