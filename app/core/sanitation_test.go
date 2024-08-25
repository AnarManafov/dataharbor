package core

import (
	"testing"
	"time"
)

func TestIsOlderThanXHours(t *testing.T) {
	type args struct {
		_t time.Time
		_x uint
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"", args{time.Date(
			2021, 8, 15, 14, 30, 45, 100, time.Local), 3}, true},
		{"", args{time.Now().Add(-time.Hour * time.Duration(2)), 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsOlderThanXHours(tt.args._t, tt.args._x); got != tt.want {
				t.Errorf("IsOlderThanXHours(%v, %v) = %v, want %v", tt.args._t, tt.args._x, got, tt.want)
			}
		})
	}
}
