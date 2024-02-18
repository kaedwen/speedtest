package cmd

import (
	"context"
	"net"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	utils "github.com/kaedwen/speedtest/pkg/untils"
	"github.com/showwin/speedtest-go/speedtest"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type TestResult struct {
	Success  bool
	Host     string
	Server   string
	Latency  time.Duration
	Distance float64
	Download float64
	Upload   float64
	DNS      bool
}

func NewTestCommand(lg *zap.Logger) *cobra.Command {
	cmd := cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			config, err := utils.GetConfig()
			if err != nil {
				return err
			}

			client := influxdb2.NewClient(config.Host, config.Token)

			results, dns, err := test(ctx, lg, config)
			if err == nil {
				for _, result := range results {
					err := save(ctx, client, config, &result, dns)
					if err != nil {
						return err
					}
				}
			} else {
				err := save(ctx, client, config, &TestResult{
					Success: false,
				}, dns)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}

	return &cmd
}

func test(ctx context.Context, lg *zap.Logger, config *utils.TestConfig) ([]TestResult, bool, error) {
	var stc = speedtest.New()

	user, _ := stc.FetchUserInfo()
	lg.Info("user meta", zap.Any("info", user))

	serverList, _ := stc.FetchServers()
	targets, _ := serverList.FindServer([]int{})

	results := make([]TestResult, 0, len(targets))

	for _, s := range targets {
		err := s.PingTestContext(ctx, nil)
		if err != nil {
			return nil, false, err
		}

		err = s.DownloadTestContext(ctx)
		if err != nil {
			return nil, false, err
		}

		err = s.UploadTestContext(ctx)
		if err != nil {
			return nil, false, err
		}

		lg.Info("Result", zap.Duration("Latency", s.Latency), zap.Float64("Download", s.DLSpeed), zap.Float64("Upload", s.ULSpeed))

		results = append(results, TestResult{
			Success:  true,
			Server:   s.Name,
			Host:     s.Host,
			Latency:  s.Latency,
			Download: s.DLSpeed * 1_000_000,
			Upload:   s.ULSpeed * 1_000_000,
			Distance: s.Distance,
		})

		s.Context.Reset()
	}

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			return d.DialContext(ctx, network, net.JoinHostPort(config.DNS.Target, "53"))
		},
	}

	dns := true
	ip, err := r.LookupHost(context.Background(), config.DNS.Host)
	if err != nil || len(ip) == 0 {
		dns = false
	}

	lg.Info("Speedtest RUN ok", zap.Bool("DNS", dns))

	return results, dns, nil
}

func save(ctx context.Context, client influxdb2.Client, config *utils.TestConfig, result *TestResult, dns bool) error {
	writeAPI := client.WriteAPIBlocking(config.Org, config.Bucket)

	dp := influxdb2.NewPointWithMeasurement("download").
		AddTag("id", config.ID).
		AddField("connected", utils.Btoi(dns)).
		AddField("value", 0)

	up := influxdb2.NewPointWithMeasurement("upload").
		AddTag("id", config.ID).
		AddField("connected", utils.Btoi(dns)).
		AddField("value", 0)

	if result.Success {
		dp = dp.AddTag("server", result.Server).
			AddTag("host", result.Host).
			AddField("distance", result.Distance).
			AddField("latency", result.Latency).
			AddField("value", result.Download)

		up = influxdb2.NewPointWithMeasurement("upload").
			AddTag("server", result.Server).
			AddTag("host", result.Host).
			AddField("distance", result.Distance).
			AddField("latency", result.Latency).
			AddField("value", result.Upload)
	}

	return writeAPI.WritePoint(ctx, dp, up)
}
