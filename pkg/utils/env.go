package utils

import (
	"fmt"
	"os"
)

type TestConfig struct {
	Org    string
	Bucket string
	Host   string
	Token  string
	ID     string
	DNS    struct {
		Target string
		Host   string
	}
}

func GetConfig() (*TestConfig, error) {
	org, has := os.LookupEnv("INFLUX_ORG")
	if !has {
		return nil, fmt.Errorf("no influx org given, please set the environment variable INFLUX_ORG")
	}

	bucket, has := os.LookupEnv("INFLUX_BUCKET")
	if !has {
		return nil, fmt.Errorf("no influx bucket given, please set the environment variable INFLUX_BUCKET_NAME")
	}

	id, has := os.LookupEnv("ID_TAG")
	if !has {
		return nil, fmt.Errorf("no id tag given, please set the environment variable ID_TAG")
	}

	host, has := os.LookupEnv("INFLUX_HOST")
	if !has {
		host = "http://localhost:8086"
	}

	dnsTarget, has := os.LookupEnv("TEST_DNS_TARGET")
	if !has {
		dnsTarget = "8.8.8.8"
	}

	dnsHost, has := os.LookupEnv("TEST_DNS_HOST")
	if !has {
		return nil, fmt.Errorf("no dns host given, please set the environment variable TEST_DNS_HOST")
	}

	return &TestConfig{
		Org:    org,
		Bucket: bucket,
		Host:   host,
		Token:  os.Getenv("INFLUX_TOKEN"),
		ID:     id,
		DNS: struct {
			Target string
			Host   string
		}{
			Target: dnsTarget,
			Host:   dnsHost,
		},
	}, nil
}

func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
