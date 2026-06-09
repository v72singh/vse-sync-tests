// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import (
	"testing"
)

func TestDeviceDetailsStrictIntelNIC(t *testing.T) {
	t.Parallel()

	nonE810 := &DeviceDetails{
		VendorID: "0x8086",
		DeviceID: "0x1234",
		strict:   true,
	}
	if err := nonE810.Verify(); err == nil {
		t.Fatal("expected strict E810 check to fail for non-reference device ID")
	}

	relaxed := &DeviceDetails{
		VendorID: "0x8086",
		DeviceID: "0x1234",
		strict:   false,
	}
	if err := relaxed.Verify(); err != nil {
		t.Fatalf("expected relaxed check to pass, got %v", err)
	}

	e810 := &DeviceDetails{
		VendorID: VendorIntel,
		DeviceID: E810WesportChannel,
		strict:   true,
	}
	if err := e810.Verify(); err != nil {
		t.Fatalf("expected E810 device to pass strict check, got %v", err)
	}
}
