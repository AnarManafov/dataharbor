package common

import (
	"testing"

	"github.com/spf13/viper"
)

func resetAndParseXrdConfig() {
	viper.Reset()
	ParseXrdConfig()
}

func TestParseXrdConfig(t *testing.T) {
	resetAndParseXrdConfig()

	tests := []struct {
		field    string
		expected interface{}
		actual   interface{}
	}{
		{"initial_ir", "/tmp/", XrdConfig.InitialDir},
		{"host", "localhost", XrdConfig.Host},
		{"process_timeout", 60, int(XrdConfig.ProcessTimeout)},
	}

	for _, tt := range tests {
		if tt.actual != tt.expected {
			t.Errorf("Expected %s to be %v, but got %v", tt.field, tt.expected, tt.actual)
		}
	}
}
