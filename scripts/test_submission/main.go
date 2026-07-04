package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type orderResponse struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

type orderbookResponse struct {
	Bids []any `json:"bids"`
	Asks []any `json:"asks"`
}

var orderCounter uint64
// edit changed
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/orderbook", handleOrderbook)
	mux.HandleFunc("/order", handleOrder)
	mux.HandleFunc("/order/", handleOrderByID)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout:  5 * time.Second,
		WriteTimeout:       10 * time.Second,
		IdleTimeout:        30 * time.Second,
	}

	log.Println("test submission listening on :8080")
	log.Fatal(server.ListenAndServe())
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleOrderbook(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, orderbookResponse{Bids: []any{}, Asks: []any{}})
}

func handleOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := nextOrderID()
	writeJSON(w, http.StatusOK, orderResponse{OrderID: id, Status: "ACCEPTED"})
}

func handleOrderByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/order/"):]
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"orderId": id,
		"status":  "CANCELLED",
	})
}

func nextOrderID() string {
	value := atomic.AddUint64(&orderCounter, 1)
	return fmt.Sprintf("order-%d-%s", value, strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
