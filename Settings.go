package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
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
	ShowOnCustomer         bool
}

type actionType uint8

const (
	ElConductivityHigh     = iota + 1 // Any time conductivity high
	ElPowerAndConductivity            // On first electrolyser powered if conductivity is high
	ElStartAndConductivity            // On first electrolyser started if conductivity is high
	ElPowered                         // Always on first electrolyser powered
	ElStart                           // Always on first electrolyser started
	ELRun                             // Any time an electrolyser is powered up or conductivity is high
)

type PortNameType struct {
	Name string
	Port uint8
}

type InputNameType struct {
	Name           string
	Port           uint8
	ShowOnCustomer bool
}

type ButtonType struct {
	Name           string
	Pressed        bool
	ShowOnCustomer bool
}

type ModbusNameType struct {
	Name    string
	SlaveID uint8
}

type FuelCellSettingsType struct {
	HighBatterySetpoint float64 `json:"HighBatterySetpoint"` // Default high battery setpoint
	LowBatterySetpoint  float64 `json:"LowBatterySetpoint"`  // Default low battery setpoint
	PowerSetting        float64 `json:"PowerSetting"`        // Default power level
	IgnoreIsoLow        bool    `json:"IgnoreIsoLow"`        // Flag to control IsoLow fault behaviour. True = suppress fault
	Enabled             bool    `json:"Enabled"`             // Allow us to control the fuel cell
	Capacity            uint16  `json:"Capacity"`            // Capacity in kW
	StartSOC            uint16  `json:"StartSOC"`            // Battery SOC at which the fuel cell will start
	StopSOC             uint16  `json:"StopSOC"`             // Battery SOC at which the fuel cell will stop
	MaxRunTime          uint16  `json:"MaxRunTime"`          // Maximum number of minutes the fuel cell is allowed to run
	MaximumOutput       uint16  `json:"MaximumOutput"`       // Maximum power the fuel cell can deliver
	Efficiency          uint8   `json:"Efficiency"`          // Average efficiency
}

type ElectrolyserSettingType struct {
	Name       string `json:"name"`
	IP         string `json:"ip"`
	Serial     string `json:"serial"`
	PowerRelay uint8  `json:"relay"`
	//	HasDryer   bool   `json:"dryer"`
	Enabled   bool   `json:"enabled"`
	StackTime uint32 `json:"stackTime"`
}

type UrlLinkType struct {
	Title                string `json:"name"`
	External             string `json:"external"`
	Internal             string `json:"internal"`
	ShowOnCustomerScreen bool   `json:"showOnCustomerScreen"`
}

func (lk *UrlLinkType) buildLink(external bool) string {
	if external {
		return fmt.Sprintf(`<li><a href="%s" class="urlLink" target="_blank">%s</a></li>`, lk.External, lk.Title)
	} else {
		return fmt.Sprintf(`<li><a href="%s" class="urlLink" target="_blank">%s</a></li>`, lk.Internal, lk.Title)
	}
}

type SettingsType struct {
	Name                             string                    `json:"Name"`
	FuelCell                         bool                      `json:"FuelCell"`
	AnalogChannels                   [8]AnalogSettingType      `json:"AnalogChannels"`
	DigitalInputs                    [4]InputNameType          `json:"DigitalInputs"`
	DigitalOutputs                   [6]PortNameType           `json:"DigitalOutputs"`
	Relays                           [16]PortNameType          `json:"Relays"`
	Buttons                          [20]ButtonType            `json:"Buttons"`
	FuelCellSettings                 FuelCellSettingsType      `json:"FuelCellSettings"`
	ACMeasurement                    [4]ModbusNameType         `json:"ACMeasurement"`
	DCMeasurement                    [4]ModbusNameType         `json:"DCMeasurement"`
	Subnet                           string                    `json:"subnet"`
	ElectrolyserMaxStackVoltsTurnOff int                       `json:"electrolyserMaxStackVoltsForShutdown"`
	ElectrolyserStopToStartTime      int                       `json:"electrolyserStopToStartTime"`
	ElectrolyserStartToStopTime      int                       `json:"electrolyserStartToStopTime"`
	DryerRelay                       int                       `json:"dryerRelay"`
	Electrolysers                    []ElectrolyserSettingType `json:"electrolysers"`
	APIKey                           string                    `json:"apiKey"`
	WaterDumpRelay                   uint8                     `json:"water"`
	WaterDumpSeconds                 uint8                     `json:"waterSeconds"`
	MaximumConductivity              float64                   `json:"maxConductivity"`
	WaterQualityAlarm                float32                   `json:"waterQualityAlarm"`
	WaterDumpAction                  actionType                `json:"waterDumpAction"`
	SessionKey                       string                    `json:"sessionKey"`
	MaxGasPressure                   uint16                    `json:"maxGasPressure"`
	GasCapacity                      uint32                    `json:"gasCapacity"`
	GasVolumeUnits                   string                    `json:"gasVolumeUnits"`
	GasPressureUnits                 string                    `json:"gasPressureUnits"`
	GasLevelType                     string                    `json:"gasLevelType"`
	GasPressureInput                 uint8                     `json:"gasPressureInput"`
	GasDetectorThreshold             uint16                    `json:"gasDetectorThreshold"`
	GasDetectorInput                 uint8                     `json:"gasDetectorInput"`
	ConductivityGreenMax             float32                   `json:"conductivityGreenMax"`
	ConductivityYellowMax            float32                   `json:"conductivityYellowMax"`
	CoolingPumpRelay                 uint8                     `json:"coolingPumpRelay"`
	CoolingPumpStartTemperature      uint8                     `json:"coolingPumpStartTemperature"`
	CoolingPumpStopTemperature       uint8                     `json:"coolingPumpStopTemperature"`
	BoardVersion                     string                    `json:"boardVersion"`
	boardVersion                     uint16
	Links                            []UrlLinkType `json:"links"`
	filepath                         string
	acquiringElectrolysers           bool
	scanningElectrolysers            bool
	rescanElectrolysers              bool
}

func NewSettings() *SettingsType {
	settings := new(SettingsType)
	settings.Name = "FireflyService"
	settings.Subnet = "192.168.88.0"
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
		settings.DigitalInputs[idx].Name = fmt.Sprintf("Input-%d", idx)
	}

	for idx := range settings.DigitalOutputs {
		settings.DigitalOutputs[idx].Port = uint8(idx)
		settings.DigitalOutputs[idx].Name = fmt.Sprintf("Output-%d", idx)
	}
	for idx := range settings.Buttons {
		settings.Buttons[idx].Name = fmt.Sprintf("Button-%d", idx)
		settings.Buttons[idx].Pressed = false
	}
	for idx := range settings.Relays {
		settings.Relays[idx].Port = uint8(idx)
		settings.Relays[idx].Name = fmt.Sprintf("Relay-%d", idx)
	}
	settings.FuelCellSettings.IgnoreIsoLow = false
	settings.FuelCellSettings.Enabled = false
	settings.FuelCellSettings.Capacity = 20
	settings.FuelCellSettings.MaximumOutput = 10
	settings.FuelCellSettings.StartSOC = 50
	settings.FuelCellSettings.StopSOC = 70
	settings.FuelCellSettings.MaxRunTime = 90
	settings.FuelCellSettings.Efficiency = 42

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
	settings.ElectrolyserStopToStartTime = 600
	settings.ElectrolyserStartToStopTime = 600
	settings.WaterDumpRelay = 0
	settings.WaterDumpSeconds = 10
	settings.MaximumConductivity = 2.5
	settings.GasPressureUnits = "bar"
	settings.GasVolumeUnits = "litres"
	settings.GasLevelType = "volume"
	settings.MaxGasPressure = 35
	settings.GasCapacity = 74724 // 8 standard tanks @ 35Bar
	settings.GasPressureInput = 0
	settings.ConductivityYellowMax = 7.5
	settings.ConductivityGreenMax = 3.5
	settings.WaterQualityAlarm = 9.0
	settings.CoolingPumpRelay = 255
	settings.CoolingPumpStartTemperature = 42
	settings.CoolingPumpStopTemperature = 38
	return settings
}

// findButtonByName returns the button given the name
func (settings *SettingsType) findButtonByName(name string) (idx int, pressed bool) {
	for idx, btn := range settings.Buttons {
		if strings.EqualFold(btn.Name, name) {
			return idx, btn.Pressed
		}
	}
	return -1, false
}

// setButtonByName sets the button to the given value
func (settings *SettingsType) setButtonByName(name string, pressed bool) error {
	for idx, btn := range settings.Buttons {
		if strings.EqualFold(btn.Name, name) {
			settings.Buttons[idx].Pressed = pressed
			return nil
		}
	}
	return fmt.Errorf("button %s not found", name)
}

// findExistingElByRelay returns a pointer to the matching electrolyser from the given array or null if not found
func (settings *SettingsType) findElByRelay(relay uint8) *ElectrolyserSettingType {
	for el := range settings.Electrolysers {
		if settings.Electrolysers[el].PowerRelay == relay {
			return &settings.Electrolysers[el]
		}
	}
	return nil
}

// findExistingElByName returns a pointer to the matching electrolyser from the given array or null if not found
func (settings *SettingsType) findElByName(name string) *ElectrolyserSettingType {
	for el := range settings.Electrolysers {
		if strings.ToLower(settings.Electrolysers[el].Name) == strings.ToLower(name) {
			return &settings.Electrolysers[el]
		}
	}
	return nil
}

// findExistingElByIP returns a pointer to the matching electrolyser from the given array or null if not found
func (settings *SettingsType) findElByIP(ip string) *ElectrolyserSettingType {
	for el := range settings.Electrolysers {
		if settings.Electrolysers[el].IP == ip {
			return &settings.Electrolysers[el]
		}
	}
	return nil
}

// addElectrolyser adds a new electrolyser to the settings object
// func (settings *SettingsType) addElectrolyser(Relay uint8, Name string, HasDryer bool, Enabled bool) {
func (settings *SettingsType) addElectrolyser(Relay uint8, Name string, Enabled bool) {
	var el ElectrolyserSettingType

	el.Name = Name
	el.PowerRelay = Relay
	//	el.HasDryer = HasDryer
	el.Enabled = Enabled
	settings.Electrolysers = append(settings.Electrolysers, el)
}

func (settings *SettingsType) LoadSettings(filepath string) error {
	if file, err := os.ReadFile(filepath); err != nil {
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
	// log.Println("Updating the electrolysers")
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

	// Sort the array by stack time
	// log.Printf("Electrolysers : %s=%d | %s=%d | %s = %d",
	//	settings.Electrolysers[0].Name, settings.Electrolysers[0].StackTime,
	//	settings.Electrolysers[1].Name, settings.Electrolysers[1].StackTime,
	//	settings.Electrolysers[2].Name, settings.Electrolysers[2].StackTime)
	els := settings.Electrolysers[:]
	slices.SortStableFunc(els, func(a ElectrolyserSettingType, b ElectrolyserSettingType) int {
		return cmp.Compare(a.StackTime, b.StackTime)
	})
	//log.Printf("Electrolysers after sort : %s=%d | %s=%d | %s = %d",
	//	settings.Electrolysers[0].Name, settings.Electrolysers[0].StackTime,
	//	settings.Electrolysers[1].Name, settings.Electrolysers[1].StackTime,
	//	settings.Electrolysers[2].Name, settings.Electrolysers[2].StackTime)

	for _, el := range settings.Electrolysers {
		if el.Name != "" {
			//			log.Printf("finding %s", el.Name)
			if elect := Electrolysers.FindByRelay(el.PowerRelay); elect != nil {
				// If we have an ip address, try and assign it.
				//				log.Printf("%s ip = %s", el.Name, el.IP)
				if el.IP != "" {
					if err := elect.status.IP.UnmarshalText([]byte(el.IP)); err != nil {
						// We failed to parse the ip address provided from the settings object
						log.Println(err)
					}
				}
				elect.status.Name = el.Name
				elect.status.PowerRelay = el.PowerRelay
				elect.status.Enabled = el.Enabled
				elect.enabled = el.Enabled
				elect.status.StackTotalRunTime = el.StackTime
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
				//				newEl.hasDryer = el.HasDryer
				newEl.status.Enabled = el.Enabled
				newEl.status.StackTotalRunTime = el.StackTime
				newEl.enabled = el.Enabled
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
		log.Println("New electrolyser array assigned")
	}
	//log.Printf("Actual electrolysers :  %s=%d | %s=%d | %s = %d",
	//	Electrolysers.Arr[0].status.Name, Electrolysers.Arr[0].status.StackTotalRunTime,
	//	Electrolysers.Arr[1].status.Name, Electrolysers.Arr[1].status.StackTotalRunTime,
	//	Electrolysers.Arr[2].status.Name, Electrolysers.Arr[2].status.StackTotalRunTime)
}

func (settings *SettingsType) ListBackupFiles(name string) ([]string, error) {
	root := name[:strings.LastIndex(name, "/")]
	var files []string
	f, err := os.Open(root)
	if err != nil {
		return files, err
	}
	fileInfo, err := f.Readdir(-1)
	if closeErr := f.Close(); closeErr != nil {
		log.Println(closeErr)
	}
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		name := file.Name()
		if strings.Contains(name, ".backup-") {
			files = append(files, name)
		}
	}
	sort.Strings(files)
	return files, nil
}

const FilesToKeep = 30

func (settings *SettingsType) Backup(filepath string) {
	backupFile := fmt.Sprintf("%s.backup-%s", filepath, time.Now().Format("2006-01-02"))
	if _, err := os.Stat(backupFile); err == nil {
		return
	}
	if renameErr := os.Rename(filepath, backupFile); renameErr != nil {
		log.Println(renameErr)
	}
	searchFile := fmt.Sprintf("%s.backup-*", filepath)
	if files, err := settings.ListBackupFiles(searchFile); err != nil {
		log.Printf("Error reading directory: %s", err)
	} else {
		if len(files) > FilesToKeep {
			for _, file := range files[:len(files)-FilesToKeep] {
				//				log.Printf("Removing backup file %s", file)
				fileToDelete := filepath[:strings.LastIndex(filepath, "/")+1] + file
				if removeErr := os.Remove(fileToDelete); removeErr != nil {
					log.Println(removeErr)
				}
			}
		}
	}
}

func (settings *SettingsType) SaveSettings(filepath string) error {
	settings.Backup(filepath)
	settings.filepath = filepath
	if bData, err := json.MarshalIndent(settings, "", "    "); err != nil {
		log.Println("Error converting settings to text -", err)
		return err
	} else {
		if err = os.WriteFile(settings.filepath, bData, 0644); err != nil {
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

func hasDuplicateSerial(arr []ElectrolyserSettingType) bool {
	visited := make(map[string]bool, 0)
	for i := 0; i < len(arr); i++ {
		if visited[arr[i].Serial] == true {
			return true
		} else {
			visited[arr[i].Serial] = true
		}
	}
	return false
}

func hasDuplicateIP(arr []ElectrolyserSettingType) bool {
	visited := make(map[string]bool, 0)
	for i := 0; i < len(arr); i++ {
		if visited[arr[i].IP] == true {
			return true
		} else {
			visited[arr[i].IP] = true
		}
	}
	return false
}

// validateSettings will adjust any values that are obviously out of range and bring them withing the limits defined
func (settings *SettingsType) validateSettings() {
	if settings.FuelCellSettings.LowBatterySetpoint < 35.0 ||
		settings.FuelCellSettings.LowBatterySetpoint > settings.FuelCellSettings.HighBatterySetpoint {
		settings.FuelCellSettings.LowBatterySetpoint = 42.0
	}
	if settings.FuelCellSettings.HighBatterySetpoint > 70.0 ||
		settings.FuelCellSettings.HighBatterySetpoint < settings.FuelCellSettings.LowBatterySetpoint {
		settings.FuelCellSettings.HighBatterySetpoint = 65.0
	}
	// Find duplicate serial numbers in eletrolysers
	if hasDuplicateSerial(settings.Electrolysers) {
		settings.rescanElectrolysers = true
	}
	if hasDuplicateIP(settings.Electrolysers) {
		settings.rescanElectrolysers = true
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
		settings.DigitalInputs[din].ShowOnCustomer = r.FormValue(fmt.Sprintf("di%duser", din)) != ""
	}
	// Digital output names
	for dOut := 0; dOut < 6; dOut++ {
		settings.DigitalOutputs[dOut].Name = r.FormValue(fmt.Sprintf("do%dname", dOut))
	}
	// Buttons
	for btn := range settings.Buttons {
		settings.Buttons[btn].Name = r.FormValue(fmt.Sprintf("btn%dname", btn))
		settings.Buttons[btn].ShowOnCustomer = r.FormValue(fmt.Sprintf("btn%duser", btn)) != ""
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
		settings.AnalogChannels[analog].ShowOnCustomer = r.FormValue(fmt.Sprintf("a%dShowOnCustomer", analog)) != ""
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
	if val, err := strconv.ParseInt(r.FormValue("GasCapacity"), 10, 32); err != nil {
		ReturnJSONError(w, DeviceString+"GasCapacity", err, http.StatusInternalServerError, true)
		return
	} else {
		settings.GasCapacity = uint32(val)
	}
	//	settings.GasUnits = r.FormValue("GasUnits")
	settings.GasPressureUnits = r.FormValue("GasPressureUnits")
	settings.GasVolumeUnits = r.FormValue("GasVolumeUnits")
	settings.GasLevelType = r.FormValue("GasLevelType")
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
		settings.FuelCellSettings.Capacity = uint16(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("fcStartSOC"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"FuelCellStartSOC", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.StartSOC = uint16(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("fcStopSOC"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"FuelCellStopSOC", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.StopSOC = uint16(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("fcMaxTime"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"FuelCellMaxTime", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.MaxRunTime = uint16(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("fcMaxOutput"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"FuelCellMaxOutput", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.MaximumOutput = uint16(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("fcEfficiency"), 10, 16); err != nil {
		ReturnJSONError(w, DeviceString+"FuelCellEfficiency", err, http.StatusBadRequest, true)
		return
	} else {
		settings.FuelCellSettings.Efficiency = uint8(val)
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
	if val, err := strconv.ParseInt(r.FormValue("electrolyserStopToStartTime"), 10, 16); err != nil {
		ReturnJSONError(w, "SaveSettings", err, http.StatusBadRequest, true)
		return
	} else {
		settings.ElectrolyserStopToStartTime = int(val)
	}

	if val, err := strconv.ParseInt(r.FormValue("electrolyserStartToStopTime"), 10, 16); err != nil {
		ReturnJSONError(w, "SaveSettings", err, http.StatusBadRequest, true)
		return
	} else {
		settings.ElectrolyserStartToStopTime = int(val)
	}
	log.Println("subnet = ", r.FormValue("subnet"))
	if val := r.FormValue("subnet"); val != "" {
		if ip := net.ParseIP(val); ip != nil {
			if ip[3] == 0 {
				settings.Subnet = val
			} else {
				log.Println("subnet should end in 0 - ", val)
			}
		} else {
			log.Println("bad subnet - ", val)
		}
	} else {
		log.Println("no subnet in settings")
	}

	if val, err := strconv.ParseInt(r.FormValue("dryerRelay"), 10, 16); err != nil {
		ReturnJSONError(w, "SaveSettings", err, http.StatusBadRequest, true)
		return
	} else {
		settings.DryerRelay = int(val)
	}
	if val, err := strconv.ParseInt(r.FormValue("waterDumpSeconds"), 10, 8); err != nil {
		ReturnJSONError(w, DeviceString+":Water Dump Seconds", err, http.StatusBadRequest, true)
		return
	} else {
		settings.WaterDumpSeconds = uint8(val)
	}
	if val, err := strconv.ParseUint(r.FormValue("waterDumpRelay"), 10, 8); err != nil {
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
		if debugOutput {
			log.Println("Water dump action = ", val)
		}
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
		settings.FuelCellSettings.Capacity = uint16(val)
	}
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
					//					el.HasDryer = electrolyser.HasDryer
					el.IP = ""
					el.Serial = ""
					el.Enabled = electrolyser.Enabled
				}
			} else {
				el.Enabled = electrolyser.Enabled
			}
		} else {
			// We did not find a match, so we should try by name
			if el := settings.findElByRelay(electrolyser.Relay); el != nil {
				// We found an electrolyser with a different name on the same relay.
				if Relays.Relays[el.PowerRelay].On {
					ReturnJSONErrorString(w, "SaveSettings", "You cannot change the settings name of an active electrolyser", http.StatusBadRequest, true)
					return
				} else {
					el.PowerRelay = electrolyser.Relay
					//					el.HasDryer = electrolyser.HasDryer
					el.Serial = ""
					el.IP = ""
					el.Enabled = electrolyser.Enabled
				}
			} else {
				// Cannot find a match, so we should add this one in
				//				settings.addElectrolyser(electrolyser.Relay, electrolyser.Name, electrolyser.HasDryer, electrolyser.Enabled)
				settings.addElectrolyser(electrolyser.Relay, electrolyser.Name, electrolyser.Enabled)
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

	LinkSettings := r.FormValue("Links")
	//	log.Println(LinkSettings)
	newLinks := make([]UrlLinkType, 0)
	if err := json.Unmarshal([]byte(LinkSettings), &newLinks); err != nil {
		ReturnJSONError(w, DeviceString+":Links", err, http.StatusBadRequest, true)
		return
	}
	settings.Links = newLinks

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

func (settings *SettingsType) buildLinks(external bool, admin bool) string {
	linkSet := make([]string, 0)

	for _, link := range settings.Links {
		if admin || link.ShowOnCustomerScreen {
			linkSet = append(linkSet, link.buildLink(external))
		}
	}
	return strings.Join(linkSet, "")
}

func (settings *SettingsType) hasACDevices() bool {
	for _, device := range settings.ACMeasurement {
		if device.Name != "" {
			return true
		}
	}
	return false
}

func (settings *SettingsType) hasDCDevices() bool {
	for _, device := range settings.DCMeasurement {
		if device.Name != "" {
			return true
		}
	}
	return false
}

func (settings *SettingsType) hasFuelCell() bool {
	return settings.FuelCell
}
