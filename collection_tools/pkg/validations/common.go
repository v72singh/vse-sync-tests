// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/mod/semver"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/utils"
)

const (
	TGMTestIDBase   = "https://github.com/openshift-kni/vse-sync-tests/tree/main/tests"
	TGMEnvModelPath = TGMTestIDBase + "/environment/model"
	TGMEnvVerPath   = TGMTestIDBase + "/environment/version"
	TGMSyncEnvPath  = TGMTestIDBase + "/sync/G.8272/environment/status"
)

const (
	clusterVersionOrdering int = iota
	ptpOperatorVersionOrdering
	gpsdVersionOrdering
	deviceDetailsOrdering
	deviceDriverVersionOrdering
	deviceFirmwareOrdering
	gnssModuleOrdering
	gnssVersionOrdering
	gnssProtOrdering
	hasGNSSDevicesOrdering
	gnssConnectedToAntOrdering
	gnssReceivingDataOrdering
	configuredForGrandMasterOrdering
)

type VersionCheck struct {
	id           string `json:"-"`
	Version      string `json:"version"`
	checkVersion string `json:"-"`
	MinVersion   string `json:"expected"`
	description  string `json:"-"`
	order        int    `json:"-"`
	strict       bool   `json:"-"`
}

func normalizeSemverVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")

	return "v" + strings.ReplaceAll(version, "_", "-")
}

func (verCheck *VersionCheck) Verify() error {
	if !verCheck.strict {
		return nil
	}

	ver := normalizeSemverVersion(verCheck.checkVersion)
	if !semver.IsValid(ver) {
		return fmt.Errorf("could not parse version %s", ver)
	}

	if semver.Compare(ver, normalizeSemverVersion(verCheck.MinVersion)) < 0 {
		return utils.NewInvalidEnvError(
			fmt.Errorf("unexpected version: %s < %s", verCheck.checkVersion, verCheck.MinVersion),
		)
	}

	return nil
}

func (verCheck *VersionCheck) GetID() string {
	return verCheck.id
}

func (verCheck *VersionCheck) GetDescription() string {
	return verCheck.description
}

func (verCheck *VersionCheck) GetData() any { //nolint:ireturn // data will vary for each validation
	return verCheck
}

func (verCheck *VersionCheck) GetOrder() int {
	return verCheck.order
}

type VersionWithError struct {
	Error   error  `json:"fetchError"`
	Version string `json:"version"`
}

func MarshalVersionAndError(ver *VersionWithError) ([]byte, error) {
	var err any
	if ver.Error != nil {
		err = ver.Error.Error()
	}

	marsh, marshalErr := json.Marshal(&struct {
		Error   any    `json:"fetchError"`
		Version string `json:"version"`
	}{
		Version: ver.Version,
		Error:   err,
	})

	return marsh, fmt.Errorf("failed to marshal VersionWithError %w", marshalErr)
}

type VersionWithErrorCheck struct {
	Error error
	VersionCheck
}

func (verCheck *VersionWithErrorCheck) MarshalJSON() ([]byte, error) {
	return MarshalVersionAndError(&VersionWithError{
		Version: verCheck.Version,
		Error:   verCheck.Error,
	})
}

func (verCheck *VersionWithErrorCheck) Verify() error {
	if !verCheck.strict {
		return nil
	}

	if verCheck.Error != nil {
		return verCheck.Error
	}

	return verCheck.VersionCheck.Verify()
}
