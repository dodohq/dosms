package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
)

var dbConn *sql.DB
var httpRouter *httprouter.Router

func main() {
	isDevEnv := os.Getenv("GO_ENV") == "development"
	if isDevEnv {
		if err := godotenv.Load(); err != nil {
			log.Fatal(err)
		}
	}

	conn, err := sql.Open("postgres", os.Getenv("DB"))
	if err != nil {
		log.Fatal(err)
	}
	dbConn = conn
	defer conn.Close()

	httpRouter = httprouter.New()

	httpRouter.POST("/api/provider", createNewProvider)
	httpRouter.PUT("/api/provider/:id/set_reminder", setProviderReminderTime)
	httpRouter.GET("/api/provider", getAllProviders)
	httpRouter.GET("/api/provider/:id", getProviderByID)
	httpRouter.DELETE("/api/provider/:id", deleteProvider)

	httpRouter.POST("/api/time_slot", createNewTimeSlot)
	httpRouter.GET("/api/time_slot/:provider_id", getTimeSlotsByProvider)
	httpRouter.DELETE("/api/time_slot/:id", deleteTimeSlot)

	httpRouter.POST("/api/sms", sendAnSms)
	httpRouter.POST("/api/sms/reply", respondToSms)

	httpRouter.POST("/api/order", createNewOrder)
	httpRouter.GET("/api/order/:provider_id", getOrdersByProvider)
	httpRouter.DELETE("/api/order/:id", deleteOrder)

	httpRouter.POST("/api/choice", createNewChoice)
	httpRouter.GET("/api/choice/:order_id", getChoicesByOrder)
	httpRouter.DELETE("/api/choice/:order_id/:time_slot_id", deleteChoice)

	whereToListen := ":" + os.Getenv("PORT")
	if isDevEnv {
		whereToListen = "localhost" + whereToListen
	}
	fmt.Println("Starting Server on " + whereToListen)
	log.Fatal(http.ListenAndServe(whereToListen, httpRouter))
}
