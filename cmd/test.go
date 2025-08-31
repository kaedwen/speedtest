package cmd

import (
	"context"
	"net"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/kaedwen/speedtest/pkg/config"
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

type TestHandler struct {
	lg     *zap.Logger
	client influxdb2.Client
}

func NewTestHandler(lg *zap.Logger) (*TestHandler, error) {
	return &TestHandler{lg, influxdb2.NewClient(config.InfluxHost, config.InfluxToken)}, nil
}

func NewTestCommand(lg *zap.Logger) *cobra.Command {
	cmd := cobra.Command{
		Use: "test",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			hndl, err := NewTestHandler(lg)
			if err != nil {
				return err
			}

			results, dns, err := hndl.test(ctx)
			if err == nil {
				for _, result := range results {
					err := hndl.save(ctx, &result, dns)
					if err != nil {
						return err
					}
				}
			} else {
				err := hndl.save(ctx, &TestResult{
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

func (h *TestHandler) test(ctx context.Context) ([]TestResult, bool, error) {
	var stc = speedtest.New()

	user, _ := stc.FetchUserInfo()
	h.lg.Info("user meta", zap.Any("info", user))

	serverList, _ := stc.FetchServers()
	targets, _ := serverList.FindServer(nil)

	results := make([]TestResult, 0, len(targets))

	for _, s := range targets {
		if err := s.PingTestContext(ctx, nil); err != nil {
			return nil, false, err
		}

		if err := s.DownloadTestContext(ctx); err != nil {
			return nil, false, err
		}

		if err := s.UploadTestContext(ctx); err != nil {
			return nil, false, err
		}

		h.lg.Info("result", zap.Duration("Latency", s.Latency), zap.Float64("Download", s.DLSpeed.Mbps()), zap.Float64("Upload", s.ULSpeed.Mbps()))

		results = append(results, TestResult{
			Success:  true,
			Server:   s.Name,
			Host:     s.Host,
			Latency:  s.Latency,
			Download: float64(s.DLSpeed) * 8,
			Upload:   float64(s.ULSpeed) * 8,
			Distance: s.Distance,
		})

		s.Context.Reset()
	}

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			return d.DialContext(ctx, network, net.JoinHostPort(config.TestDNSTarget, "53"))
		},
	}

	dns := true
	ip, err := r.LookupHost(context.Background(), config.TestDNSHost)
	if err != nil || len(ip) == 0 {
		dns = false
	}

	h.lg.Info("speedtest ok", zap.Bool("DNS", dns))

	return results, dns, nil
}

func (h *TestHandler) save(ctx context.Context, result *TestResult, dns bool) error {
	writeAPI := h.client.WriteAPIBlocking(config.InfluxOrg, config.InfluxBucket)

	p := influxdb2.NewPointWithMeasurement(config.InfluxMeasurement).
		AddField("connected", dns).
		AddField("distance", float64(0)).
		AddField("latency", float64(0)).
		AddField("download", float64(0)).
		AddField("upload", float64(0))

	if result.Success {
		p.AddTag("server", result.Server).
			AddTag("host", result.Host).
			AddField("distance", result.Distance).
			AddField("latency", float64(result.Latency.Milliseconds())).
			AddField("download", result.Download).
			AddField("upload", result.Upload)
	}

	if err := writeAPI.WritePoint(ctx, p); err != nil {
		return err
	}

	h.lg.Info("result saved")

	return nil
}
