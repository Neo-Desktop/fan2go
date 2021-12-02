package hwmon

import (
	"errors"
	"fmt"
	"github.com/markusressel/fan2go/internal/fans"
	"github.com/markusressel/fan2go/internal/sensors"
	"github.com/md14454/gosensors"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	BusTypeIsa     = 1
	BusTypePci     = 2
	BusTypeVirtual = 4
	BusTypeAcpi    = 5
	BusTypeHid     = 6
)

type HwMonController struct {
	Name     string
	DType    string
	Modalias string
	Platform string
	Path     string

	Fans    []*fans.HwMonFan
	Sensors []*sensors.HwmonSensor
}

func GetChips() []*HwMonController {
	gosensors.Init()
	defer gosensors.Cleanup()
	chips := gosensors.GetDetectedChips()

	var list []*HwMonController

	for i := 0; i < len(chips); i++ {
		chip := chips[i]

		var identifier = computeIdentifier(chip)
		dType := getDeviceType(chip.Path)
		modalias := getDeviceModalias(chip.Path)
		platform := findPlatform(chip.Path)
		if len(platform) <= 0 {
			platform = identifier
		}

		fansList := GetFans(chip)
		sensorsList := GetTempSensors(chip)

		if len(fansList) <= 0 && len(sensorsList) <= 0 {
			continue
		}

		c := &HwMonController{
			Name:     identifier,
			DType:    dType,
			Modalias: modalias,
			Platform: platform,
			Path:     chip.Path,
			Fans:     fansList,
			Sensors:  sensorsList,
		}
		list = append(list, c)
	}

	return list
}

// getDeviceName read the name of a device
func getDeviceName(devicePath string) string {
	namePath := devicePath + "/name"
	content, _ := ioutil.ReadFile(namePath)
	name := string(content)
	return strings.TrimSpace(name)
}

// getDeviceModalias read the modalias of a device
func getDeviceModalias(devicePath string) string {
	modaliasPath := devicePath + "/device/modalias"
	content, _ := ioutil.ReadFile(modaliasPath)
	return strings.TrimSpace(string(content))
}

// getDeviceType read the type of a device
func getDeviceType(devicePath string) string {
	modaliasPath := devicePath + "/device/type"
	content, _ := ioutil.ReadFile(modaliasPath)
	return strings.TrimSpace(string(content))
}

func GetTempSensors(chip gosensors.Chip) []*sensors.HwmonSensor {
	var sensorList []*sensors.HwmonSensor

	features := chip.GetFeatures()
	for j := 0; j < len(features); j++ {
		feature := features[j]

		if feature.Type != gosensors.FeatureTypeTemp {
			continue
		}

		subfeatures := feature.GetSubFeatures()

		if containsSubFeature(subfeatures, gosensors.SubFeatureTypeTempInput) {
			inputSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeTempInput)
			sensorInputPath := fmt.Sprintf("%s/%s", chip.Path, inputSubFeature.Name)

			max := -1
			if containsSubFeature(subfeatures, gosensors.SubFeatureTypeTempMax) {
				maxSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeTempMax)
				max = int(maxSubFeature.GetValue())
			}

			min := -1
			if containsSubFeature(subfeatures, gosensors.SubFeatureTypeTempMin) {
				minSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeTempMin)
				min = int(minSubFeature.GetValue())
			}

			label := getLabel(chip.Path, inputSubFeature.Name)

			sensorList = append(
				sensorList,
				&sensors.HwmonSensor{
					Label:     label,
					Index:     len(sensorList) + 1,
					Input:     sensorInputPath,
					Max:       max,
					Min:       min,
					MovingAvg: inputSubFeature.GetValue(),
				})
		}
	}

	return sensorList
}

func GetFans(chip gosensors.Chip) []*fans.HwMonFan {
	var fanList []*fans.HwMonFan

	features := chip.GetFeatures()
	for j := 0; j < len(features); j++ {
		feature := features[j]

		if feature.Type != gosensors.FeatureTypeFan {
			continue
		}

		subfeatures := feature.GetSubFeatures()

		if containsSubFeature(subfeatures, gosensors.SubFeatureTypeFanInput) {
			inputSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeFanInput)
			rpmInput := fmt.Sprintf("%s/%s", chip.Path, inputSubFeature.Name)

			max := -1
			if containsSubFeature(subfeatures, gosensors.SubFeatureTypeFanMax) {
				maxSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeFanMax)
				max = int(maxSubFeature.GetValue())
			} else {
				max = fans.MaxPwmValue
			}

			min := -1
			if containsSubFeature(subfeatures, gosensors.SubFeatureTypeFanMin) {
				minSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeFanMin)
				min = int(minSubFeature.GetValue())
			} else {
				min = fans.MinPwmValue
			}

			pwmOutput := ""
			if containsSubFeature(subfeatures, gosensors.SubFeatureTypeFanPulses) {
				pulsesSubFeature := getSubFeature(subfeatures, gosensors.SubFeatureTypeFanPulses)
				pwmOutput = fmt.Sprintf("%s/%s", chip.Path, pulsesSubFeature.Name)
				folder, _ := filepath.Split(pwmOutput)
				pwmOutput = fmt.Sprintf("%spwm%d", folder, len(fanList)+1)
			}

			if len(pwmOutput) <= 0 {
				continue
			}

			label := getLabel(chip.Path, inputSubFeature.Name)

			fan := &fans.HwMonFan{
				Label:        label,
				Index:        len(fanList) + 1,
				PwmOutput:    pwmOutput,
				RpmInput:     rpmInput,
				RpmMovingAvg: inputSubFeature.GetValue(),
				MinPwm:       min,
				MaxPwm:       max,
			}

			fanList = append(fanList, fan)
		}
	}

	return fanList
}

func getSubFeature(subfeatures []gosensors.SubFeature, input gosensors.SubFeatureType) gosensors.SubFeature {
	for _, a := range subfeatures {
		if a.Type == input {
			return a
		}
	}
	panic(errors.New(fmt.Sprintf("No such element: %v", input)))
}

func containsSubFeature(s []gosensors.SubFeature, e gosensors.SubFeatureType) bool {
	for _, a := range s {
		if a.Type == e {
			return true
		}
	}
	return false
}

// getLabel read the label of a in/output of a device
func getLabel(devicePath string, input string) string {
	labelPath := strings.TrimSuffix(devicePath+"/"+input, "input") + "label"

	content, _ := ioutil.ReadFile(labelPath)
	label := string(content)
	if len(label) <= 0 {
		_, label = filepath.Split(devicePath)
	}
	return strings.TrimSpace(label)
}

func computeIdentifier(chip gosensors.Chip) (name string) {
	name = chip.Prefix

	devicePath := chip.Path
	if len(name) <= 0 {
		name = getDeviceName(devicePath)
	}

	if len(name) <= 0 {
		_, name = filepath.Split(devicePath)
	}

	identifier := name
	switch chip.Bus.Type {
	case BusTypeIsa:
		identifier = fmt.Sprintf("%s-isa-%d", identifier, chip.Bus.Nr)
	case BusTypePci:
		identifier = fmt.Sprintf("%s-pci-%d%x", identifier, chip.Bus.Nr, chip.Addr)
	case BusTypeVirtual:
		identifier = fmt.Sprintf("%s-virtual-%d", identifier, chip.Bus.Nr)
	case BusTypeAcpi:
		identifier = fmt.Sprintf("%s-acpi-%d", identifier, chip.Bus.Nr)
	case BusTypeHid:
		identifier = fmt.Sprintf("%s-hid-%d-%d", identifier, chip.Bus.Nr, chip.Addr)
	}

	return identifier
}

func findPlatform(devicePath string) string {
	platformRegex := regexp.MustCompile(".*/platform/{}/.*")
	return platformRegex.FindString(devicePath)
}
