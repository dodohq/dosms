package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var dbConn *sql.DB
var httpRouter *httprouter.Router
var stopSignal = make(chan int)
var httpsClient *http.Client

func init() {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pemCerts)
	httpsClient = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: pool},
		},
	}
}

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

	go initCron(stopSignal)
	defer func() { stopSignal <- 1 }()

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
	httpRouter.POST("/api/order/:provider_id/csv_upload", newOrdersFromCsv)
	httpRouter.GET("/api/order/:provider_id", getOrdersByProvider)
	httpRouter.DELETE("/api/order/:id", deleteOrder)

	httpRouter.POST("/api/choice", createNewChoice)
	httpRouter.GET("/api/choice/:order_id", getChoicesByOrder)
	httpRouter.DELETE("/api/choice/:order_id/:time_slot_id", deleteChoice)

	httpRouter.GET("/api/cron/test", trialExecutionCron)
	httpRouter.GET("/api/cron/trigger/:order_id", trialTriggerReminder)

	routerWithCors := cors.AllowAll().Handler(httpRouter)

	whereToListen := ":" + os.Getenv("PORT")
	if isDevEnv {
		whereToListen = "localhost" + whereToListen
	}
	fmt.Println("Starting Server on " + whereToListen)
	log.Fatal(http.ListenAndServe(whereToListen, routerWithCors))
}
