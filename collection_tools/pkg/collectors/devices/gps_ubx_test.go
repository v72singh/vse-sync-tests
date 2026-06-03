// SPDX-License-Identifier: GPL-2.0-or-later

package devices_test

import (
	"bufio"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/remotecommand"

	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/clients"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/pkg/collectors/devices"
	"github.com/openshift-kni/vse-sync-tests/collection_tools/testutils"
)

var _ = Describe("GetGPSNav", func() {
	var clientset *clients.Clientset
	var response map[string][]byte
	BeforeEach(func() { //nolint:dupl // this is test setup code
		clientset = testutils.GetMockedClientSet(testPod)
		response = make(map[string][]byte)
		responder := func(method string, url *url.URL, options remotecommand.StreamOptions) ([]byte, []byte, error) {
			reader := bufio.NewReader(options.Stdin)
			cmd := ""
			keepReading := true
			var cmdSb30 strings.Builder
			for keepReading {
				line, prefix, _ := reader.ReadLine()
				keepReading = prefix
				cmdSb30.WriteString(string(line))
			}
			cmd += cmdSb30.String()
			return response[cmd], []byte(""), nil
		}
		clients.NewSPDYExecutor = testutils.NewFakeNewSPDYExecutor(responder, nil)
	})

	When("called GetGPSNav", func() {
		It("should return a valid GPSNav", func() {
			expectedInput := "echo '<GPS>';ubxtool -t -p NAV-STATUS -p NAV-CLOCK -p MON-RF -P 29.20;echo '</GPS>';"

			expectedOutput := strings.Join([]string{
				"<GPS>",
				"1686916187.0584",
				"UBX-MON-RF:",
				" version 0 nBlocks 2 reserved1 0 0",
				"   blockId 0 flags x0 antStatus 2 antPower 1 postStatus 0 reserved2 0 0 0 0",
				"    noisePerMS 82 agcCnt 6318 jamInd 3 ofsI 15 magI 154 ofsQ 2 magQ 145",
				"    reserved3 0 0 0",
				"   blockId 1 flags x0 antStatus 2 antPower 1 postStatus 0 reserved2 0 0 0 0",
				"    noisePerMS 49 agcCnt 6669 jamInd 2 ofsI -11 magI 146 ofsQ -1 magQ 139",
				"    reserved3 0 0 0",
				"",
				"1686916187.0584",
				"UBX-NAV-STATUS:",
				"  iTOW 474605000 gpsFix 3 flags 0xdd fixStat 0x0 flags2 0x8",
				"  ttff 25030, msss 4294967295",
				"",
				"1686916187.0586",
				"UBX-NAV-CLOCK:",
				"  iTOW 474605000 clkB -61594 clkD -56 tAcc 5 fAcc 164",
				"</GPS>",
			}, "\n")
			response[expectedInput] = []byte(expectedOutput)

			ctx, err := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer", "TestNodeName")
			Expect(err).NotTo(HaveOccurred())

			gpsInfo, err := devices.GetGPSNav(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(gpsInfo.NavStatus.Timestamp).To(Equal("2023-06-16T11:49:47.0584Z"))
			Expect(gpsInfo.NavStatus.GPSFix).To(Equal(3))

			Expect(gpsInfo.NavClock.Timestamp).To(Equal("2023-06-16T11:49:47.0586Z"))
			Expect(gpsInfo.NavClock.TimeAcc).To(Equal(5))
			Expect(gpsInfo.NavClock.FreqAcc).To(Equal(164))

			Expect(gpsInfo.AntennaDetails[0].Timestamp).To(Equal("2023-06-16T11:49:47.0584Z"))
			Expect(gpsInfo.AntennaDetails[0].BlockID).To(Equal(0))
			Expect(gpsInfo.AntennaDetails[0].Status).To(Equal(2))
			Expect(gpsInfo.AntennaDetails[0].Power).To(Equal(1))

			Expect(gpsInfo.AntennaDetails[1].Timestamp).To(Equal("2023-06-16T11:49:47.0584Z"))
			Expect(gpsInfo.AntennaDetails[1].BlockID).To(Equal(1))
			Expect(gpsInfo.AntennaDetails[1].Status).To(Equal(2))
			Expect(gpsInfo.AntennaDetails[1].Power).To(Equal(1))

		})
	})

	When("called GetGPSNav with newer ubxtool MON-RF output", func() {
		It("should parse antenna blocks", func() {
			expectedInput := "echo '<GPS>';ubxtool -t -p NAV-STATUS -p NAV-CLOCK -p MON-RF -P 29.20;echo '</GPS>';"

			expectedOutput := strings.Join([]string{
				"<GPS>",
				"1780484520.8614",
				"UBX-MON-RF:",
				" version 0 nBlocks 2 reserved1 x0",
				"   0: blockId 0 flags x0 antStatus 2 antPower 1 postStatus 0 reserved2 x0",
				"      noisePerMS 87 agcCnt 6318 jamInd 5 ofsI 8 magI 169 ofsQ 10 magQ 163",
				"      reserved3 0 0 0",
				"   1: blockId 1 flags x0 antStatus 2 antPower 1 postStatus 0 reserved2 x0",
				"      noisePerMS 50 agcCnt 6669 jamInd 2 ofsI 13 magI 168 ofsQ 7 magQ 167",
				"      reserved3 0 0 0",
				"",
				"1780484521.0667",
				"UBX-NAV-STATUS:",
				"  iTOW 298939000 gpsFix 5 flags 0xdd fixStat 0x0 flags2 0x8",
				"  ttff 34269, msss 11081275",
				"",
				"1780484521.0668",
				"UBX-NAV-CLOCK:",
				"  iTOW 298939000 clkB 709233 clkD 432 tAcc 2 fAcc 144",
				"</GPS>",
			}, "\n")
			response[expectedInput] = []byte(expectedOutput)

			ctx, err := clients.NewContainerContext(clientset, "TestNamespace", "Test", "TestContainer", "TestNodeName")
			Expect(err).NotTo(HaveOccurred())

			gpsInfo, err := devices.GetGPSNav(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(gpsInfo.AntennaDetails).To(HaveLen(2))
			Expect(gpsInfo.AntennaDetails[0].Status).To(Equal(2))
			Expect(gpsInfo.AntennaDetails[0].Power).To(Equal(1))
			Expect(gpsInfo.NavStatus.GPSFix).To(Equal(5))
		})
	})
})
