package destinations

import (
	"testing"
	"time"
)

func TestParseTimestamp_Valid(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid RFC3339",
			input:   "2024-01-15T10:30:00Z",
			wantErr: false,
		},
		{
			name:    "valid RFC3339 with timezone",
			input:   "2024-01-15T10:30:00-08:00",
			wantErr: false,
		},
		{
			name:    "invalid format",
			input:   "not-a-time",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid date",
			input:   "2024-13-45T25:99:99Z",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify we got a valid time back
				if result.IsZero() {
					t.Error("parseTimestamp() returned zero time for valid input")
				}
				// Verify round-trip
				formatted := result.Format(time.RFC3339)
				result2, err := parseTimestamp(formatted)
				if err != nil {
					t.Errorf("round-trip failed: %v", err)
				}
				if !result.Equal(result2) {
					t.Errorf("round-trip mismatch: got %v, want %v", result2, result)
				}
			}
		})
	}
}
