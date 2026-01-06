package redis

import (
	"reflect"
	"testing"
)

func TestParseClusterOptions(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []string
		password string
	}{
		{
			name:     "simple comma separated hosts",
			url:      "host1:6379,host2:6379",
			expected: []string{"host1:6379", "host2:6379"},
			password: "",
		},
		{
			name:     "redis urls with password",
			url:      "redis://:pass@host1:6379,redis://:pass@host2:6379",
			expected: []string{"host1:6379", "host2:6379"},
			password: "pass",
		},
		{
			name:     "mixed urls and hosts",
			url:      "redis://:pass@host1:6379,host2:6379",
			expected: []string{"host1:6379", "host2:6379"},
			password: "pass",
		},
		{
			name:     "whitespace handling",
			url:      " host1:6379 , host2:6379 ",
			expected: []string{"host1:6379", "host2:6379"},
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := parseClusterOptions(tt.url)
			if !reflect.DeepEqual(opts.Addrs, tt.expected) {
				t.Errorf("expected addrs %v, got %v", tt.expected, opts.Addrs)
			}
			if opts.Password != tt.password {
				t.Errorf("expected password %q, got %q", tt.password, opts.Password)
			}
		})
	}
}
