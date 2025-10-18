package helper

import (
	"math"
	"testing"
	"time"
)

func TestDurationToUint32(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     uint32
	}{
		{
			name:     "zero duration",
			duration: 0,
			want:     0,
		},
		{
			name:     "positive duration",
			duration: 10 * time.Second,
			want:     10,
		},
		{
			name:     "negative duration",
			duration: -5 * time.Second,
			want:     0,
		},
		{
			name:     "max uint32 boundary",
			duration: time.Duration(math.MaxUint32) * time.Second,
			want:     math.MaxUint32,
		},
		{
			name:     "exceeds max uint32",
			duration: time.Duration(math.MaxUint32+1000) * time.Second,
			want:     math.MaxUint32,
		},
		{
			name:     "one second",
			duration: 1 * time.Second,
			want:     1,
		},
		{
			name:     "milliseconds rounded down",
			duration: 1500 * time.Millisecond,
			want:     1,
		},
		{
			name:     "one minute",
			duration: 1 * time.Minute,
			want:     60,
		},
		{
			name:     "one hour",
			duration: 1 * time.Hour,
			want:     3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DurationToUint32(tt.duration)
			if got != tt.want {
				t.Errorf("DurationToUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntToUint32(t *testing.T) {
	tests := []struct {
		name string
		val  int
		want uint32
	}{
		{
			name: "zero",
			val:  0,
			want: 0,
		},
		{
			name: "positive integer",
			val:  100,
			want: 100,
		},
		{
			name: "negative integer",
			val:  -50,
			want: 0,
		},
		{
			name: "max uint32",
			val:  math.MaxUint32,
			want: math.MaxUint32,
		},
		{
			name: "exceeds max uint32",
			val:  math.MaxUint32 + 1000,
			want: math.MaxUint32,
		},
		{
			name: "one",
			val:  1,
			want: 1,
		},
		{
			name: "large negative",
			val:  -999999,
			want: 0,
		},
		{
			name: "max int32",
			val:  math.MaxInt32,
			want: math.MaxInt32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntToUint32(tt.val)
			if got != tt.want {
				t.Errorf("IntToUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}
