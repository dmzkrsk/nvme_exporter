package main

import (
	"context"
	"encoding/json"
	"log"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prometheus/client_golang/prometheus"
)

type App struct {
	b             *backoff.ExponentialBackOff
	checkInterval time.Duration
	knownDevices  map[string]bool
	deviceData    map[string]NVMeDevice
}

func (a *App) Run(ctx context.Context) {
	for {
		var tm *time.Timer

		success := a.runLoop(ctx)

		if !success {
			d := a.b.NextBackOff()
			tm = time.NewTimer(d)
		} else {
			a.b.Reset()
			tm = time.NewTimer(a.checkInterval)
			loopRuns.Inc()
		}

		select {
		case <-tm.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) runLoop(context.Context) bool {
	jsonOutput, err := exec.Command("sudo", "nvme", "list", "-o", "json").Output()
	if err != nil {
		log.Printf("[ERR]  error running nvme list command: %s", err)
		return false
	}

	var devices NVMeDevices
	err = json.Unmarshal(jsonOutput, &devices)
	if err != nil {
		log.Printf("[ERR]  error unmarshalling nvme list output: %s", err)
		return false
	}

	var hasErrors bool

	for _, device := range devices.Devices {
		jsonOutput, err = exec.Command("sudo", "nvme", "smart-log", device.DevicePath, "-o", "json").Output() // nolint:gosec
		if err != nil {
			log.Printf("[ERR]  error running nvme smart-log command for %s: %s", device.DevicePath, err)
			hasErrors = true
			continue
		}

		var stats *NVMeStats
		stats, err = parseNVMeStats(jsonOutput)
		if err != nil {
			log.Printf("[ERR]  error unmarshalling nvme smart-log output for %s: %s", device.DevicePath, err)
			hasErrors = true
			continue
		}

		a.knownDevices[device.DevicePath] = true

		prevInfo := a.deviceData[device.DevicePath]
		if prevInfo.ModelNumber != device.ModelNumber ||
			prevInfo.SerialNumber != device.SerialNumber ||
			prevInfo.Firmware != device.Firmware {
			nvmeInfo.DeletePartialMatch(prometheus.Labels{
				"device": device.DevicePath,
			})
		}

		a.deviceData[device.DevicePath] = device

		nvmeInfo.WithLabelValues(device.DevicePath, device.ModelNumber, device.SerialNumber, device.Firmware).Set(1)

		nvmeUsedBytes.WithLabelValues(device.DevicePath).Set(float64(device.UsedBytes))
		nvmeMaximumLBA.WithLabelValues(device.DevicePath).Set(float64(device.MaximumLBA))
		nvmePhysicalSize.WithLabelValues(device.DevicePath).Set(float64(device.PhysicalSize))
		nvmeSectorSize.WithLabelValues(device.DevicePath).Set(float64(device.SectorSize))

		nvmeCriticalWarning.WithLabelValues(device.DevicePath).Set(float64(stats.CriticalWarning))
		nvmeTemperature.WithLabelValues(device.DevicePath).Set(float64(stats.Temperature))
		nvmeAvailSpare.WithLabelValues(device.DevicePath).Set(float64(stats.AvailSpare))
		nvmeSpareThresh.WithLabelValues(device.DevicePath).Set(float64(stats.SpareThresh))
		nvmePercentUsed.WithLabelValues(device.DevicePath).Set(float64(stats.PercentUsed))
		nvmeEnduranceGrpCriticalWarningSummary.WithLabelValues(device.DevicePath).Set(float64(stats.EnduranceGrpCriticalWarningSummary))
		nvmeDataUnitsRead.WithLabelValues(device.DevicePath).Set(float64(stats.DataUnitsRead))
		nvmeDataUnitsWritten.WithLabelValues(device.DevicePath).Set(float64(stats.DataUnitsWritten))
		nvmeHostReadCommands.WithLabelValues(device.DevicePath).Set(float64(stats.HostReadCommands))
		nvmeHostWriteCommands.WithLabelValues(device.DevicePath).Set(float64(stats.HostWriteCommands))
		nvmeControllerBusyTime.WithLabelValues(device.DevicePath).Set(float64(stats.ControllerBusyTime))
		nvmePowerCycles.WithLabelValues(device.DevicePath).Set(float64(stats.PowerCycles))
		nvmePowerOnHours.WithLabelValues(device.DevicePath).Set(float64(stats.PowerOnHours))
		nvmeUnsafeShutdowns.WithLabelValues(device.DevicePath).Set(float64(stats.UnsafeShutdowns))
		nvmeMediaErrors.WithLabelValues(device.DevicePath).Set(float64(stats.MediaErrors))
		nvmeNumErrLogEntries.WithLabelValues(device.DevicePath).Set(float64(stats.NumErrLogEntries))
		nvmeWarningTempTime.WithLabelValues(device.DevicePath).Set(float64(stats.WarningTempTime))
		nvmeCriticalCompTime.WithLabelValues(device.DevicePath).Set(float64(stats.CriticalCompTime))
		nvmeThmTemp1TransCount.WithLabelValues(device.DevicePath).Set(float64(stats.ThmTemp1TransCount))
		nvmeThmTemp2TransCount.WithLabelValues(device.DevicePath).Set(float64(stats.ThmTemp2TransCount))
		nvmeThmTemp1TotalTime.WithLabelValues(device.DevicePath).Set(float64(stats.ThmTemp1TotalTime))
		nvmeThmTemp2TotalTime.WithLabelValues(device.DevicePath).Set(float64(stats.ThmTemp2TotalTime))
	}

	for d, ok := range a.knownDevices {
		if ok {
			a.knownDevices[d] = false
			continue
		}

		delete(a.knownDevices, d)

		nvmeInfo.DeletePartialMatch(prometheus.Labels{
			"device": d,
		})

		nvmeUsedBytes.DeleteLabelValues(d)
		nvmeMaximumLBA.DeleteLabelValues(d)
		nvmePhysicalSize.DeleteLabelValues(d)
		nvmeSectorSize.DeleteLabelValues(d)

		nvmeCriticalWarning.DeleteLabelValues(d)
		nvmeTemperature.DeleteLabelValues(d)
		nvmeAvailSpare.DeleteLabelValues(d)
		nvmeSpareThresh.DeleteLabelValues(d)
		nvmePercentUsed.DeleteLabelValues(d)
		nvmeEnduranceGrpCriticalWarningSummary.DeleteLabelValues(d)
		nvmeDataUnitsRead.DeleteLabelValues(d)
		nvmeDataUnitsWritten.DeleteLabelValues(d)
		nvmeHostReadCommands.DeleteLabelValues(d)
		nvmeHostWriteCommands.DeleteLabelValues(d)
		nvmeControllerBusyTime.DeleteLabelValues(d)
		nvmePowerCycles.DeleteLabelValues(d)
		nvmePowerOnHours.DeleteLabelValues(d)
		nvmeUnsafeShutdowns.DeleteLabelValues(d)
		nvmeMediaErrors.DeleteLabelValues(d)
		nvmeNumErrLogEntries.DeleteLabelValues(d)
		nvmeWarningTempTime.DeleteLabelValues(d)
		nvmeCriticalCompTime.DeleteLabelValues(d)
		nvmeThmTemp1TransCount.DeleteLabelValues(d)
		nvmeThmTemp2TransCount.DeleteLabelValues(d)
		nvmeThmTemp1TotalTime.DeleteLabelValues(d)
		nvmeThmTemp2TotalTime.DeleteLabelValues(d)
	}

	return !hasErrors
}
