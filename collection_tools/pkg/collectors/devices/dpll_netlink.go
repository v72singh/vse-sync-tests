// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/callbacks"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/clients"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/fetcher"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

var states = map[string]string{
	"unknown":       "-1",
	"invalid":       "0",
	"freerun":       "1",
	"locked":        "2",
	"locked-ho-acq": "3",
	"holdover":      "4",
}

const (
	dpllYNLCLIPath  = "/linux/tools/net/ynl/cli.py"
	dpllYNLSpecPath = "/linux/Documentation/netlink/specs/dpll.yaml"
	dpllJSONEncoder = "python3 /root/custom_scripts/json_encoder.py"
	maxDPLLPinProbe     = 32
	maxDPLLDeviceProbe  = 8

	OnePPSLabel = "GNSS-1PPS"
	SMA1Label   = "SMA1"
	SMA2Label   = "SMA2"

	OnePPSSubtype  = "dpll"
	SMA1Subtype    = "dpll-sma1"
	UnknownSubtype = "unknown"

	InputDirection = "input"
	ConnectedState = "connected"

	EECOffsetParentID      = 0
	PPSOffesetParentID     = 1
	DPLLPhaseOffsetDivider = 1000
)

type DevNetlinkDPLLInfo struct {
	PinType   string
	Timestamp string `fetcherKey:"date"       json:"timestamp"`
	EECState  string `fetcherKey:"eec"        json:"eecstate"`
	PPSState  string `fetcherKey:"pps"        json:"state"`
	PPSOffset int64  `fetcherKey:"pps_offset" json:"terror"`
	EECOffset int64  `fetcherKey:"eec_offset" json:"eecterror"`
}

func convertNetlinkOffset(offset int64) float64 {
	// Convert to nano seconds with 3 decimal places
	return float64(int64(math.Round(float64(offset/DPLLPhaseOffsetDivider)))) / 1000 //nolint:mnd // this is just for decimal places
}

// AnalyserJSON returns the json expected by the analysers
func (dpllInfo *DevNetlinkDPLLInfo) GetAnalyserFormat() ([]*callbacks.AnalyserFormatType, error) {
	subType := UnknownSubtype

	switch dpllInfo.PinType {
	case OnePPSLabel:
		subType = OnePPSSubtype
	case SMA1Label:
		subType = SMA1Subtype
	}

	formatted := callbacks.AnalyserFormatType{
		ID: subType + "/time-error",
		Data: map[string]any{
			"timestamp": dpllInfo.Timestamp,
			"eecstate":  dpllInfo.EECState,
			"state":     dpllInfo.PPSState,
			"terror":    convertNetlinkOffset(dpllInfo.PPSOffset),
			"eecterror": convertNetlinkOffset(dpllInfo.EECOffset),
		},
	}

	return []*callbacks.AnalyserFormatType{&formatted}, nil
}

type NetlinkStateEntry struct {
	LockStatus string `json:"lock-status"` //nolint:tagliatelle // not my choice
	Driver     string `json:"module-name"` //nolint:tagliatelle // not my choice
	ClockType  string `json:"type"`        //nolint:tagliatelle // not my choice
	ClockID    uint64 `json:"clock-id"`    //nolint:tagliatelle // not my choice
	ID         int    `json:"id"`          //nolint:tagliatelle // not my choice
}

// # Example output
// [{'clock-id': 5799633565435100136,
//   'id': 0,
//   'lock-status': 'locked-ho-acq',
//   'mode': 'automatic',
//   'mode-supported': ['automatic'],
//   'module-name': 'ice',
//   'type': 'eec'},
//  {'clock-id': 5799633565435100136,
//   'id': 1,
//   'lock-status': 'locked-ho-acq',
//   'mode': 'automatic',
//   'mode-supported': ['automatic'],
//   'module-name': 'ice',
//   'type': 'pps'}]

type NetlinkPin struct {
	Type                 string                            `json:"type"`                //nolint:tagliatelle // not my choice
	ModuleName           string                            `json:"module-name"`         //nolint:tagliatelle // not my choice
	Label                string                            `json:"board-label"`         //nolint:tagliatelle // not my choice
	Capabilities         []string                          `json:"capabilities"`        //nolint:tagliatelle // not my choice
	FrequenciesSupported []*NetlinkFrequencySupportedRange `json:"frequency-supported"` //nolint:tagliatelle // not my choice
	ParentDevices        []*NetlinkParentDevice            `json:"parent-device"`       //nolint:tagliatelle // not my choice
	ParentPins           []*NetlinkParentPin               `json:"parent-pin"`          //nolint:tagliatelle // not my choice
	ClockID              uint64                            `json:"clock-id"`            //nolint:tagliatelle // not my choice
	Frequency            uint64                            `json:"frequency"`           //nolint:tagliatelle // not my choice
	ID                   int32                             `json:"id"`                  //nolint:tagliatelle // not my choice
	PhaseAdjust          int32                             `json:"phase-adjust"`        //nolint:tagliatelle // not my choice
	PhaseAdjustMax       int32                             `json:"phase-adjust-max"`    //nolint:tagliatelle // not my choice
	PhaseAdjustMin       int32                             `json:"phase-adjust-min"`    //nolint:tagliatelle // not my choice
}

type NetlinkParentDevice struct {
	Direction   string `json:"direction"`    //nolint:tagliatelle // not my choice
	State       string `json:"state"`        //nolint:tagliatelle // not my choice
	ParentID    int    `json:"parent-id"`    //nolint:tagliatelle // not my choice
	PhaseOffset int64  `json:"phase-offset"` //nolint:tagliatelle // not my choice
	Prio        int    `json:"prio"`         //nolint:tagliatelle // not my choice
}

type NetlinkParentPin struct {
	State    string `json:"state"`     //nolint:tagliatelle // not my choice
	ParentID int32  `json:"parent-id"` //nolint:tagliatelle // not my choice
}

type NetlinkFrequencySupportedRange struct {
	Max int32 `json:"frequency-max"` //nolint:tagliatelle // not my choice
	Min int32 `json:"frequency-min"` //nolint:tagliatelle // not my choice
}

// # Example output
// {
// 	'board-label': 'GNSS-1PPS',
// 	'capabilities': 6,
// 	'clock-id': 5799633565433967608,
// 	'frequency': 1,
// 	'frequency-supported': [
// 		{
// 			'frequency-max': 1,
// 			'frequency-min': 1
// 		}
// 	],
// 	'id': 6,
// 	'module-name': 'ice',
// 	'parent-device': [
// 		{
// 			'direction': 'input',
// 			'parent-id': 0,
// 			'phase-offset': 406616064733390,
// 			'prio': 0,
// 			'state': 'connected'
// 		},
// 		{
// 			'direction': 'input',
// 			'parent-id': 1,
// 			'phase-offset': -1870360,
// 			'prio': 0,
// 			'state': 'connected'
// 		}
// 	],
// 	'phase-adjust': 0,
// 	'phase-adjust-max': 16723,
// 	'phase-adjust-min': -16723,
// 	'type': 'gnss'
// },

var (
	dpllClockIDFetcher   map[string]*fetcher.Fetcher
	deviceDumpWarnOnce   sync.Once
	pinDumpWarnOnce      sync.Once
)

func init() {
	dpllClockIDFetcher = make(map[string]*fetcher.Fetcher)
}

func runTimestamp(ctx clients.ExecContext) (string, error) {
	stdout, _, err := ctx.ExecCommand([]string{"date", "+%s.%N"})
	if err != nil {
		return "", fmt.Errorf("failed to read timestamp: %w", err)
	}

	return formatTimestampAsRFC3339Nano(stdout)
}

func collectDPLLNetlinkSample(ctx clients.ExecContext, params NetlinkParameters) (map[string]any, error) {
	processedResult := make(map[string]any)

	deviceIDs := params.DeviceIDs
	if len(deviceIDs) == 0 {
		deviceIDs = []int{0, 1}
	}

	for _, deviceID := range deviceIDs {
		out, err := runDPLLYNLCommand(ctx, fmt.Sprintf("--do device-get --json '{\"id\": %d}'", deviceID))
		if err != nil || out == "" {
			log.Debugf("skipping DPLL device poll for id %d: %v", deviceID, err)
			continue
		}

		var entry NetlinkStateEntry

		err = json.Unmarshal([]byte(out), &entry)
		if err != nil {
			log.Debugf("failed to parse DPLL device %d: %v", deviceID, err)
			continue
		}

		if entry.ClockID != params.ClockID {
			continue
		}

		state, ok := states[entry.LockStatus]
		if !ok {
			log.Errorf("Unknown state: %s", entry.LockStatus)
			state = "-1"
		}

		processedResult[entry.ClockType] = state
	}

	pinOut, err := runDPLLYNLCommand(ctx, fmt.Sprintf("--do pin-get --json '{\"id\": %d}'", params.OffsetPin))
	if err != nil {
		return processedResult, fmt.Errorf("failed to read offset pin: %w", err)
	}

	pin := NetlinkPin{}

	err = json.Unmarshal([]byte(pinOut), &pin)
	if err != nil {
		return processedResult, fmt.Errorf("failed to unmarshal netlink pin output: %w", err)
	}

	for _, parentPin := range pin.ParentDevices {
		switch parentPin.ParentID % 2 {
		case EECOffsetParentID:
			processedResult["ecc_offset"] = parentPin.PhaseOffset
		case PPSOffesetParentID:
			processedResult["pps_offset"] = parentPin.PhaseOffset
		}
	}

	return processedResult, nil
}

// ValidateNetlinkDPLLSupported checks that required ynl operations work on this node.
func ValidateNetlinkDPLLSupported(ctx clients.ExecContext, params NetlinkParameters) error {
	if len(params.DeviceIDs) == 0 {
		return errors.New("no DPLL device IDs resolved for this interface")
	}

	_, err := runDPLLYNLCommand(
		ctx,
		fmt.Sprintf("--do device-get --json '{\"id\": %d}'", params.DeviceIDs[0]),
	)
	if err != nil {
		return fmt.Errorf("device %d unavailable: %w", params.DeviceIDs[0], err)
	}

	_, err = runDPLLYNLCommand(ctx, fmt.Sprintf("--do pin-get --json '{\"id\": %d}'", params.OffsetPin))
	if err != nil {
		return fmt.Errorf("offset pin %d unavailable: %w", params.OffsetPin, err)
	}

	return nil
}

func resolveDeviceIDsForClock(ctx clients.ExecContext, clockID uint64) ([]int, error) {
	devicesJSON, err := discoverDevicesJSON(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]NetlinkStateEntry, 0)

	err = json.Unmarshal(devicesJSON, &entries)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal netlink device list: %w", err)
	}

	deviceIDs := make([]int, 0, len(entries))

	for _, entry := range entries {
		if entry.ClockID == clockID {
			deviceIDs = append(deviceIDs, entry.ID)
		}
	}

	if len(deviceIDs) == 0 {
		log.Warnf("No DPLL devices matched clock ID %d, defaulting to device ids 0 and 1", clockID)
		return []int{0, 1}, nil
	}

	return deviceIDs, nil
}

// GetDevDPLLNetlinkInfo returns the device DPLL info for an interface.
func GetDevDPLLNetlinkInfo(ctx clients.ExecContext, params NetlinkParameters) (*DevNetlinkDPLLInfo, error) {
	dpllInfo := &DevNetlinkDPLLInfo{PinType: params.PinType}

	timestamp, err := runTimestamp(ctx)
	if err != nil {
		return dpllInfo, err
	}

	dpllInfo.Timestamp = timestamp

	processed, err := collectDPLLNetlinkSample(ctx, params)
	if err != nil {
		return dpllInfo, fmt.Errorf("failed to fetch dpllInfo via netlink: %w", err)
	}

	if eecState, ok := processed["eec"].(string); ok {
		dpllInfo.EECState = eecState
	}

	if ppsState, ok := processed["pps"].(string); ok {
		dpllInfo.PPSState = ppsState
	}

	if ppsOffset, ok := processed["pps_offset"].(int64); ok {
		dpllInfo.PPSOffset = ppsOffset
	}

	if eecOffset, ok := processed["ecc_offset"].(int64); ok {
		dpllInfo.EECOffset = eecOffset
	}

	return dpllInfo, nil
}

func runDPLLYNLCommand(ctx clients.ExecContext, args string) (string, error) {
	command := fmt.Sprintf(
		"%s --spec %s %s | %s",
		dpllYNLCLIPath,
		dpllYNLSpecPath,
		args,
		dpllJSONEncoder,
	)

	stdout, stderr, err := ctx.ExecCommand([]string{"/usr/bin/sh", "-c", command})
	if stderr != "" {
		log.Debugf("DPLL ynl stderr: %s", stderr)
	}

	if err != nil {
		return "", fmt.Errorf("dpll ynl command failed: %w", err)
	}

	return strings.TrimSpace(stdout), nil
}

func discoverDevicesJSON(ctx clients.ExecContext) ([]byte, error) {
	out, err := runDPLLYNLCommand(ctx, "--dump device-get")
	if err == nil && out != "" {
		entries := make([]NetlinkStateEntry, 0)

		unmarshalErr := json.Unmarshal([]byte(out), &entries)
		if unmarshalErr == nil && len(entries) > 0 {
			return []byte(out), nil
		}

		log.Debugf("DPLL device-get dump returned unusable data: %v", unmarshalErr)
	}

	deviceDumpWarnOnce.Do(func() {
		log.Warn("DPLL device-get dump unavailable, probing devices individually (once at setup)")
	})

	return probeDevicesIndividually(ctx)
}

func probeDevicesIndividually(ctx clients.ExecContext) ([]byte, error) {
	entries := make([]NetlinkStateEntry, 0)

	for id := 0; id < maxDPLLDeviceProbe; id++ {
		out, err := runDPLLYNLCommand(ctx, fmt.Sprintf("--do device-get --json '{\"id\": %d}'", id))
		if err != nil || out == "" {
			continue
		}

		var entry NetlinkStateEntry

		unmarshalErr := json.Unmarshal([]byte(out), &entry)
		if unmarshalErr != nil {
			log.Debugf("skipping DPLL device %d: %v", id, unmarshalErr)
			continue
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, errors.New("no DPLL devices found via individual device-get")
	}

	return json.Marshal(entries)
}

func discoverPinsJSON(ctx clients.ExecContext) ([]byte, error) {
	out, err := runDPLLYNLCommand(ctx, "--dump pin-get")
	if err == nil && out != "" {
		entries := make([]*NetlinkPin, 0)

		unmarshalErr := json.Unmarshal([]byte(out), &entries)
		if unmarshalErr == nil && len(entries) > 0 {
			return []byte(out), nil
		}

		log.Debugf("DPLL pin-get dump returned unusable data: %v", unmarshalErr)
	}

	pinDumpWarnOnce.Do(func() {
		log.Warn("DPLL pin-get dump unavailable, probing pins individually (once at setup)")
	})

	return probePinsIndividually(ctx)
}

func probePinsIndividually(ctx clients.ExecContext) ([]byte, error) {
	pins := make([]*NetlinkPin, 0)

	for id := int32(0); id < maxDPLLPinProbe; id++ {
		out, err := runDPLLYNLCommand(ctx, fmt.Sprintf("--do pin-get --json '{\"id\": %d}'", id))
		if err != nil || out == "" {
			continue
		}

		var pin NetlinkPin

		unmarshalErr := json.Unmarshal([]byte(out), &pin)
		if unmarshalErr != nil {
			log.Debugf("skipping DPLL pin %d: %v", id, unmarshalErr)
			continue
		}

		pins = append(pins, &pin)
	}

	if len(pins) == 0 {
		return nil, utils.NewRequirementsNotMetError(errors.New("no pins found via individual pin-get"))
	}

	return json.Marshal(pins)
}

func BuildNetlinkInfoFetcher(interfaceName string) error {
	fetcherInst, err := fetcher.FetcherFactory(
		[]*clients.Cmd{dateCmd},
		[]fetcher.AddCommandArgs{
			{
				Key: "dpll-netlink-clock-serial-number",
				Command: fmt.Sprintf(
					`export IFNAME=%s; export BUSID=$(readlink /sys/class/net/$IFNAME/device | xargs basename | cut -d ':' -f 2,3);`+
						` echo $(lspci -v | grep $BUSID -A 20 |grep 'Serial Number' | awk '{print $NF}' | tr -d '-')`,
					interfaceName,
				),
				Trim: true,
			},
		},
	)
	if err != nil {
		log.Errorf("failed to create fetcher for dpll clock ID: %s", err.Error())
		return fmt.Errorf("failed to create fetcher for dpll clock ID: %w", err)
	}

	fetcherInst.SetPostProcessor(postProcessDPLLNetlinkClockID)
	dpllClockIDFetcher[interfaceName] = fetcherInst

	return nil
}

func selectPin(pinsJSON []byte, clockID uint64) (int32, string, error) { //nolint:funlen,gocritic,cyclop // allow slightly longer function for sake of readability
	entries := make([]*NetlinkPin, 0)

	err := json.Unmarshal(pinsJSON, &entries)
	if err != nil {
		return 0, "", fmt.Errorf("failed to unmarshal netlink output: %s", err.Error())
	}

	if len(entries) == 0 {
		return 0, "", utils.NewRequirementsNotMetError(errors.New("no pins found"))
	}

	var onePPSPin, sma1Pin *NetlinkPin

	log.Debug("entries: ", entries)

	for _, pin := range entries {
		if pin.ClockID != clockID {
			continue
		}

		switch pin.Label {
		case OnePPSLabel:
			onePPSPin = pin
		case SMA1Label:
			sma1Pin = pin
		}
	}

	choosePPS := onePPSPin != nil

	if choosePPS {
		for _, parentDev := range onePPSPin.ParentDevices {
			if parentDev.State != ConnectedState {
				choosePPS = false
				break
			}
		}
	}

	chooseSMA1 := sma1Pin != nil

	if chooseSMA1 {
		for _, parentDev := range sma1Pin.ParentDevices {
			if parentDev.Direction != InputDirection || parentDev.State != ConnectedState {
				chooseSMA1 = false
				break
			}
		}
	}

	//nolint:gocritic // this is clearer
	if choosePPS {
		return onePPSPin.ID, OnePPSLabel, nil
	}

	if chooseSMA1 {
		return sma1Pin.ID, SMA1Label, nil
	}

	// Fall back to labeled pins without requiring parent link state.
	if onePPSPin != nil {
		log.Debug("Using GNSS-1PPS pin without connected parent-device state")
		return onePPSPin.ID, OnePPSLabel, nil
	}

	if sma1Pin != nil {
		log.Debug("Using SMA1 pin without connected parent-device state")
		return sma1Pin.ID, SMA1Label, nil
	}

	// Last resort: first pin associated with this clock.
	for _, pin := range entries {
		if pin.ClockID == clockID {
			label := pin.Label
			if label == "" {
				label = UnknownSubtype
			}

			log.Warnf("Using DPLL pin %d (%s) as offset pin", pin.ID, label)

			return pin.ID, label, nil
		}
	}

	return 0, "", utils.NewRequirementsNotMetError(errors.New("failed to determine correct offset pin"))
}

func postProcessDPLLNetlinkClockID(result map[string]string) (map[string]any, error) {
	processedResult := make(map[string]any)

	clockID, err := strconv.ParseUint(result["dpll-netlink-clock-serial-number"], 16, 64)
	if err != nil {
		return processedResult, fmt.Errorf("failed to parse int for clock id: %w", err)
	}

	processedResult["clockID"] = clockID

	return processedResult, nil
}

type NetlinkParameters struct {
	Timestamp  string `fetcherKey:"date"      json:"timestamp"`
	PinType    string `fetcherKey:"pinType"   json:"pinType"`
	ClockID    uint64 `fetcherKey:"clockID"   json:"clockId"`
	OffsetPin  int32  `fetcherKey:"offsetPin" json:"offsetPin"`
	DeviceIDs  []int  `json:"deviceIds"`
}

func GetNetlinkParameters(ctx clients.ExecContext, interfaceName string) (NetlinkParameters, error) {
	netlinkInfo := NetlinkParameters{}

	fetcherInst, fetchedInstanceOk := dpllClockIDFetcher[interfaceName]
	if !fetchedInstanceOk {
		err := BuildNetlinkInfoFetcher(interfaceName)
		if err != nil {
			return netlinkInfo, err
		}

		fetcherInst, fetchedInstanceOk = dpllClockIDFetcher[interfaceName]
		if !fetchedInstanceOk {
			return netlinkInfo, errors.New("failed to create fetcher for DPLLInfo using netlink interface")
		}
	}

	err := fetcherInst.Fetch(ctx, &netlinkInfo)
	if err != nil {
		return netlinkInfo, fmt.Errorf("failed to fetch netlink info %w", err)
	}

	pinsJSON, err := discoverPinsJSON(ctx)
	if err != nil {
		return netlinkInfo, fmt.Errorf("failed to discover dpll pins: %w", err)
	}

	offsetPinID, pinType, err := selectPin(pinsJSON, netlinkInfo.ClockID)
	if err != nil {
		return netlinkInfo, err
	}

	netlinkInfo.OffsetPin = offsetPinID
	netlinkInfo.PinType = pinType

	deviceIDs, err := resolveDeviceIDsForClock(ctx, netlinkInfo.ClockID)
	if err != nil {
		return netlinkInfo, fmt.Errorf("failed to resolve dpll device ids: %w", err)
	}

	netlinkInfo.DeviceIDs = deviceIDs

	return netlinkInfo, nil
}
