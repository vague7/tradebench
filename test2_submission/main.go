package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	http.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId":"test-order-id","status":"ACCEPTED"}`))
		} else if r.Method == http.MethodDelete {
			// This matches /order/:id if strict routing isn't used, but let's handle /order/ path below instead.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId":"test-order-id","status":"CANCELLED"}`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// For DELETE /order/:id
	http.HandleFunc("/order/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId":"test-order-id","status":"CANCELLED"}`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.HandleFunc("/orderbook", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"bids":[],"asks":[]}`))
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	http.ListenAndServe(":8080", nil)
}
