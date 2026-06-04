// SPDX-License-Identifier: GPL-2.0-or-later

package contexts

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/clients"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/devices"
)

// NetlinkExec pairs the command execution context for DPLL netlink with optional pod lifecycle.
type NetlinkExec struct {
	Exec      clients.ExecContext
	Lifecycle *clients.ContainerCreationExecContext
}

// ResolveNetlinkExecContext uses the linuxptp-daemon pod when it already has ynl tools,
// otherwise schedules the dpll-debug pod (with PodSecurity labels for privileged profile).
func ResolveNetlinkExecContext(
	clientset *clients.Clientset,
	ptpNodeName string,
	unmanagedDebugPod bool,
) (*NetlinkExec, error) {
	ptpCtx, err := GetPTPDaemonContext(clientset, ptpNodeName)
	if err == nil && devices.NetlinkToolsAvailable(ptpCtx) {
		log.Info("Using linuxptp-daemon container for DPLL netlink collection")
		return &NetlinkExec{Exec: ptpCtx}, nil
	}

	podCtx, err := GetNetlinkContext(clientset, ptpNodeName, unmanagedDebugPod)
	if err != nil {
		return nil, err
	}

	log.Debug("Using dpll-debug pod for DPLL netlink collection")

	return &NetlinkExec{Exec: podCtx, Lifecycle: podCtx}, nil
}

func (n *NetlinkExec) Start() error {
	if n.Lifecycle == nil {
		return nil
	}

	err := n.Lifecycle.CreatePodAndWait()
	if err != nil {
		return fmt.Errorf("dpll netlink debug pod failed to start: %w", err)
	}

	return nil
}

func (n *NetlinkExec) Stop() error {
	if n.Lifecycle == nil {
		return nil
	}

	err := n.Lifecycle.DeletePodAndWait()
	if err != nil {
		return fmt.Errorf("dpll netlink debug pod failed to stop: %w", err)
	}

	return nil
}
