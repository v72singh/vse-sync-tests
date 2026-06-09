// SPDX-License-Identifier: GPL-2.0-or-later

package detect

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parsePTPClockIndexFromEthtool", func() {
	It("parses the legacy PTP Hardware Clock line", func() {
		out := `Time stamping parameters for eth0:
Capabilities:
	hardware-transmit
	hardware-raw-clock
PTP Hardware Clock: 2
Hardware Transmit Timestamp Modes:
	off
	on`

		index, ok := parsePTPClockIndexFromEthtool(out)
		Expect(ok).To(BeTrue())
		Expect(index).To(Equal("2"))
	})

	It("parses the hardware timestamp provider index line", func() {
		out := `Time stamping parameters for ens3f0:
Capabilities:
	hardware-transmit
	hardware-raw-clock
Hardware timestamp provider index: 0
Hardware timestamp provider qualifier: Precise (IEEE 1588 quality)
Hardware Transmit Timestamp Modes:
	off
	on`

		index, ok := parsePTPClockIndexFromEthtool(out)
		Expect(ok).To(BeTrue())
		Expect(index).To(Equal("0"))
	})

	It("ignores PTP Hardware Clock set to none", func() {
		out := `PTP Hardware Clock: none
Hardware timestamp provider index: 1`

		index, ok := parsePTPClockIndexFromEthtool(out)
		Expect(ok).To(BeTrue())
		Expect(index).To(Equal("1"))
	})

	It("returns false when no clock index is present", func() {
		out := `Time stamping parameters for eth0:
Capabilities:
	software-transmit`

		_, ok := parsePTPClockIndexFromEthtool(out)
		Expect(ok).To(BeFalse())
	})
})

func TestSortAndDeduplicateInterfacesByPTPDevice(t *testing.T) {
	t.Parallel()

	ifaces := []DetectedInterface{
		{Name: "eno8803np1", PTPClockDevicePath: "/dev/ptp0", Primary: false},
		{Name: "eno8703np0", PTPClockDevicePath: "/dev/ptp0", Primary: true},
		{Name: "enp108s0f0np0", PTPClockDevicePath: "/dev/ptp1", Primary: false},
	}
	result := sortAndDeduplicateInterfaces(ifaces)
	if len(result) != 2 {
		t.Fatalf("expected 2 interfaces after dedup, got %d", len(result))
	}
	if result[0].Name != "eno8703np0" {
		t.Fatalf("expected primary interface eno8703np0 to win ptp0 dedup, got %q", result[0].Name)
	}
}

func TestEnsureAtLeastOnePrimary(t *testing.T) {
	t.Parallel()

	ifaces := []DetectedInterface{
		{Name: "eno8703np0", Primary: false},
		{Name: "enp108s0f0np0", Primary: false},
	}
	result := ensureAtLeastOnePrimary(ifaces)
	if !result[0].Primary {
		t.Fatal("expected first interface to be promoted to primary")
	}
	if result[0].Name != "eno8703np0" {
		t.Fatalf("unexpected primary interface %q", result[0].Name)
	}
}

func TestDetect(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Detect Suite")
}
