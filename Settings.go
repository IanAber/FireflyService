package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type AnalogSettingType struct {
	Name                   string
	Port                   uint8
	LowerCalibrationActual float32
	LowerCalibrationAtoD   uint16
	UpperCalibrationActual float32
	UpperCalibrationAtoD   uint16
	calibrationConstant    float32
	calibrationMultiplier  float32
	MaxVal                 float32
	MinVal                 float32
}

type actionType uint8

const (
	CONDUCTIVITYHIGH = iota + 1
	ELPOWERANDCONDUCTIVITY
	ELSTARTANDCONDUCTIVITY
	ELPOWERED
	ELSTART
)

type PortNameType struct {
	Name string
	Port uint8
}

type ModbusNameType struct {
	Name    string
	SlaveID uint8
}

type FuelCellSettingsType struct {
	HighBatterySetpoint float64 // Default high battery setpoint
	LowBatterySetpoint  float64 // Default low battery setpoint
	PowerSetting        float64 // Default power level
	IgnoreIsoLow        bool    // Flag to control IsoLow fault behaviour. True = suppress fault
	Enabled             bool    // Allow us to control the fuel cell
	Capacity            int16   // Capacity in kW
}

type ElectrolyserSettingType struct {
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Serial     string `json:"serial"`
	PowerRelay uint8  `json:"relay"`
	HasDryer   bool   `json:"dryer"`
}

type SettingsType struct {
	Name                             string                    `json:"Name"`
	FuelCell                         bool                      `json:"FuelCell"`
	AnalogChannels                   [8]AnalogSettingType      `json:"AnalogChannels"`
	DigitalInputs                    [4]PortNameType           `json:"DigitalInputs"`
	DigitalOutputs                   [6]PortNameType           `json:"DigitalOutputs"`
	Relays                           [16]PortNameType          `json:"Relays"`
	FuelCellSettings                 FuelCellSettingsType      `json:"FuelCellSettings"`
	ACMeasurement                    [4]ModbusNameType         `json:"ACMeasurement"`
	DCMeasurement                    [4]ModbusNameType         `json:"DCMeasurement"`
	ElectrolyserMaxStackVoltsTurnOff int                       `json:"electrolyserMaxStackVoltsForShutdown"`
	Electrolysers                    []ElectrolyserSettingType `json:"electrolysers"`
	NodeRED                          string                    `json:"nodeRED"`
	APIKey                           string                    `json:"apiKey"`
	WaterDumpRelay                   uint8                     `json:"water"`
	WaterDumpSeconds                 uint8                     `json:"waterSeconds"`
	MaximumConductivity              float64                   `json:"maxConductivity"`
	WaterQualityAlarm                float32                   `json:"waterQualityAlarm"`
	WaterDumpAction                  actionType                `json:"waterDumpAction"`
	SessionKey                       string                    `json:"sessionKey"`
	MaxGasPressure                   uint16                    `json:"maxGasPressure"`
	GasUnits                         string                    `json:"gasUnits"`
	GasPressureInput                 uint8                     `json:"gasPressureInput"`
	GasDetectorThreshold             uint16                    `json:"gasDetectorThreshold"`
	GasDetectorInput                 uint8                     `json:"gasDetectorInput"`
	ConductivityGreenMax             float32                   `json:"conductivityGreenMax"`
	ConductivityYellowMax            float32                   `json:"conductivityYellowMax"`
	CoolingPumpRelay                 uint8                     `json:"coolingPumpRelay"`
	CoolingPumpStartTemperature      uint8                     `json:"coolingPumpStartTemperature"`
	CoolingPumpStopTemperature       uint8                     `json:"coolingPumpStopTemperature"`
	filepath                         string
}

func NewSettings() *SettingsType {
	settings := new(SettingsType)
	settings.Name = "FireflyService"
	for idx := range settings.AnalogChannels {
		settings.AnalogChannels[idx].Port = uint8(idx)
		settings.AnalogChannels[idx].Name = fmt.Sprintf("Analog-%d", idx)
		settings.AnalogChannels[idx].UpperCalibrationActual = 1024
		settings.AnalogChannels[idx].UpperCalibrationAtoD = 1024
		settings.AnalogChannels[idx].LowerCalibrationActual = 0
		settings.AnalogChannels[idx].LowerCalibrationAtoD = 0
		settings.AnalogChannels[idx].calculateConstants()
		settings.AnalogChannels[idx].MaxVal = 1024
		settings.AnalogChannels[idx].MinVal = 0
	}
	for idx := range settings.DigitalInputs {
		settings.DigitalInputs[idx].Port = uint8(idx)
		settings.DigitalInputs[idx].Name = fmt.Sprintf("Intput-%d", idx)
	}

	for idx := range settings.DigitalOutputs {
		settings.DigitalOutputs[idx].Port = uint8(idx)
		settings.DigitalOutputs[idx].Name = fmt.Sprintf("Output-%d", idx)
	}

	for idx := range settings.Relays {
		settings.Relays[idx].Port = uint8(idx)
		settings.Relays[idx].Name = fmt.Sprintf("Relay-%d", idx)
	}
	settings.FuelCellSettings.IgnoreIsoLow = false
	settings.FuelCellSettings.Enabled = false
	settings.FuelCellSettings.Capacity = 20

	for i := range settings.ACMeasurement {
		settings.ACMeasurement[i].Name = ""
		settings.ACMeasurement[i].SlaveID = 0x20 + uint8(i)
	}
	for i := range settings.DCMeasurement {
		settings.DCMeasurement[i].Name = ""
		settings.DCMeasurement[i].SlaveID = 0x10 + uint8(i)
	}
	// Default to just one AC measurement device and no DC measurement devices.
	settings.ACMeasurement[0].Name = "Firefly"
	settings.ElectrolyserMaxStackVoltsTurnOff = 30
	settings.NodeRED = ""
	settings.WaterDumpRelay = 0
	settings.WaterDumpSeconds = 10
	settings.MaximumConductivity = 2.5
	settings.GasUnits = "Bar"
	settings.MaxGasPressure = 35
	settings.GasPressureInput = 0
	settings.ConductivityYellowMax = 7.5
	settings.ConductivityGreenMax = 3.5
	settings.WaterQualityAlarm = 9.0
	settings.CoolingPumpRelay = 255
	settings.CoolingPumpStartTemperature = 42
	settings.CoolingPumpStopTemperature = 38
	return settings
}

/**
findExistingElByRelay returns a pointer to the matching electrolyser from the given array or null if not found
*/
func (settings *SettingsType) findElByRelay(relay uint8) *ElectrolyserSettingType {
	for el := range settings.Electrolysers {
		if settings.Electrolysers[el].PowerRelay == relay {
			return &settings.Electrolysers[el]
		}
	}
	return nil
}

/**
findExistingElByName returns a pointer to the matching electrolyser from the given array or null if not found
*/
func (settings *SettingsType) findElByName(name string) *ElectrolyserSettingType {
	for el := range settings.Electrolysers {
		if strings.ToLower(settings.Electrolysers[el].Name) == strings.ToLower(name) {
			return &settings.Electrolysers[el]
		}
	}
	return nil
}

/**
findExistingElByIP returns a pointer to the matching electrolyser from the given array or null if not found
*/
func (settings *SettingsType) findElByIP(ip string) *ElectrolyserSettingType {
	for el := range settings.Electrolysers {
		if settings.Electrolysers[el].IP == ip {
			return &settings.Electrolysers[el]
		}
	}
	return nil
}

/**
addElectrolyser adds a new electrolyser to the settings object
*/
func (settings *SettingsType) addElectrolyser(Relay uint8, Name string, HasDryer bool) {
	var el ElectrolyserSettingType

	el.Name = Name
	el.PowerRelay = Relay
	el.HasDryer = HasDryer

	settings.Electrolysers = append(settings.Electrolysers, el)
}

func (settings *SettingsType) LoadSettings(filepath string) error {
	if file, err := ioutil.ReadFile(filepath); err != nil {
		log.Println(err)
		if err := settings.SaveSettings(filepath); err != nil {
			return err
		}
	} else {
		settings.filepath = filepath
		if err := json.Unmarshal(file, settings); err != nil {
			return err
		}
		settings.validateSettings()
	}
	settings.filepath = filepath
	settings.calculateConstants()
	for _, rl := range settings.Relays {
		Relays.Relays[rl.Port].Name = rl.Name
	}
	for _, op := range settings.DigitalOutputs {
		Outputs.Outputs[op.Port].Name = op.Name
	}
	for _, ip := range settings.DigitalInputs {
		Inputs.Inputs[ip.Port].Name = ip.Name
	}
	for _, analog := range settings.AnalogChannels {
		AnalogInputs.Inputs[analog.Port].Name = analog.Name
	}
	for i, ac := range settings.ACMeasurement {
		ACMeasurements[i].Name = ac.Name
	}
	for i, dc := range settings.DCMeasurement {
		DCMeasurements[i].Name = dc.Name
	}
	settings.UpdateElectrolyserArray()

	return nil
}

func (settings *SettingsType) FindElectrolyserByRelay(relay uint8) *ElectrolyserSettingType {
	for idx, el := range settings.Electrolysers {
		if el.PowerRelay == relay {
			return &settings.Electrolysers[idx]
		}
	}
	return nil
}

func (settings *SettingsType) UpdateElectrolyserArray() {
	// Update the electrolysers or add new ones as necessary
	Electrolysers.mu.Lock()
	defer Electrolysers.mu.Unlock()
	for _, el := range settings.Electrolysers {
		if el.Name != "" {
			if elect := Electrolysers.FindByRelay(el.PowerRelay); elect != nil {
				// If we have an ip address, try and assign it.
				if el.IP != "" {
					if err := elect.status.IP.UnmarshalText([]byte(el.IP)); err != nil {
						// We failed to parse the ip address provided from the settings object
						log.Println(err)
					}
				}
				elect.status.Name = el.Name
				elect.status.PowerRelay = el.PowerRelay
			} else {
				// We did not find an electrolyser on that relay, so we should add a new one
				IP := net.IPv4zero
				if el.IP != "" {
					err := IP.UnmarshalText([]byte(el.IP))
					if err != nil {
						log.Println(err)
					}
				}
				newEl := NewElectrolyser(IP)
				newEl.powerRelay = el.PowerRelay
				newEl.status.Name = el.Name
				newEl.status.PowerRelay = el.PowerRelay
				newEl.hasDryer = el.HasDryer
				Electrolysers.Arr = append(Electrolysers.Arr, newEl)
			}
		}
	}
	// Copy all the electrolysers that match one in the settings by relay
	newArr := make([]*ElectrolyserType, 0)
	for _, el := range Electrolysers.Arr {
		if settings.FindElectrolyserByRelay(el.powerRelay) != nil {
			newArr = append(newArr, el)
		}
	}
	// If the length is different, then replace the array to get rid of ones we no longer have
	if len(newArr) != len(Electrolysers.Arr) {
		Electrolysers.Arr = newArr
	}
}

func (settings *SettingsType) SaveSettings(filepath string) error {
	settings.filepath = filepath
	if bData, err := json.Marshal(settings); err != nil {
		log.Println("Error converting settings to text -", err)
		return err
	} else {
		if err = ioutil.WriteFile(settings.filepath, bData, 0644); err != nil {
			log.Println("Error writing JSON settings file -", err)
			return err
		}
	}
	return nil
}

func (AnalogSetting *AnalogSettingType) calculateConstants() {
	AnalogSetting.calibrationMultiplier = (AnalogSetting.UpperCalibrationActual - AnalogSetting.LowerCalibrationActual) / float32(AnalogSetting.UpperCalibrationAtoD-AnalogSetting.LowerCalibrationAtoD)
	AnalogSetting.calibrationConstant = AnalogSetting.LowerCalibrationActual - (float32(AnalogSetting.LowerCalibrationAtoD) * AnalogSetting.calibrationMultiplier)
}

func (settings *SettingsType) calculateConstants() {
	for idx := range settings.AnalogChannels {
		settings.AnalogChannels[idx].calculateConstants()
	}
}

func (settings *SettingsType) SendSettingsJSON(w http.ResponseWriter) {
	if bData, err := json.Marshal(settings); err != nil {
		log.Println("Error converting settings to text -", err)
	} else {
		if _, err := fmt.Fprint(w, string(bData)); err != nil {
			log.Print(err)
		}
	}
}

/**
validateSettings will adjust any values that are obviously out of range and bring them withing the limits defined
*/
func (settings *SettingsType) validateSettings() {
	if settings.FuelCellSettings.LowBatterySetpoint < 35.0 ||
		settings.FuelCellSettings.LowBatterySetpoint > settings.FuelCellSettings.HighBatterySetpoint {
		settings.FuelCellSettings.LowBatterySetpoint = 42.0
	}
	if settings.FuelCellSettings.HighBatterySetpoint > 70.0 ||
		settings.FuelCellSettings.HighBatterySetpoint < settings.FuelCellSettings.LowBatterySetpoint {
		settings.FuelCellSettings.HighBatterySetpoint = 65.0
	}
}

func (settings *SettingsType) LoadFromJSON(jsonData []byte) error {
	if err := json.Unmarshal(jsonData, settings); err != nil {
		return err
	} else {
		settings.validateSettings()
		return nil
	}
}

func (settings *SettingsType) getModbusFlags() (flags uint8) {
	flags = 0
	if len(settings.ACMeasurement[0].Name) > 0 {
		flags |= 0b00000001
	}
	if len(settings.ACMeasurement[1].Name) > 0 {
		flags |= 0b00000010
	}
	if len(settings.ACMeasurement[2].Name) > 0 {
		flags |= 0b00000100
	}
	if len(settings.ACMeasurement[3].Name) > 0 {
		flags |= 0b00001000
	}
	if len(settings.DCMeasurement[0].Name) > 0 {
		flags |= 0b00010000
	}
	if len(settings.DCMeasurement[1].Name) > 0 {
		flags |= 0b00100000
	}
	if len(settings.DCMeasurement[2].Name) > 0 {
		flags |= 0b01000000
	}
	if len(settings.DCMeasurement[3].Name) > 0 {
		flags |= 0b10000000
	}
	return
}

func (settings *SettingsType) IsFuelCellEnabled() bool {
	if settings.FuelCell && settings.FuelCellSettings.Enabled {
		return true
	} else {
		return false
	}
}

func (settings *SettingsType) setSettings(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "Save Settings"
	if err := r.ParseForm(); err != nil {
		ReturnJSONError(w, "setSettings", err, http.StatusBadRequest, true)
		return
	}
	// System name
	settings.Name = r.FormValue("name")
	// Relay names
	for relay := 0; relay < 16; relay++ {
		settings.Relays[relay].Name = r.FormValue(fmt.Sprintf("relay%dname", relay))
	}
	// Digital Input names
	for din := 0; din < 4; din++ {
		settings.DigitalInputs[din].Name = r.FormValue(fmt.Sprintf("di%dname", din))
	}
	// Digital output names
	for dout := 0; dout < 4; dout++ {
		settings.DigitalOutputs[dout].Name = r.FormValue(fmt.Sprintf("do%dname", dout))
	}
	// Analogue names and settings
	for analog := range settings.AnalogChannels {
		settings.AnalogChannels[analog].Name = r.FormValue(fmt.Sprintf("a%dname", analog))
		if f, err := strconv.ParseFloat(r.FormValue(fmt.Sprintf("a%dLowVal", analog)), 32); err != nil {
			ReturnJSONError(w, DeviceString+"Analog Low Value", err, http.StatusInternalServerError, true)
			return
		} else {
			settings.AnalogChannels[analog].LowerCalibrationActual = float32(f)
		}
		if f, err := strconv.ParseFloat(r.FormValue(fmt.Sprintf("a%dHighVal", analog)), 32); err != nil {
			ReturnJSONError(w, DeviceString+"Analog High Value", err, http.StatusInternalServerError, true)
			return
		} else {
			settings.AnalogChannels[analog].UpperCalibrationActual = float32(f)
		}
		if f, err := strconv.ParseInt(r.FormValue(fmt.Sprintf("a%dLowA2D", analog)), 10, 32); err != nil {
			ReturnJSONError(w, DeviceString+"A-D Low Value", err, http.StatusInternalServerError, true)
			return
		} else {
			settings.AnalogChannels[analog].LowerCalibrationAtoD = uint16(f)
		}
		if f, err := strconv.ParseInt(r.FormValue(fmt.Sprintf("a%dHighA2D", analog)), 10, 32); err != nil {
			ReturnJSONError(w, DeviceString+"A-D High Value", err, http.StatusInternalServerError, true)
			return
		} else {
			settings.AnalogChannels[analog].UpperCalibrationAtoD = uint16(f)
		}
		if f, err := strconv.ParseFloat(r.FormValue(fmt.Sprintf("a%dMinVal", analog)), 32); err != nil {
			ReturnJSONError(w, DeviceString+"Analog Min Value", err, http.StatusInternalServerError, true)
			return
		} else {
			settings.AnalogChannels[analog].MinVal = float32(f)
		}
		if f, err := strconv.ParseFloat(r.FormValue(fmt.Sprintf("a%dMaxVal", analog)), 32); err != nil {
			ReturnJSONError(w, DeviceString+"Analog Max Value", err, http.StatusInternalServerError, true)
			return
		} else {
			settings.AnalogChannels[analog].MaxVal = float32(f)
		}
		settings.AnalogChannels[analog].calculateConstants()
	}
	if r.FormValue("isoLowBehaviour") == "true" {
		settings.FuelCellSettings.IgnoreIsoLow = true
	} else {
		settings.FuelCellSettings.IgnoreIsoLow = false
	}
	settings.APIKey = r.FormValue("APIKey")
	settings.FuelCell = len(r.FormValue("FuelCell")) > 0
	if val, err := strconv.ParseInt(r.FormValue("MaxGasPressure"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"MaxGasPressure", err, http.StatusInternalServerError, true)
		return
	} else {
		settings.MaxGasPressure = uint16(val)
	}
	settings.GasUnits = r.FormValue("GasUnits")
	if val, err := strconv.ParseInt(r.FormValue("GasInput"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+"GasInput", err, http.StatusInternalServerError, true)
		return
	} else {
		settings.GasPressureInput = uint8(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("GasDetectorThreshold"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"GasDetectorThreshold", err, http.StatusInternalServerError, true)
		return
	} else {
		settings.GasDetectorThreshold = uint16(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("GasDetectorInput"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+"GasDetectorInput", err, http.StatusInternalServerError, true)
		return
	} else {
		settings.GasDetectorInput = uint8(val)
	}

	if val, err := strconv.ParseInt(r.FormValue("fcCapacity"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"FuelCellCapacity", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.Capacity = int16(val)
	}
	settings.ACMeasurement[0].Name = strings.TrimSpace(r.FormValue("ACMeasurement20"))
	settings.ACMeasurement[0].SlaveID = 20
	settings.ACMeasurement[1].Name = strings.TrimSpace(r.FormValue("ACMeasurement21"))
	settings.ACMeasurement[1].SlaveID = 21
	settings.ACMeasurement[2].Name = strings.TrimSpace(r.FormValue("ACMeasurement22"))
	settings.ACMeasurement[2].SlaveID = 22
	settings.ACMeasurement[3].Name = strings.TrimSpace(r.FormValue("ACMeasurement23"))
	settings.ACMeasurement[3].SlaveID = 23
	settings.DCMeasurement[0].Name = strings.TrimSpace(r.FormValue("DCMeasurement10"))
	settings.DCMeasurement[0].SlaveID = 10
	settings.DCMeasurement[1].Name = strings.TrimSpace(r.FormValue("DCMeasurement11"))
	settings.DCMeasurement[1].SlaveID = 11
	settings.DCMeasurement[2].Name = strings.TrimSpace(r.FormValue("DCMeasurement12"))
	settings.DCMeasurement[2].SlaveID = 12
	settings.DCMeasurement[3].Name = strings.TrimSpace(r.FormValue("DCMeasurement13"))
	settings.DCMeasurement[3].SlaveID = 13

	if val, err := strconv.ParseInt(r.FormValue("electrolyserMaxStackVoltsForShutdown"), 10, 16); err != nil {
		ReturnJSONError(w, "SaveSettings", err, http.StatusBadRequest, true)
		return
	} else {
		settings.ElectrolyserMaxStackVoltsTurnOff = int(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("waterDumpSeconds"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Water Dump Seconds", err, http.StatusBadRequest, true)
		return
	} else {
		settings.WaterDumpSeconds = uint8(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("waterDumpRelay"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Water Dump Relay", err, http.StatusBadRequest, true)
		return
	} else {
		settings.WaterDumpRelay = uint8(val)
	}
	if val, err := strconv.ParseFloat(r.FormValue("maxConductivity"), 64); err != nil {
		ReturnJSONError(w, DeviceString+":Maximum Conductivity", err, http.StatusBadRequest, true)
		return
	} else {
		settings.MaximumConductivity = val
	}
	if val, err := strconv.ParseFloat(r.FormValue("maxGreenConductivity"), 64); err != nil {
		ReturnJSONError(w, DeviceString+":Maximum Green Conductivity", err, http.StatusBadRequest, true)
		return
	} else {
		settings.ConductivityGreenMax = float32(val)
	}
	if val, err := strconv.ParseFloat(r.FormValue("maxYellowConductivity"), 64); err != nil {
		ReturnJSONError(w, DeviceString+":Maximum Yellow Conductivity", err, http.StatusBadRequest, true)
		return
	} else {
		settings.ConductivityYellowMax = float32(val)
	}
	if val, err := strconv.ParseFloat(r.FormValue("waterQualityAlarm"), 64); err != nil {
		ReturnJSONError(w, DeviceString+":Water Quality Alarm", err, http.StatusBadRequest, true)
		return
	} else {
		settings.WaterQualityAlarm = float32(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("waterDumpAction"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Water Dump Action", err, http.StatusBadRequest, true)
		return
	} else {
		log.Println("Water dump action = ", val)
		settings.WaterDumpAction = actionType(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("coolingPumpStartTemperature"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Cooling Pump Start Temperature", err, http.StatusBadRequest, true)
		return
	} else {
		settings.CoolingPumpStartTemperature = uint8(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("coolingPumpStopTemperature"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Cooling Pump Stop Temperature", err, http.StatusBadRequest, true)
		return
	} else {
		settings.CoolingPumpStopTemperature = uint8(val)
	}
	if val, err := strconv.ParseUint(r.FormValue("coolingPumpRelay"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Cooling Pump Relay", err, http.StatusBadRequest, true)
		return
	} else {
		settings.CoolingPumpRelay = uint8(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("fcCapacity"), 10, 16); err != nil {
		ReturnJSONError(w, "SaveSettings", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.Capacity = int16(val)
	}
	settings.NodeRED = r.FormValue("nodeRed")
	//	settings.Subnet = r.FormValue("subnet")
	ElectrolysersSettings := r.FormValue("ElectrolyserRelays")
	if debugOutput {
		log.Println("Settings = " + ElectrolysersSettings)
	}
	newElectrolysers := make([]elSettingType, 0)
	if err := json.Unmarshal([]byte(ElectrolysersSettings), &newElectrolysers); err != nil {
		ReturnJSONError(w, DeviceString+":Electrolyser Relay", err, http.StatusBadRequest, true)
		return
	}

	// Loop through the defined electrolyers
	// If we have an electrolyser on the same relay with a different name and the relay is off,
	// change the name and clear its IP address. This will force a rescan
	for _, electrolyser := range newElectrolysers {
		if el := settings.findElByRelay(electrolyser.Relay); el != nil {
			if el.Name != electrolyser.Name {
				if Relays.Relays[el.PowerRelay].On {
					ReturnJSONErrorString(w, "SaveSettings", "You cannot change the settings name of an active electrolyser", http.StatusBadRequest, true)
					return
				} else {
					// Change the name and clear the serial and IP address so this one will get rediscovered.
					el.Name = electrolyser.Name
					el.HasDryer = electrolyser.HasDryer
					el.IP = ""
					el.Serial = ""
				}
			}
		} else {
			// We did not find a match so we should try by name
			if el := settings.findElByRelay(electrolyser.Relay); el != nil {
				// We found an electrolyser with a different name on the same relay.
				if Relays.Relays[el.PowerRelay].On {
					ReturnJSONErrorString(w, "SaveSettings", "You cannot change the settings name of an active electrolyser", http.StatusBadRequest, true)
					return
				} else {
					el.PowerRelay = electrolyser.Relay
					el.HasDryer = electrolyser.HasDryer
					el.Serial = ""
					el.IP = ""
				}
			} else {
				// Cannot find a match so we should add this one in
				settings.addElectrolyser(electrolyser.Relay, electrolyser.Name, electrolyser.HasDryer)
			}
		}
	}

	// No lets get rid of any electrolysers we don't want any more
	for len(settings.Electrolysers) > len(newElectrolysers) {
		for idx, el := range settings.Electrolysers {
			if findNewElByRelay(newElectrolysers, el.PowerRelay) == -1 {
				if Relays.Relays[el.PowerRelay].On {
					ReturnJSONErrorString(w, "SaveSettings", "You cannot remove an active electrolyser", http.StatusBadRequest, true)
					return
				}
				settings.Electrolysers = append(settings.Electrolysers[:idx], settings.Electrolysers[idx+1:]...)
				break
			}
		}
	}

	if err := settings.SaveSettings(settings.filepath); err != nil {
		log.Print(err)
	}
	if err := settings.LoadSettings(settings.filepath); err != nil {
		log.Print(err)
	}

	doScan := false
	if r.FormValue("forceScan") != "" {
		doScan = true
	} else {
		for _, el := range Electrolysers.Arr {
			if el.status.IP.Equal(net.IPv4zero) {
				log.Printf("%s : %d\n", el.status.Name, el.status.IP[0])
				doScan = true
			}
		}
	}
	if doScan {
		if !acquireAllElectrolysers(w) {
			return
		}
	}
	http.Redirect(w, r, "/config.html", http.StatusTemporaryRedirect)
}
