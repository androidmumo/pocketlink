package config

import "testing"

func TestDefaults(t *testing.T) {
	c, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddress != "127.0.0.1:8080" || c.ShutdownTimeout.Seconds() != 10 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}
func TestInvalidSettings(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"POCKETLINK_LISTEN", "example.com:8080"}, {"POCKETLINK_LISTEN", ":0"}, {"POCKETLINK_LISTEN", ":65536"},
		{"POCKETLINK_DATABASE", ":memory:"}, {"POCKETLINK_DATABASE", "file:test?mode=memory"}, {"POCKETLINK_DATABASE", " "},
		{"POCKETLINK_SHUTDOWN_TIMEOUT", "0s"}, {"POCKETLINK_SHUTDOWN_TIMEOUT", "2m"}, {"POCKETLINK_LOG_LEVEL", "secret-value"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			_, err := Load(func(k string) string {
				if k == tc.key {
					return tc.value
				}
				return ""
			})
			if err == nil {
				t.Fatal("accepted invalid setting")
			}
		})
	}
}
func TestIPv6(t *testing.T) {
	_, err := Load(func(k string) string {
		if k == "POCKETLINK_LISTEN" {
			return "[::1]:8080"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
}
