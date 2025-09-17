// internal/websocket/webhook.go
package websocket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// WebhookHandler verifies the HMAC, parses the Event, and pushes it onto Events.
func WebhookHandler(secret []byte) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        body, err := io.ReadAll(r.Body)
        r.Body.Close()
        if err != nil {
            log.Printf("webhook read error: %v", err)
            http.Error(w, "bad request", http.StatusBadRequest)
            return
        }

        if len(secret) == 0 {
            log.Printf("webhook refused: missing secret (set WEBHOOK_SECRET)")
            http.Error(w, "service unavailable", http.StatusServiceUnavailable)
            return
        }

		sig := r.Header.Get("X-Hook-Signature")
		if !verifySignature(body, sig, secret) {
			log.Printf("webhook invalid signature: %q", sig)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var evt Event
		if err := json.Unmarshal(body, &evt); err != nil {
			log.Printf("webhook JSON error: %v", err)
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}

		log.Printf("webhook received:")
		log.Printf("  EventType: %#v", evt.EventType)
		log.Printf("  ModuleID:  %#v", evt.ModuleID)
		log.Printf("  Timestamp: %#v", evt.Timestamp)
		log.Printf("  Payload:   %+v", string(evt.Payload))
		Events <- evt
		w.WriteHeader(http.StatusAccepted)
	}
}

func verifySignature(body []byte, sigHeader string, secret []byte) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sigHeader))
}
