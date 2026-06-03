// SPDX-License-Identifier: GPL-2.0-or-later

package collectors

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/contexts"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/devices"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

const (
	DPLLCollectorName = "DPLL"
)

// Returns a new DPLLCollector from the CollectionConstuctor Factory
func NewDPLLCollector(constructor *CollectionConstructor) (Collector, error) {
	ctx, err := contexts.GetPTPDaemonContext(constructor.Clientset, constructor.PTPNodeName)
	if err != nil {
		return &DPLLNetlinkCollector{}, fmt.Errorf("failed to create DPLLCollector: %w", err)
	}

	dpllFSExists, err := devices.IsDPLLFileSystemPresent(ctx, constructor.PTPInterface)
	log.Debug("DPLL FS exists: ", dpllFSExists)

	if dpllFSExists && err == nil {
		return NewDPLLFilesystemCollector(constructor)
	}

	collector, netlinkErr := NewDPLLNetlinkCollector(constructor)
	if netlinkErr == nil {
		return collector, nil
	}

	log.Warnf("DPLL netlink collector unavailable for %s: %v", constructor.PTPInterface, netlinkErr)

	dpllFSExists, fsErr := devices.IsDPLLFileSystemPresent(ctx, constructor.PTPInterface)
	if dpllFSExists && fsErr == nil {
		log.Infof("Using DPLL sysfs collector for %s after netlink setup failed", constructor.PTPInterface)
		return NewDPLLFilesystemCollector(constructor)
	}

	var missingRequirements *utils.RequirementsNotMetError
	if errors.As(netlinkErr, &missingRequirements) {
		return nil, netlinkErr
	}

	return nil, utils.NewRequirementsNotMetError(fmt.Errorf("DPLL collector unavailable: %w", netlinkErr))
}

func init() {
	RegisterCollector(DPLLCollectorName, NewDPLLCollector, optional)
}
