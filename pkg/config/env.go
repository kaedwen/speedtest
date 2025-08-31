package config

import (
	"os"
)

func init() {
	ReadString("INFLUX_ORG", &InfluxOrg)
	ReadString("INFLUX_BUCKET", &InfluxBucket)
	TryReadString("INFLUX_HOST", &InfluxHost)
	ReadString("INFLUX_TOKEN", &InfluxToken)
	TryReadString("INFLUX_MEASUREMENT", &InfluxMeasurement)
	TryReadString("TEST_DNS_TARGET", &TestDNSTarget)
	TryReadString("TEST_DNS_HOST", &TestDNSHost)
	TryReadString("TEST_SCHEDULE", &TestSchedule)
}

func TryReadString(key string, target *string) bool {
	if value, exists := os.LookupEnv(key); target != nil && exists {
		*target = value
		return true
	}

	return false
}

func ReadString(key string, target *string) {
	if !TryReadString(key, target) {
		panic("environment variable " + key + " not set")
	}
}
