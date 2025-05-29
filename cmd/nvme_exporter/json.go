package main

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

type NVMeDevice struct {
	NameSpace    int    `json:"NameSpace"`
	DevicePath   string `json:"DevicePath"`
	Firmware     string `json:"Firmware"`
	ModelNumber  string `json:"ModelNumber"`
	SerialNumber string `json:"SerialNumber"`
	UsedBytes    uint64 `json:"UsedBytes"`
	MaximumLBA   uint64 `json:"MaximumLBA"`
	PhysicalSize uint64 `json:"PhysicalSize"`
	SectorSize   uint64 `json:"SectorSize"`
}

type NVMeDevices struct {
	Devices []NVMeDevice `json:"Devices"`
}

type NVMeStats struct {
	CriticalWarning                    int
	Temperature                        int
	AvailSpare                         int
	SpareThresh                        int
	PercentUsed                        int
	EnduranceGrpCriticalWarningSummary int
	DataUnitsRead                      uint64
	DataUnitsWritten                   uint64
	HostReadCommands                   uint64
	HostWriteCommands                  uint64
	ControllerBusyTime                 uint64
	PowerCycles                        uint64
	PowerOnHours                       uint64
	UnsafeShutdowns                    uint64
	MediaErrors                        uint64
	NumErrLogEntries                   uint64
	WarningTempTime                    int
	CriticalCompTime                   int
	ThmTemp1TransCount                 int
	ThmTemp2TransCount                 int
	ThmTemp1TotalTime                  int
	ThmTemp2TotalTime                  int
}

type rawNVMeStats struct {
	CriticalWarning                    int    `json:"critical_warning"`
	Temperature                        int    `json:"temperature"`
	AvailSpare                         int    `json:"avail_spare"`
	SpareThresh                        int    `json:"spare_thresh"`
	PercentUsed                        int    `json:"percent_used"`
	EnduranceGrpCriticalWarningSummary int    `json:"endurance_grp_critical_warning_summary"`
	DataUnitsRead                      string `json:"data_units_read"`
	DataUnitsWritten                   string `json:"data_units_written"`
	HostReadCommands                   string `json:"host_read_commands"`
	HostWriteCommands                  string `json:"host_write_commands"`
	ControllerBusyTime                 string `json:"controller_busy_time"`
	PowerCycles                        string `json:"power_cycles"`
	PowerOnHours                       string `json:"power_on_hours"`
	UnsafeShutdowns                    string `json:"unsafe_shutdowns"`
	MediaErrors                        string `json:"media_errors"`
	NumErrLogEntries                   string `json:"num_err_log_entries"`
	WarningTempTime                    int    `json:"warning_temp_time"`
	CriticalCompTime                   int    `json:"critical_comp_time"`
	ThmTemp1TransCount                 int    `json:"thm_temp1_trans_count"`
	ThmTemp2TransCount                 int    `json:"thm_temp2_trans_count"`
	ThmTemp1TotalTime                  int    `json:"thm_temp1_total_time"`
	ThmTemp2TotalTime                  int    `json:"thm_temp2_total_time"`
}

func parseNumberWithCommas(s string) (uint64, error) {
	clean := strings.ReplaceAll(s, ",", "")
	return strconv.ParseUint(clean, 10, 64)
}

func parseNVMeStats(data []byte) (*NVMeStats, error) {
	var raw rawNVMeStats
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	parse := func(s string) uint64 {
		n, _ := parseNumberWithCommas(s)
		return n
	}

	return &NVMeStats{
		CriticalWarning:                    raw.CriticalWarning,
		Temperature:                        int(math.Round(float64(raw.Temperature) - 273.15)),
		AvailSpare:                         raw.AvailSpare,
		SpareThresh:                        raw.SpareThresh,
		PercentUsed:                        raw.PercentUsed,
		EnduranceGrpCriticalWarningSummary: raw.EnduranceGrpCriticalWarningSummary,
		DataUnitsRead:                      parse(raw.DataUnitsRead),
		DataUnitsWritten:                   parse(raw.DataUnitsWritten),
		HostReadCommands:                   parse(raw.HostReadCommands),
		HostWriteCommands:                  parse(raw.HostWriteCommands),
		ControllerBusyTime:                 parse(raw.ControllerBusyTime),
		PowerCycles:                        parse(raw.PowerCycles),
		PowerOnHours:                       parse(raw.PowerOnHours),
		UnsafeShutdowns:                    parse(raw.UnsafeShutdowns),
		MediaErrors:                        parse(raw.MediaErrors),
		NumErrLogEntries:                   parse(raw.NumErrLogEntries),
		WarningTempTime:                    raw.WarningTempTime,
		CriticalCompTime:                   raw.CriticalCompTime,
		ThmTemp1TransCount:                 raw.ThmTemp1TransCount,
		ThmTemp2TransCount:                 raw.ThmTemp2TransCount,
		ThmTemp1TotalTime:                  raw.ThmTemp1TotalTime,
		ThmTemp2TotalTime:                  raw.ThmTemp2TotalTime,
	}, nil
}
