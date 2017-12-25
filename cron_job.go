package main

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/luca-moser/chronos"
)

func initCron(stopSignal <-chan int) {
	orders, err := fetchOrders(`SELECT id, customer_name, contact_number, to_char(delivery_date, 'YYYY-MM-DD'), provider_id FROM orders WHERE NOT deleted`)
	if err != nil {
		log.Fatal("Failed to query for all orders to initiate cron job")
		return
	}

	for _, o := range orders {
		p, err := fetchProviders(`
			SELECT id, title, contact_number, EXTRACT(HOUR FROM timezone('UTC', reminder_time))
			FROM providers WHERE id = $1 AND NOT deleted LIMIT 1`,
			o.ProviderID,
		)
		if err != nil {
			log.Fatal("Failed to get provider for order ", o.ID)
			return
		}

		o.Provider = p[0]
	}

	for _, o := range orders {
		if o.Provider.ReminderTime == "" {
			continue
		}
		datetime, err := generateGoDateFromString(o.DeliveryDate, o.Provider.ReminderTime)
		datetime.Add(time.Hour * time.Duration(24))
		if err != nil {
			log.Fatal("Failed to generate datetime of order ", o.ID)
		}
		if datetime.Before(time.Now()) {
			continue
		}

		plan := chronos.NewOnceAtDatePlan(datetime)
		task := chronos.NewScheduledTask(func() {
			fmt.Println("order ", o.ID)
		}, plan)
		defer task.Stop()

		task.Start()
	}

	<-stopSignal
}

// dateStr in format YYYY-MM-DD
// hourStr is
func generateGoDateFromString(dateStr, hourStr string) (time.Time, error) {
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
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return time.Time{}, err
	}

	return time.Date(year, month, date, hour, 0, 0, 0, time.UTC), nil
}
