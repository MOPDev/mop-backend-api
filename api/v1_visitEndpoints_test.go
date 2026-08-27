package api

import (
	"testing"
	"time"
)

func TestGroupVisits(t *testing.T) {
	now := time.Now()
	rows := []map[string]interface{}{
		{"sagsnr": int64(1), "adresse": "Main St 1", "postnr": "1000", "bynavn": "Aarhus",
			"status": int64(4), "Fristdato": now, "klientnr": int64(1),
			"klientnavn": "X", "sagVedr": "y", "debitorId": int64(1), "navn": "A"},
		{"sagsnr": int64(1), "adresse": "MAIN ST 1, 2. sal", "postnr": "1000", "bynavn": "Aarhus",
			"status": int64(4), "Fristdato": now, "klientnr": int64(1),
			"klientnavn": "X", "sagVedr": "y", "debitorId": int64(2), "navn": "B"},
	}
	out := groupVisits(rows)
	if len(out) != 1 {
		t.Fatalf("expected 1 grouped visit, got %d", len(out))
	}
	if len(out[0]["debtors"].([]map[string]interface{})) != 2 {
		t.Fatalf("expected 2 debtors merged into one visit")
	}
	// the longer variant (before the comma is stripped, only compared) wins on raw length
	if out[0]["adresse"].(string) != "MAIN ST 1, 2. sal" {
		t.Fatalf("expected longest raw address to win, got %q", out[0]["adresse"])
	}
}
