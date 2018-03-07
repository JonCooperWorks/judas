package main

import (
	"log"
	"net/http"
	"net/http/httputil"
)

var Name = "JudasLoggingPlugin"

func Initialize() (map[string]*string, error) {
	log.Println("Initializing", Name)
	return map[string]*string{}, nil
}

func ProcessTransactions(
	transactions <-chan struct {
		Request  http.Request
		Response http.Response
	},
	arguments map[string]*string,
) {
	for transaction := range transactions {
		req, err := httputil.DumpRequest(&transaction.Request, true)
		if err != nil {
			log.Println("Error dumping request to console.")
			return
		}
		log.Println(string(req))

		resp, err := httputil.DumpResponse(&transaction.Response, false)
		if err != nil {
			log.Println("Error dumping response to console.")
			return
		}
		log.Println(string(resp))
	}
}
