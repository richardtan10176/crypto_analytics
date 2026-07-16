package config

import (
	"reflect"
	"testing"
)

func TestSymbols(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"single", "BTCUSDT", []string{"BTCUSDT"}},
		{"multiple", "BTCUSDT,ETHUSDT,SOLUSDT", []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}},
		{"mixed case", "btcusdt,EthUsdt", []string{"BTCUSDT", "ETHUSDT"}},
		{"whitespace", " BTCUSDT , ETHUSDT ", []string{"BTCUSDT", "ETHUSDT"}},
		{"trailing comma", "BTCUSDT,ETHUSDT,", []string{"BTCUSDT", "ETHUSDT"}},
		{"duplicates", "BTCUSDT,btcusdt,BTCUSDT", []string{"BTCUSDT"}},
		{"empty", "", nil},
		{"only commas", ",,", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Symbols(tt.raw)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Symbols(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
