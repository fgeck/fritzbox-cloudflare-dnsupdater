package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/cloudflare/cloudflare-go"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", mainHandler).Methods("GET")
	router.HandleFunc("/healthz", healthzHandler).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	http.Handle("/", router)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	email := r.URL.Query().Get("email")
	zone := r.URL.Query().Get("zone")
	record := r.URL.Query().Get("record")
	ipv4 := r.URL.Query().Get("ipv4")
	ipv6 := r.URL.Query().Get("ipv6")

	if token == "" {
		http.Error(w, `{"status": "error", "message": "Missing token URL parameter."}`, http.StatusBadRequest)
		return
	}
	if zone == "" {
		http.Error(w, `{"status": "error", "message": "Missing zone URL parameter."}`, http.StatusBadRequest)
		return
	}
	if ipv4 == "" && ipv6 == "" {
		http.Error(w, `{"status": "error", "message": "Missing ipv4 or ipv6 URL parameter."}`, http.StatusBadRequest)
		return
	}

	api, err := cloudflare.New(token, email)
	if err != nil {
		http.Error(w, `{"status": "error", "message": "Failed to create Cloudflare API client."}`, http.StatusInternalServerError)
		return
	}

	zones, err := api.ListZones(r.Context(), zone)
	if err != nil || len(zones) == 0 {
		http.Error(w, `{"status": "error", "message": "Failed to find zone or zone does not exist."}`, http.StatusNotFound)
		return
	}

	recordZoneConcat := zone
	if record != "" {
		recordZoneConcat = fmt.Sprintf("%s.%s", record, zone)
	}

	aRecord, err := api.DNSRecords(zones[0].ID, cloudflare.DNSRecord{Name: recordZoneConcat, Type: "A"})
	if err != nil || len(aRecord) == 0 {
		http.Error(w, fmt.Sprintf(`{"status": "error", "message": "A record for %s does not exist."}`, recordZoneConcat), http.StatusNotFound)
		return
	}

	aaaaRecord, err := api.DNSRecords(zones[0].ID, cloudflare.DNSRecord{Name: recordZoneConcat, Type: "AAAA"})
	if err != nil || len(aaaaRecord) == 0 {
		http.Error(w, fmt.Sprintf(`{"status": "error", "message": "AAAA record for %s does not exist."}`, recordZoneConcat), http.StatusNotFound)
		return
	}

	if ipv4 != "" && aRecord[0].Content != ipv4 {
		updateRecord(api, zones[0].ID, aRecord[0].ID, "A", ipv4, aRecord[0].Proxied, aRecord[0].TTL)
	}

	if ipv6 != "" && aaaaRecord[0].Content != ipv6 {
		updateRecord(api, zones[0].ID, aaaaRecord[0].ID, "AAAA", ipv6, aaaaRecord[0].Proxied, aaaaRecord[0].TTL)
	}

	fmt.Fprintf(w, `{"status": "success", "message": "Update successful."}`)
}

func updateRecord(api *cloudflare.API, zoneID, recordID, recordType, content string, proxied bool, ttl int) {
	record := cloudflare.DNSRecord{
		ID:      recordID,
		Type:    recordType,
		Content: content,
		Proxied: proxied,
		TTL:     ttl,
	}
	err := api.UpdateDNSRecord(zoneID, record.ID, record)
	if err != nil {
		fmt.Printf("Failed to update %s record: %s\n", recordType, err)
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "OK"})
}
