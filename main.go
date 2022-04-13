// Copyright 2022 Adam Ashley
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	webConfig    = webflag.AddFlags(kingpin.CommandLine)
	addr         = kingpin.Flag("web.listen-address", "The address to listen for HTTP requests.").Default(":9915").String()
	token        = kingpin.Flag("token", "Authorisation token to talk to the PowerPal API.").String()
	device       = kingpin.Flag("device", "The device ID of the PowerPal you wish to query.").String()
	powerpalHost = kingpin.Flag("powerpal-host", "The hostname of the Powerpal API to connect to.").Default("readings.powerpal.net").String()
	refreshTime  = kingpin.Flag("refresh", "Frequency of refresh from Powerpal API in seconds").Default("30").Int()

	// Metrics about the exporter itself
	apiDuration = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "powerpal_api_duration_seconds",
			Help: "Duration of request to Powerpal API by exporter",
		},
		[]string{"endpoint"},
	)
	apiRequestErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "powerpal_api_errors_total",
			Help: "Errors in requests to the Powerpal API",
		},
	)
)

type DeviceStats struct {
	SerialNumber           string  `json:"serial_number"`
	TotalMeterReadingCount int     `json:"total_meter_reading_count"`
	TotalWattHours         int     `json:"total_watt_hours"`
	TotalCost              float64 `json:"total_cost"`
	FirstReadingTimestamp  int     `json:"first_reading_timestamp"`
	LastReadingTimestamp   int     `json:"last_reading_timestamp"`
	LastReadingWattHours   int     `json:"last_reading_watt_hours"`
	LastReadingCost        float64 `json:"last_reading_cost"`
	AvailableDays          int     `json:"available_days"`
}

func getDeviceData(logger log.Logger) string {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://%s/api/v1/device/%s", *powerpalHost, *device), nil)
	if err != nil {
		level.Error(logger).Log("msg", "Error creating HTTP Request", "err", err)
		apiRequestErrors.Inc()
		return "unknown"
	}

	req.Header.Add("Authorization", *token)

	resp, err := client.Do(req)
	if err != nil {
		level.Error(logger).Log("msg", "Error requesting device information from API", "err", err)
		apiRequestErrors.Inc()
		return "unknown"
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			level.Error(logger).Log("msg", "Error reading API response", "err", err)
			apiRequestErrors.Inc()
			return "unknown"
		}
		return string(body)
	}

	level.Error(logger).Log("msg", fmt.Sprintf("Got status code %d from API", resp.StatusCode), "err", resp.Status)
	apiRequestErrors.Inc()
	return "unknown"
}

func watchPowerpal(registry prometheus.Registry, logger log.Logger) {
	availableDays := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_available_days",
		Help: "The number of days of data available within Powerpal for this device",
	})
	firstReading := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_reading_timestamp_first",
		Help: "The timestamp of the first reading",
	})
	lastReading := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_reading_timestamp_last",
		Help: "The timestamp of the last reading",
	})
	cost := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_cost",
		Help: "The cost at the last reading",
	})
	wattHours := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_watt_hours",
		Help: "The watt hours being consumed at the last reading",
	})
	totalCost := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_cost_total",
		Help: "The total cost recorded by this device",
	})
	totalWattHours := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_watt_hours_total",
		Help: "The total watt hours recorded by this device",
	})
	totalReadings := promauto.NewGauge(prometheus.GaugeOpts{
		Name: "powerpal_reading_count",
		Help: "The total number of readings recorded by this device",
	})

	registry.MustRegister(availableDays)
	registry.MustRegister(firstReading)
	registry.MustRegister(lastReading)
	registry.MustRegister(cost)
	registry.MustRegister(wattHours)
	registry.MustRegister(totalCost)
	registry.MustRegister(totalWattHours)
	registry.MustRegister(totalReadings)
	registry.MustRegister(apiDuration)
	registry.MustRegister(apiRequestErrors)

	go func() {
		for {
			apiJsonData := getDeviceData(logger)
			if apiJsonData != "unknown" {
				var powerpalMetrics DeviceStats
				if err := json.Unmarshal([]byte(apiJsonData), &powerpalMetrics); err != nil {
					level.Error(logger).Log("msg", "Error parsing API response", "err", err)
					apiRequestErrors.Inc()
				} else {
					availableDays.Set(float64(powerpalMetrics.AvailableDays))
					firstReading.Set(float64(powerpalMetrics.FirstReadingTimestamp))
					lastReading.Set(float64(powerpalMetrics.LastReadingTimestamp))
					cost.Set(float64(powerpalMetrics.LastReadingCost))
					wattHours.Set(float64(powerpalMetrics.LastReadingWattHours))
					totalCost.Set(float64(powerpalMetrics.TotalCost))
					totalWattHours.Set(float64(powerpalMetrics.TotalWattHours))
					totalReadings.Set(float64(powerpalMetrics.TotalMeterReadingCount))
				}
			}
			time.Sleep(time.Duration(*refreshTime) * time.Second)
		}
	}()
}

func main() {
	promlogConfig := &promlog.Config{}
	kingpin.Version(version.Print("powerpal-exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	level.Info(logger).Log("msg", "Starting powerpal_exporter", "version", version.Info())
	level.Info(logger).Log("build_context", version.BuildContext())

	apiDuration.WithLabelValues("api_device")
	//apiDuration.WithLabelValues("api_meter_reading")

	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector("powerpal_exporter"))
	watchPowerpal(*r, logger)
	handler := promhttp.HandlerFor(r, promhttp.HandlerOpts{})

	http.Handle("/powerpal", handler)
	http.Handle("/metrics", promhttp.Handler())
	level.Info(logger).Log("msg", "Listening on address", "address", *addr)
	srv := &http.Server{Addr: *addr}
	if err := web.ListenAndServe(srv, *webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP Server", "err", err)
		os.Exit(1)
	}
}
