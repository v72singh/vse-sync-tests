// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"strings"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/devices"
)

const (
	deviceFirmwareID          = TGMEnvVerPath + "/nic-firmware/"
	deviceFirmwareDescription = "Verify NIC firmware version"
)

var (
	MinFirmwareVersion = "4.20"
)

func NewDeviceFirmware(ptpDevInfo *devices.PTPDeviceInfo, strict bool) *VersionCheck {
	parts := strings.Split(ptpDevInfo.FirmwareVersion, " ")

	return &VersionCheck{
		id:           deviceFirmwareID,
		Version:      ptpDevInfo.FirmwareVersion,
		checkVersion: parts[0],
		MinVersion:   MinFirmwareVersion,
		description:  deviceFirmwareDescription,
		order:        deviceFirmwareOrdering,
		strict:       strict,
	}
}
