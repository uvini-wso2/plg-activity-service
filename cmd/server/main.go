package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/uvini-wso2/plg-activity-service/internal/apim"
	emailpkg "github.com/uvini-wso2/plg-activity-service/internal/email"
	"github.com/uvini-wso2/plg-activity-service/internal/handler"
	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

func main() {
	// Load .env if present — silently ignored if it doesn't exist.
	_ = godotenv.Load()

	apiKey := os.Getenv("ASGARDEO_MOESIF_API_KEY")
	baseURL := os.Getenv("MOESIF_BASE_URL")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	client := moesif.NewClient(moesif.Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /validate", handler.Validate(client))

	apimClient := apim.NewClient(apim.Config{
		APIKey:  os.Getenv("APIM_MOESIF_API_KEY"),
		BaseURL: os.Getenv("MOESIF_BASE_URL"),
	})

	emailGenerator := emailpkg.NewClient(emailpkg.Config{
		APIKey: os.Getenv("ANTHROPIC_API_KEY"),
		Model:  "claude-sonnet-4-5",
	})
	mux.HandleFunc("GET /generate-email", handler.GenerateEmail(client, emailGenerator))

	// Raw activity endpoints (/events, /apim/events) are internal-only per
	// team direction (2026-09-16): only /validate and /generate-email
	// should be publicly exposed. Kept available locally for
	// testing/debugging via ENABLE_RAW_ACTIVITY_ROUTES=true in .env — omit
	// it (or set to anything else) on any real deployment to keep them
	// off. NOTE: /apim/events is currently the ONLY way to retrieve APIM
	// data at all (no APIM /validate equivalent exists yet) — revisit
	// this once that's built.
	if os.Getenv("ENABLE_RAW_ACTIVITY_ROUTES") == "true" {
		mux.HandleFunc("GET /events", handler.Events(client))
		mux.HandleFunc("GET /apim/events", handler.APIMEvents(apimClient))
		slog.Info("raw activity routes enabled (local/testing only)")
	}
	slog.Info("starting server", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("server failed", "error", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
