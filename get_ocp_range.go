// This package creates a CLI that allows to derive a range of OCP versions from a given range of
// Kubernetes versions. It uses Mastermind/semver/v3 under the hood.

package getocprange

import (
	"embed"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

//go:embed kubeOpenShiftVersionMap.yaml
var content embed.FS

type versionMap struct {
	Versions []*versionMapping `yaml:"versions"`
}

type versionMapping struct {
	KubeVersion string `yaml:"kube-version"`
	OcpVersion  string `yaml:"ocp-version"`
}

// Maps a Kubernetes version to its corresponding OCP version(s). A Kubernetes
// version can map to more than one OCP version, e.g. Kubernetes 1.36 maps to
// both OCP 4.23 and OCP 5.0.
var kubeOpenShiftVersionMap map[string][]string

// Contains the highest known Kubernetes version. Used to determine if the provided Kubernetes
// range is open-ended.
var upperKubeVersion *semver.Version

// Upon initialization of this package, the kubeOpenShiftVersionMap variable is populated with the
// content of kubeOpenShiftVersionMap.yaml.
func init() {
	kubeOpenShiftVersionMap = make(map[string][]string)

	yamlFile, err := content.ReadFile("kubeOpenShiftVersionMap.yaml")
	if err != nil {
		panic(fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err))
	}

	versions := versionMap{}
	err = yaml.Unmarshal(yamlFile, &versions)
	if err != nil {
		panic(fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err))
	}

	upperKubeVersion, _ = semver.NewVersion("0.0")
	for _, versionMap := range versions.Versions {
		// Register the highest known Kubernetes version.
		kubeVersion, err := semver.NewVersion(versionMap.KubeVersion)
		if err != nil {
			panic(fmt.Sprintf("Error reading content of kubeOpenShiftVersionMap.yaml: %v", err))
		}
		if kubeVersion.GreaterThan(upperKubeVersion) {
			upperKubeVersion = kubeVersion
		}

		// Build Kubernetes version to OCP version mapping.
		kubeOpenShiftVersionMap[versionMap.KubeVersion] = append(
			kubeOpenShiftVersionMap[versionMap.KubeVersion], versionMap.OcpVersion)
	}
}

// GetOCPRange derives a range of OCP versions given a range of Kubernetes versions.
//
// To do so, it first ensures that the provided range of Kubernetes versions is a valid SemVer
// constraint using Mastermind/semver/v3. It then checks which of the known Kubernetes versions
// validate the Constraint, and registers the corresponding minimum and maximum OCP versions.
// Finally, it builds the resulting range of OCP versions.
//
// A Kubernetes version can map to several OCP versions (e.g. Kubernetes 1.36 maps to both OCP
// 4.23 and 5.0). When the provided range of Kubernetes versions matches a single Kubernetes
// version in that situation, the lowest OCP version is returned by default (e.g. "1.36" returns
// "4.23"), for compatibility with consumers expecting a single version. Ranges matching several
// Kubernetes versions keep using the true minimum and maximum OCP versions (e.g. ">=1.14 <=1.36"
// returns ">=4.2 <=5.0").
//
// This function currently doesn't support the || operator for the provided range of Kubernetes
// versions.
func GetOCPRange(kubeVersionRange string) (string, error) {
	// Return an error if the provided range of Kubernetes versions contains unsupported operators.
	if strings.Contains(kubeVersionRange, "||") {
		return "", fmt.Errorf("range %s contains unsupported operator ||", kubeVersionRange)
	}

	minOCPRange, _ := semver.NewVersion("9.9")
	maxOCPRange, _ := semver.NewVersion("0.0")

	// Number of known Kubernetes versions matching the Constraint, and the OCP versions of the
	// last matched Kubernetes version.
	matchedKubeVersions := 0
	var matchedOCPVersions []string

	// Ensure that the provided range of Kubernetes versions is a valid SemVer constraint.
	kubeVersionRangeConstraint, err := semver.NewConstraint(kubeVersionRange)
	if err != nil {
		return "", fmt.Errorf("error converting %s to Constraint: %v", kubeVersionRange, err)
	}

	for kubeVersionString, OCPVersionStrings := range kubeOpenShiftVersionMap {
		// Check which of the known Kubernetes versions validate the Constraint.
		kubeVersionObj, err := semver.NewVersion(kubeVersionString)
		if err != nil {
			return "", fmt.Errorf("error converting %s to Version: %v", kubeVersionString, err)
		}
		isInRange, _ := kubeVersionRangeConstraint.Validate(kubeVersionObj)
		if isInRange {
			matchedKubeVersions++
			matchedOCPVersions = OCPVersionStrings

			// Register the corresponding minimum and maximum OCP versions.
			for _, OCPVersionString := range OCPVersionStrings {
				OCPVersionObj, err := semver.NewVersion(OCPVersionString)
				if err != nil {
					return "", fmt.Errorf("error converting %s to Version: %v", OCPVersionString, err)
				}
				if OCPVersionObj.LessThan(minOCPRange) {
					minOCPRange = OCPVersionObj
				}
				if OCPVersionObj.GreaterThan(maxOCPRange) {
					maxOCPRange = OCPVersionObj
				}
			}
		}
	}

	// Build the resulting range of OCP versions.
	if minOCPRange.Original() == "9.9" {
		// If the minimum was never set, it means we didn't match any known Kubernetes version.
		return "", fmt.Errorf("failed to match any known Kubernetes version to the provided range %s", kubeVersionRange)
	}
	if matchedKubeVersions == 1 && len(matchedOCPVersions) > 1 {
		// If a single Kubernetes version was matched and it maps to several OCP versions, default
		// to the lowest OCP version.
		maxOCPRange = minOCPRange
	}
	if isRangeOpenEnded(kubeVersionRangeConstraint) {
		// If the provided range is open-ended, the result range should also be open-ended.
		return ">=" + minOCPRange.Original(), nil
	}
	if minOCPRange.Equal(maxOCPRange) {
		return minOCPRange.Original(), nil
	}
	return ">=" + minOCPRange.Original() + " <=" + maxOCPRange.Original(), nil
}

// isRangeOpenEnded returns a boolean set to True if the provided range of Kubernetes versions is
// open-ended (e.g. ">=1.13").
//
// To do so, we incremente the Patch value of the highest known Kubernetes Version, and check if it
// belongs to the range.
//
// TODO: This also returns True on ">=1.13 <=1.30" when upperKubeVersion=1.26.
func isRangeOpenEnded(kubeVersionRangeConstraint *semver.Constraints) bool {
	nextUpperKubeVersion := upperKubeVersion.IncMinor()
	isOpenEnded, _ := kubeVersionRangeConstraint.Validate(&nextUpperKubeVersion)
	return isOpenEnded
}
