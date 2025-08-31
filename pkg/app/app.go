package app

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/kaedwen/speedtest/pkg/config"
	"github.com/robfig/cron/v3"
	"github.com/showwin/speedtest-go/speedtest"
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

type Application struct {
	influxdb2.Client
	log *zap.Logger
	c   atomic.Int64
}

func NewApplication(log *zap.Logger) *Application {
	return &Application{
		Client: influxdb2.NewClient(config.InfluxHost, config.InfluxToken),
		log:    log,
	}
}

func (a *Application) Run(ctx context.Context) error {
	c := cron.New(cron.WithSeconds())
	_, err := c.AddFunc(config.TestSchedule, func() {
		log := a.log.With(zap.Int64("run", a.c.Add(1)), zap.Time("at", time.Now()))

		results, dns, err := a.test(ctx, log)
		if err != nil {
			if err := a.save(ctx, log, &TestResult{
				Success: false,
			}, dns); err != nil {
				log.Error("failed to save result", zap.Error(err))
			}

			return
		}

		for _, result := range results {
			if err := a.save(ctx, log, &result, dns); err != nil {
				log.Error("failed to save result", zap.Error(err))
				return
			}
		}
	})

	if err != nil {
		return err
	}

	c.Start()

	return nil
}

func (a *Application) test(ctx context.Context, log *zap.Logger) ([]TestResult, bool, error) {
	stc := speedtest.New()

	user, _ := stc.FetchUserInfo()
	log.Info("user meta", zap.Any("info", user))

	serverList, _ := stc.FetchServers()
	targets, _ := serverList.FindServer(nil)

	results := make([]TestResult, 0, len(targets))

	for _, s := range targets {
		log.Info("running tests", zap.String("server", s.Name), zap.String("host", s.Host), zap.Float64("distance", s.Distance))

		log.Debug("ping test")
		if err := s.PingTestContext(ctx, nil); err != nil {
			log.Error("ping test failed", zap.Error(err))
			return nil, false, err
		}

		log.Debug("download test")
		if err := s.DownloadTestContext(ctx); err != nil {
			log.Error("download test failed", zap.Error(err))
			return nil, false, err
		}

		log.Debug("upload test")
		if err := s.UploadTestContext(ctx); err != nil {
			log.Error("upload test failed", zap.Error(err))
			return nil, false, err
		}

		log.Info("result", zap.Duration("Latency", s.Latency), zap.Float64("Download", s.DLSpeed.Mbps()), zap.Float64("Upload", s.ULSpeed.Mbps()))

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

	log.Info("DNS test", zap.String("host", config.TestDNSHost))
	dns := true
	ips, err := r.LookupHost(context.Background(), config.TestDNSHost)
	if err != nil || len(ips) == 0 {
		log.Error("DNS test failed", zap.Strings("ips", ips), zap.Error(err))
		dns = false
	}

	log.Info("speedtest ok", zap.Bool("DNS", dns))

	return results, dns, nil
}

func (a *Application) save(ctx context.Context, log *zap.Logger, result *TestResult, dns bool) error {
	writeAPI := a.WriteAPIBlocking(config.InfluxOrg, config.InfluxBucket)

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

	log.Info("result saved")

	return nil
}
