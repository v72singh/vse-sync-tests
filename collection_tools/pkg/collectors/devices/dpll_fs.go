// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/callbacks"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/clients"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/fetcher"
)

const (
	unitConversionFactor = 100
)

type DevFilesystemDPLLInfo struct {
	PreferSMA1 bool
	Timestamp  string  `fetcherKey:"date"          json:"timestamp"`
	EECState   string  `fetcherKey:"dpll_0_state"  json:"eecstate"`
	PPSState   string  `fetcherKey:"dpll_1_state"  json:"state"`
	PPSOffset  float64 `fetcherKey:"dpll_1_offset" json:"terror"`
}

// AnalyserJSON returns the json expected by the analysers
func (dpllInfo *DevFilesystemDPLLInfo) GetAnalyserFormat() ([]*callbacks.AnalyserFormatType, error) {
	sampleID := "dpll/time-error"
	if dpllInfo.PreferSMA1 {
		sampleID = "dpll-sma1/time-error"
	}

	formatted := callbacks.AnalyserFormatType{
		ID: sampleID,
		Data: map[string]any{
			"timestamp": dpllInfo.Timestamp,
			"eecstate":  normalizeDPLLState(dpllInfo.EECState),
			"state":     normalizeDPLLState(dpllInfo.PPSState),
			// Convert to nano seconds
			"terror": dpllInfo.PPSOffset / unitConversionFactor,
		},
	}

	return []*callbacks.AnalyserFormatType{&formatted}, nil
}

var (
	dpllFSFetcher map[string]*fetcher.Fetcher
)

func init() {
	dpllFSFetcher = make(map[string]*fetcher.Fetcher)
}

func postProcessDPLLFilesystem(result map[string]string) (map[string]any, error) {
	processedResult := make(map[string]any)

	offset, err := strconv.ParseFloat(result["dpll_1_offset"], 32)
	if err != nil {
		return processedResult, fmt.Errorf("failed converting dpll_1_offset %w to an int", err)
	}

	processedResult["dpll_1_offset"] = offset

	return processedResult, nil
}

// BuildFilesystemDPLLInfoFetcher popluates the fetcher required for
// collecting the DPLLInfo
func BuildFilesystemDPLLInfoFetcher(interfaceName string) error { //nolint:dupl // Further dedup risks be too abstract or fragile
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key:     "dpll_0_state",
				Command: fmt.Sprintf("cat /sys/class/net/%s/device/dpll_0_state", interfaceName),
				Trim:    true,
			},
			{
				Key:     "dpll_1_state",
				Command: fmt.Sprintf("cat /sys/class/net/%s/device/dpll_1_state", interfaceName),
				Trim:    true,
			},
			{
				Key:     "dpll_1_offset",
				Command: fmt.Sprintf("cat /sys/class/net/%s/device/dpll_1_offset", interfaceName),
				Trim:    true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll: %w", err)
	}

	dpllFSFetcher[interfaceName] = fetcherInst
	fetcherInst.SetPostProcessor(postProcessDPLLFilesystem)

	return nil
}

// GetDevDPLLFilesystemInfo returns the device DPLL info for an interface.
func GetDevDPLLFilesystemInfo(ctx clients.ExecContext, interfaceName string, preferSMA1 bool) (*DevFilesystemDPLLInfo, error) {
	dpllInfo := &DevFilesystemDPLLInfo{PreferSMA1: preferSMA1}

	fetcherInst, fetchedInstanceOk := dpllFSFetcher[interfaceName]
	if !fetchedInstanceOk {
		err := BuildFilesystemDPLLInfoFetcher(interfaceName)
		if err != nil {
			return dpllInfo, err
		}

		fetcherInst, fetchedInstanceOk = dpllFSFetcher[interfaceName]
		if !fetchedInstanceOk {
			return dpllInfo, errors.New("failed to create fetcher for DPLLInfo")
		}
	}

	err := fetcherInst.Fetch(ctx, dpllInfo)
	if err != nil {
		log.Debugf("failed to fetch dpllInfo %s", err.Error())
		return dpllInfo, fmt.Errorf("failed to fetch dpllInfo %w", err)
	}

	fillFilesystemPPSState(dpllInfo, interfaceName)

	return dpllInfo, nil
}

func fillFilesystemPPSState(dpllInfo *DevFilesystemDPLLInfo, interfaceName string) {
	if normalizeDPLLState(dpllInfo.PPSState) != unknownDPLLState {
		return
	}

	if isLockedDPLLState(dpllInfo.EECState) {
		log.Debugf(
			"PPS DPLL state unavailable via sysfs for %s; using EEC state %s",
			interfaceName,
			normalizeDPLLState(dpllInfo.EECState),
		)
		dpllInfo.PPSState = dpllInfo.EECState
	}
}

func IsDPLLFileSystemPresent(ctx clients.ExecContext, interfaceName string) (bool, error) {
	command := fmt.Sprintf(
		"test -f /sys/class/net/%s/device/dpll_0_state && "+
			"test -f /sys/class/net/%s/device/dpll_1_state && "+
			"test -f /sys/class/net/%s/device/dpll_1_offset && echo yes",
		interfaceName, interfaceName, interfaceName,
	)

	stdout, _, err := ctx.ExecCommand([]string{"/usr/bin/sh", "-c", command})
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(stdout) == "yes", nil
}
