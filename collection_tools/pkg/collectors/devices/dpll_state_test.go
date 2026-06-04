// SPDX-License-Identifier: GPL-2.0-or-later

package devices

import (
	"testing"
)

func TestNormalizeDPLLState(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"locked", "2"},
		{"locked-ho-acq", "3"},
		{"3", "3"},
		{"", "-1"},
		{"bogus", "-1"},
	}

	for _, tc := range cases {
		if got := normalizeDPLLState(tc.in); got != tc.want {
			t.Errorf("normalizeDPLLState(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDPLLDeviceTypeKey(t *testing.T) {
	t.Parallel()

	if dpllDeviceTypeKey("phase", 1) != "pps" {
		t.Fatal("expected phase to map to pps")
	}

	if dpllDeviceTypeKey("", 0) != "eec" {
		t.Fatal("expected device 0 to map to eec")
	}
}

func TestDevNetlinkDPLLInfoPreferSMA1AnalyserID(t *testing.T) {
	t.Parallel()

	info := &DevNetlinkDPLLInfo{
		PinType:    OnePPSLabel,
		PreferSMA1: true,
		Timestamp:  "2026-06-04T00:00:00Z",
		EECState:   "3",
		PPSState:   "3",
	}

	formats, err := info.GetAnalyserFormat()
	if err != nil {
		t.Fatal(err)
	}

	if len(formats) != 1 || formats[0].ID != "dpll-sma1/time-error" {
		t.Fatalf("expected dpll-sma1/time-error, got %#v", formats)
	}
}

func TestFillFilesystemPPSStateUsesEECWhenPPSUnknown(t *testing.T) {
	t.Parallel()

	info := &DevFilesystemDPLLInfo{
		EECState: "locked-ho-acq",
		PPSState: "-1",
	}

	fillFilesystemPPSState(info, "ens1f0")

	if normalizeDPLLState(info.PPSState) != "3" {
		t.Fatalf("expected PPS state 3 from EEC, got %q", info.PPSState)
	}
}

func TestFillFilesystemPPSStateLeavesKnownNonLockedPPS(t *testing.T) {
	t.Parallel()

	info := &DevFilesystemDPLLInfo{
		EECState: "2",
		PPSState: "10",
	}

	fillFilesystemPPSState(info, "ens1f0")

	if info.PPSState != "10" {
		t.Fatalf("expected PPS state unchanged, got %q", info.PPSState)
	}
}
