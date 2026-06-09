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
