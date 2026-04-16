package model

import "testing"

func TestMessage_GetHostname(t *testing.T) {
	m := Message{Hostname: "vm-ABC-123", IpAddress: "10.0.0.1"}

	if m.Hostname != "vm-ABC-123" {
		t.Fatalf("expected hostname to be %q, got %q", "vm-ABC-123", m.Hostname)
	}

	// Ensure calling GetSerial does not mutate Hostname
	_ = m.GetSerial()
	if m.Hostname != "vm-ABC-123" {
		t.Fatalf("hostname was mutated by GetSerial: got %q", m.Hostname)
	}
}

func TestMessage_GetSerial(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     string
	}{
		{"no_dash", "vm", ""},
		{"single_dash", "vm-12345", "12345"},
		{"multi_dash", "vm-123-456", "123-456"},
		{"leading_dash", "-SERIAL", "SERIAL"},
		{"empty_hostname", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{Hostname: tt.hostname}
			got := m.GetSerial()
			if got != tt.want {
				t.Fatalf("GetSerial() for hostname %q = %q, want %q", tt.hostname, got, tt.want)
			}

			// Ensure result is cached in m.serial after first call, but only when non-empty
			original := got
			m.Hostname = "vm-CHANGED-999"
			if original != "" {
				if cached := m.GetSerial(); cached != original {
					t.Fatalf("expected cached serial %q after hostname change, got %q", original, cached)
				}
			}
		})
	}
}
