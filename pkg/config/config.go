package config

var (
	InfluxOrg         string
	InfluxBucket      string
	InfluxHost        string = "http://localhost:8086"
	InfluxToken       string
	InfluxMeasurement string = "speed"
	TestDNSTarget     string = "8.8.8.8"
	TestDNSHost       string
)

var (
	TestSchedule string = "0 */5 * * * *" // every 5 minutes
)
