package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	workflow := OrderWorkflow{Mail: NewInfraiClient(apiKey)}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders/receipt", func(w http.ResponseWriter, r *http.Request) {
		var order Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			http.Error(w, "invalid order", http.StatusBadRequest)
			return
		}
		result, err := workflow.SendReceipt(r.Context(), order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	log.Printf("order receipt service listening on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}
