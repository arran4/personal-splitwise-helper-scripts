package tui

import (
	"math"
	"testing"
)

func TestIsFiniteNumber(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  bool
	}{
		{name: "finite", value: 12.34, want: true},
		{name: "nan", value: math.NaN(), want: false},
		{name: "positive inf", value: math.Inf(1), want: false},
		{name: "negative inf", value: math.Inf(-1), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFiniteNumber(tt.value); got != tt.want {
				t.Fatalf("isFiniteNumber(%v) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
