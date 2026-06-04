// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/callbacks"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/contexts"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/devices"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

type DPLLNetlinkCollector struct {
	*baseCollector

	netlink           *contexts.NetlinkExec
	interfaceName     string
	params            devices.NetlinkParameters
	preferSMA1        bool
}

const (
	DPLLNetlinkCollectorName = "DPLL-Netlink"
	DPLLNetlinkInfo          = "dpll-info-nl"
)

// Start sets up the collector so it is ready to be polled
func (dpll *DPLLNetlinkCollector) Start() error {
	dpll.running = true

	err := dpll.netlink.Start()
	if err != nil {
		return fmt.Errorf("dpll netlink collector failed to start: %w", err)
	}

	log.Debug("dpll.interfaceName: ", dpll.interfaceName)

	netlinkParams, err := devices.GetNetlinkParameters(dpll.netlink.Exec, dpll.interfaceName, dpll.preferSMA1)
	if err != nil {
		return fmt.Errorf("dpll netlink collector failed to find clock id: %w", err)
	}

	log.Debug("clockIDStuct.ClockID: ", netlinkParams.ClockID)

	err = devices.ValidateNetlinkDPLLSupported(dpll.netlink.Exec, netlinkParams)
	if err != nil {
		return fmt.Errorf("dpll netlink collector not supported: %w", err)
	}

	dpll.params = netlinkParams

	return nil
}

// polls for the dpll info then passes it to the callback
func dpllNetlinkPoller(dpll *DPLLNetlinkCollector) func() (callbacks.OutputType, error) {
	return func() (callbacks.OutputType, error) {
		return devices.GetDevDPLLNetlinkInfo(dpll.netlink.Exec, dpll.params) //nolint:wrapcheck //no point wrapping this
	}
}

// Poll collects information from the cluster then
// calls the callback.Call to allow that to persist it
func (dpll *DPLLNetlinkCollector) Poll(resultsChan chan PollResult, wg *utils.WaitGroupCount) {
	defer wg.Done()

	errorsToReturn := make([]error, 0)

	err := dpll.poll()
	if err != nil {
		errorsToReturn = append(errorsToReturn, err)
	}

	resultsChan <- PollResult{
		CollectorName: DPLLNetlinkCollectorName,
		Errors:        errorsToReturn,
	}
}

// CleanUp stops a running collector
func (dpll *DPLLNetlinkCollector) CleanUp() error {
	dpll.running = false

	err := dpll.netlink.Stop()
	if err != nil {
		return fmt.Errorf("dpll netlink collector failed to clean up: %w", err)
	}

	return nil
}

// Returns a new DPLLNetlinkCollector from the CollectionConstuctor Factory
func NewDPLLNetlinkCollector(constructor *CollectionConstructor) (Collector, error) {
	netlinkExec, err := contexts.ResolveNetlinkExecContext(
		constructor.Clientset,
		constructor.PTPNodeName,
		constructor.UnmanagedDebugPod,
	)
	if err != nil {
		return &DPLLNetlinkCollector{}, fmt.Errorf("failed to create DPLLNetlinkCollector: %w", err)
	}

	collector := &DPLLNetlinkCollector{
		baseCollector: newBaseCollector(
			constructor.PollInterval,
			false,
			constructor.Callback,
			DPLLNetlinkCollectorName,
			DPLLNetlinkInfo,
		),
		interfaceName: constructor.PTPInterface,
		netlink:       netlinkExec,
		preferSMA1:    constructor.DPLLPreferSMA1,
	}
	collector.poller = dpllNetlinkPoller(collector)

	err = collector.Start()
	if err != nil {
		collector.CleanUp()
		return nil, utils.NewRequirementsNotMetError(fmt.Errorf("dpll netlink collector unavailable: %w", err))
	}

	return collector, nil
}
