package main

// Export nvme smart-log metrics in prometheus format

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	Addr          string        `envconfig:"listen_addr" default:":21405"`
	CheckInterval time.Duration `envconfig:"check_interval" default:"1m"`
}

var labels = []string{"device"}

const appName = "nvme_exporter"

var (
	nvmeInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_device_info",
			Help: "Critical warnings for the state of the controller",
		},
		append(labels, "model", "serial", "firmware"),
	)

	nvmeUsedBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_used_bytes",
			Help: "Number of bytes used on the device",
		},
		labels,
	)
	nvmeMaximumLBA = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_maximum_lba",
			Help: "Maximum Logical Block Address",
		},
		labels,
	)
	nvmePhysicalSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_physical_size",
			Help: "Physical size of the device in bytes",
		},
		labels,
	)
	nvmeSectorSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_sector_size",
			Help: "Sector size in bytes",
		},
		labels,
	)

	nvmeCriticalWarning = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_critical_warning",
			Help: "Critical warnings for the state of the controller",
		},
		labels,
	)
	nvmeTemperature = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_temperature",
			Help: "Temperature in degrees celsius",
		},
		labels,
	)
	nvmeAvailSpare = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_avail_spare",
			Help: "Normalized percentage of remaining spare capacity available",
		},
		labels,
	)
	nvmeSpareThresh = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_spare_thresh",
			Help: "Async event completion may occur when avail spare < threshold",
		},
		labels,
	)
	nvmePercentUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_percent_used",
			Help: "Vendor specific estimate of the percentage of life used",
		},
		labels,
	)
	nvmeEnduranceGrpCriticalWarningSummary = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_endurance_grp_critical_warning_summary",
			Help: "Critical warnings for the state of endurance groups",
		},
		labels,
	)
	nvmeDataUnitsRead = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_data_units_read",
			Help: "Number of 512 byte data units host has read",
		},
		labels,
	)
	nvmeDataUnitsWritten = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_data_units_written",
			Help: "Number of 512 byte data units the host has written",
		},
		labels,
	)
	nvmeHostReadCommands = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_host_read_commands",
			Help: "Number of read commands completed",
		},
		labels,
	)
	nvmeHostWriteCommands = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_host_write_commands",
			Help: "Number of write commands completed",
		},
		labels,
	)
	nvmeControllerBusyTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_controller_busy_time",
			Help: "Amount of time in minutes controller busy with IO commands",
		},
		labels,
	)
	nvmePowerCycles = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_power_cycles",
			Help: "Number of power cycles",
		},
		labels,
	)
	nvmePowerOnHours = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_power_on_hours",
			Help: "Number of power on hours",
		},
		labels,
	)
	nvmeUnsafeShutdowns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_unsafe_shutdowns",
			Help: "Number of unsafe shutdowns",
		},
		labels,
	)
	nvmeMediaErrors = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_media_errors",
			Help: "Number of unrecovered data integrity errors",
		},
		labels,
	)
	nvmeNumErrLogEntries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_num_err_log_entries",
			Help: "Lifetime number of error log entries",
		},
		labels,
	)
	nvmeWarningTempTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_warning_temp_time",
			Help: "Amount of time in minutes temperature > warning threshold",
		},
		labels,
	)
	nvmeCriticalCompTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_critical_comp_time",
			Help: "Amount of time in minutes temperature > critical threshold",
		},
		labels,
	)
	nvmeThmTemp1TransCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_thm_temp1_trans_count",
			Help: "Number of times controller transitioned to lower power",
		},
		labels,
	)
	nvmeThmTemp2TransCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_thm_temp2_trans_count",
			Help: "Number of times controller transitioned to lower power",
		},
		labels,
	)
	nvmeThmTemp1TotalTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_thm_temp1_trans_time",
			Help: "Total number of seconds controller transitioned to lower power",
		},
		labels,
	)
	nvmeThmTemp2TotalTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvme_thm_temp2_trans_time",
			Help: "Total number of seconds controller transitioned to lower power",
		},
		labels,
	)
)

func init() {
	prometheus.MustRegister(
		nvmeInfo,
		nvmeUsedBytes,
		nvmeMaximumLBA,
		nvmePhysicalSize,
		nvmeSectorSize,
		nvmeCriticalWarning,
		nvmeTemperature,
		nvmeAvailSpare,
		nvmeSpareThresh,
		nvmePercentUsed,
		nvmeEnduranceGrpCriticalWarningSummary,
		nvmeDataUnitsRead,
		nvmeDataUnitsWritten,
		nvmeHostReadCommands,
		nvmeHostWriteCommands,
		nvmeControllerBusyTime,
		nvmePowerCycles,
		nvmePowerOnHours,
		nvmeUnsafeShutdowns,
		nvmeMediaErrors,
		nvmeNumErrLogEntries,
		nvmeWarningTempTime,
		nvmeCriticalCompTime,
		nvmeThmTemp1TransCount,
		nvmeThmTemp2TransCount,
		nvmeThmTemp1TotalTime,
		nvmeThmTemp2TotalTime,
	)
}

func main() {
	var cfg Config

	err := envconfig.Process("nvme_exporter", &cfg)
	if err != nil {
		log.Fatal("[ERR]  processing envconfig:", err)
	}

	// termination

	stopCh := make(chan os.Signal, 2)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// run

	var (
		serverDone  = make(chan struct{})
		s           = newMetricsHTTPServer(cfg.Addr)
		ctx, cancel = context.WithCancel(context.Background())
		wg          sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer close(serverDone)
		defer wg.Done()

		log.Printf("[INFO] http server started on %s", s.GetAddr())
		sErr := s.Run()
		if sErr != nil {
			log.Printf("[ERR]  server run exited: %s", sErr)
		}
	}()

	app := App{
		b: backoff.NewExponentialBackOff(
			backoff.WithInitialInterval(time.Second),
			backoff.WithMaxInterval(time.Second*10),
		),
		checkInterval: cfg.CheckInterval,
		knownDevices:  make(map[string]bool),
		deviceData:    make(map[string]NVMeDevice),
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.Run(ctx)
	}()

	select {
	case <-serverDone:
		log.Print("[WARN] http server unexpectedly closed")
	case <-stopCh:
		log.Print("[INFO] terminating app")
		cancel()
		go func() {
			<-stopCh
			log.Print("[WARN] force exit app")
			os.Exit(1)
		}()

		log.Print("[INFO] terminating http server")
		err := s.Shutdown()
		if err != nil {
			log.Printf("[ERR]  server shutdown failed: %s", err)
		} else {
			log.Print("[INFO] wait for http server to exit")
			<-serverDone
			log.Print("[INFO] http server exited properly")
		}
	}

	log.Print("[INFO] waiting for processes to complete")
	wg.Wait()
	log.Print("[INFO] successfully exited")
}
