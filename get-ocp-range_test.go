package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func getUpperOCPVersion() string {
	return kubeOpenShiftVersionMap[upperKubeVersion.Original()]
}

// TODO: prerelease ?
// TODO: Tilde (~) Range Comparisons ?
// TODO: Caret (^) Range Comparisons ?
// TODO: != operator ?

var _ = Describe("Unit tests for getOCPRange", func() {
	DescribeTable("Providing a valid range of Kubernetes Version",
	func(kubeVersionRange string, expectedOCPResultRange string) {
		OCPResultRange, err := GetOCPRange(kubeVersionRange)
		Expect(err).NotTo(HaveOccurred())
		Expect(OCPResultRange).To(Equal(expectedOCPResultRange))
	},
	// Single versions
	Entry("When providing a single version", "1.14", "4.2"),
	Entry("When providing a single version with patch value to zero", "1.14.0", "4.2"),
	Entry("When providing a single version with leading 'v'", "v1.25.x", "4.12"),
	Entry("When providing a single version with with wildcard", "1.25.x", "4.12"),
	
	// Open-ended ranges
	Entry("When providing a range with a lower limit only, using space", ">= 1.14", ">=4.2"),
	Entry("When providing a range with a lower limit only, not using space", ">=1.14", ">=4.2"),
	PEntry("When providing a range with an upper limit only, using space", "<= 1.20", "<=4.7"), // TODO: current output is 4.1 - 4.7
	PEntry("When providing a range with an upper limit only, not using space", "<=1.20", "<=4.7"), // TODO: current output is 4.1 - 4.7
	Entry("When providing a range with a lower limit equal to the highest known Kubernetes Version", ">=" + upperKubeVersion.Original(), ">=" + getUpperOCPVersion()),

	
	// Ranges
	Entry("When providing a range using a hyphen", "1.14 - 1.20", ">=4.2 <=4.7"),
	Entry("When providing a range using lower and upper limits", ">=1.14 <=1.20", ">=4.2 <=4.7"),
	Entry("When providing a range with an upper limit equal to the highest known Kubernetes Version", ">=1.14 <=" + upperKubeVersion.Original(), ">=4.2 <=" + getUpperOCPVersion()),
	)

	DescribeTable("Providing an invalid range of Kubernetes Version",
	func(kubeVersionRange string) {
		_, err := GetOCPRange(kubeVersionRange)
		Expect(err).To(HaveOccurred())
		// Expect(err).To(MatchError(KuberVersionProcessingError))
	},
	Entry("When providing a version lower than the lowest known Kubernetes Version", "0.1"),
	Entry("When providing a version higher than the highest known Kubernetes Version", "8.9"),
	Entry("When providing a single version with patch value other than zero", "1.14.1"),
	PEntry("When providing only a major version", "1"), // TODO: currently getting ">=4.1"
	Entry("When providing an invalid range", ">=1.20 <=1.14"),
	Entry("When the range contains an unsupported operator ||", "1.14 || 1.16"),
	)

})