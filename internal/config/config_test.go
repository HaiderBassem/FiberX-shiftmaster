package config

import (
	"os"
	"reflect"
	"testing"
	"time"
)

func TestIsTrustedProxy(t *testing.T) {
	tests := []struct {
		name     string
		proxies  []string
		ip       string
		expected bool
	}{
		{
			name:     "Empty list",
			proxies:  []string{},
			ip:       "192.168.1.1",
			expected: false,
		},
		{
			name:     "Exact match",
			proxies:  []string{"192.168.1.1", "10.0.0.1"},
			ip:       "192.168.1.1",
			expected: true,
		},
		{
			name:     "CIDR match",
			proxies:  []string{"192.168.1.0/24"},
			ip:       "192.168.1.50",
			expected: true,
		},
		{
			name:     "CIDR mismatch",
			proxies:  []string{"192.168.1.0/24"},
			ip:       "192.168.2.50",
			expected: false,
		},
		{
			name:     "Invalid IP",
			proxies:  []string{"0.0.0.0/0"},
			ip:       "invalid-ip",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SecurityConfig{TrustedProxies: tt.proxies}
			result := cfg.IsTrustedProxy(tt.ip)
			if result != tt.expected {
				t.Errorf("IsTrustedProxy(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestUploadConfigIsAllowedType(t *testing.T) {
	cfg := UploadConfig{
		AllowedTypes: []string{"pdf", "jpg", "png"},
	}

	tests := []struct {
		ext      string
		expected bool
	}{
		{".pdf", true},
		{"pdf", true},
		{".JPG", true}, // case insensitive check
		{".jpeg", false},
		{".doc", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := cfg.IsAllowedType(tt.ext)
			if result != tt.expected {
				t.Errorf("IsAllowedType(%q) = %v, expected %v", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestGetEnvHelpers(t *testing.T) {
	// Setup test environment
	os.Setenv("TEST_INT", "42")
	os.Setenv("TEST_BOOL", "true")
	os.Setenv("TEST_SLICE", "a, b,  c  ")

	defer func() {
		os.Unsetenv("TEST_INT")
		os.Unsetenv("TEST_BOOL")
		os.Unsetenv("TEST_SLICE")
	}()

	t.Run("getEnvInt", func(t *testing.T) {
		if val := getEnvInt("TEST_INT", 0); val != 42 {
			t.Errorf("expected 42, got %v", val)
		}
		if val := getEnvInt("MISSING_INT", 99); val != 99 {
			t.Errorf("expected default 99, got %v", val)
		}
	})

	t.Run("getEnvBool", func(t *testing.T) {
		if val := getEnvBool("TEST_BOOL", false); val != true {
			t.Errorf("expected true, got %v", val)
		}
		if val := getEnvBool("MISSING_BOOL", false); val != false {
			t.Errorf("expected default false, got %v", val)
		}
	})

	t.Run("getEnvSlice", func(t *testing.T) {
		expected := []string{"a", "b", "c"}
		val := getEnvSlice("TEST_SLICE", "")
		if !reflect.DeepEqual(val, expected) {
			t.Errorf("expected %v, got %v", expected, val)
		}

		defaultExpected := []string{"x", "y"}
		val2 := getEnvSlice("MISSING_SLICE", "x,y")
		if !reflect.DeepEqual(val2, defaultExpected) {
			t.Errorf("expected default %v, got %v", defaultExpected, val2)
		}
	})

	t.Run("getEnvSeconds", func(t *testing.T) {
		if val := getEnvSeconds("TEST_INT", 0); val != 42*time.Second {
			t.Errorf("expected 42s, got %v", val)
		}
	})

}
