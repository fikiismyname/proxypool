package geoip

import (
	"testing"
)

func TestService_Lookup(t *testing.T) {
	// Assumes running from project root or having access to data dir relatively
	// Adjust path as needed for test execution context. 
	// For simplicity, we assume the test runs where data/ is available or we skip if not found.
	
	dbPath := "../../data/GeoLite2-City.mmdb"
	
	svc, err := New(dbPath)
	if err != nil {
		t.Skipf("Skipping test: DB file not found at %s: %v", dbPath, err)
	}
	defer svc.Close()

	tests := []struct {
		ip      string
		wantISO string
	}{
		{"8.8.8.8", "US"},
	}

	for _, tt := range tests {
		iso, name, err := svc.Lookup(tt.ip)
		if err != nil {
			t.Errorf("Lookup(%s) error = %v", tt.ip, err)
			continue
		}
		if iso != tt.wantISO {
			t.Errorf("Lookup(%s) ISO = %v, want %v (Name: %s)", tt.ip, iso, tt.wantISO, name)
		}
	}
}
