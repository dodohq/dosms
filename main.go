package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	isDevEnv := os.Getenv("GO_ENV") == "development"
	if isDevEnv {
		if err := godotenv.Load(); err != nil {
			log.Fatal(err)
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})

	whereToListen := ":" + os.Getenv("PORT")
	if isDevEnv {
		whereToListen = "localhost" + whereToListen
	}
	fmt.Println("Starting Server on " + whereToListen)
	log.Fatal(http.ListenAndServe(whereToListen, nil))
}
