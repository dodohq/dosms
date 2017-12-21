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
	httpRouter.GET("/api/provider", getAllProviders)
	httpRouter.GET("/api/provider/:id", getProviderByID)
	httpRouter.DELETE("/api/provider/:id", deleteProvider)

	whereToListen := ":" + os.Getenv("PORT")
	if isDevEnv {
		whereToListen = "localhost" + whereToListen
	}
	fmt.Println("Starting Server on " + whereToListen)
	log.Fatal(http.ListenAndServe(whereToListen, httpRouter))
}
