package main

import (
	"bytes"
	"cmp"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/simonvetter/modbus"
	"html"
	"log"
	"math"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

//const ElIdle = 2
//const ElSteady = 3

const ElectrolyserInsertStatement = "INSERT INTO firefly.Electrolyser (name, flow, volts, amps, temperature, errors, warnings, waterPressure, rate, innerH2Pressure, outerH2Pressure, electrolyteLevel, electrolyteFlow, electronicFanSpeed, airFanSpeed, electrolyteFanSpeed, downstreamTemperature) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?);"
const DryerInsertStatement = "INSERT INTO firefly.Dryer (temp1, temp2, temp3, temp4, inputPressure, outputPressure, errors, warnings) VALUES (?,?,?,?,?,?,?,?);"

const ElectrolyserArchiveStatement = `INSERT INTO firefly.Electrolyser_Archive (logged, name, flow, volts, amps, temperature, errors, waterPressure, rate, innerH2Pressure, outerH2Pressure, electrolyteLevel, electrolyteFlow, electronicFanSpeed, airFanSpeed, electrolyteFanSpeed, downstreamTemperature)
SELECT min(logged), min(name), avg(flow), avg(volts), avg(amps), avg(temperature), min(errors), avg(waterPressure), avg(rate), avg(innerH2Pressure), avg(outerH2Pressure), min(electrolyteLevel), avg(electrolyteFlow), avg(electronicFanSpeed), avg(airFanSpeed), avg(electrolyteFanSpeed), avg(downstreamTemperature)
  FROM firefly.Electrolyser
 WHERE logged < DATE(DATE_ADD( now(), interval -1 month))
 GROUP BY UNIX_TIMESTAMP(logged) DIV 60;`

const ElectrolyserCleanupStatement = `DELETE FROM Electrolyser WHERE logged < DATE(DATE_ADD( now(), interval -1 month))`

const DryerArchiveStatement = `INSERT INTO firefly.Dryer (logged, temp1, temp2, temp3, temp4, inputPressure, outputPressure, warnings, errors)
SELECT min(logged), avg(temp1), avg(temp2), avg(temp3), avg(temp4), avg(inputPressure), avg(outputPressure), max(warnings), max(errors)
  FROM firefly.Dryer
WHERE logged < DATE(DATE_ADD( now(), interval -1 month))
 GROUP BY UNIX_TIMESTAMP(logged) DIV 60;`

const DryerCleanupStatement = `DELETE FROM Dryer WHERE logged < DATE(DATE_ADD( now(), interval -1 month))`

// Modbus registers defined at https://handbook.enapter.com/electrolyser/el21_firmware/1.9.3/modbus_tcp_communication_interface.html#references
// Holding Registers

const REBOOT = 4
const LOCATE = 5

const MAINTENANCE = 6
const StartStop = 1000
const RATE = 1002
const BlowDown = 1010
const REFILL = 1011

//const MAINTENANCE = 1013

const PREHEAT = 1014
const BeginConfig = 4000
const CommitConfig = 4001

//const SetIP = 4020
//const SetIPMask = 4022
//const SetIPGateway = 4024
//const Serial = 4026
//const CloudLoggingEnable = 4038
//const CloudLoggingDisable = 4042
//const SyslogIP = 4044
//const SyslogPort = 4046
//const Altitude = 4142

const MaxPressure = 4308
const RestartPressure = 4310

//const StackSerial = 4376
//const DefaultRate = 4396
//const CoolingType = 4494
//const HeartBeat = 4600
//const HeartBeatGatewayTimeout = 4602
//const HeartBeatUCMTimeout = 4604
//const WarmUpperIOP = 4900
//const MinStackCurrent = 4910
//const StackCurrentCheckThreshold = 4912
//const StackCurrentPeriod = 4914
//const MembranePeriod = 4916
//const MembranePressureThreshold = 4926
//const MembraneVoltageThreshold = 4928
//const DryerStartThreshold = 6014
//const DryerStandbyThreshold = 6016

const DryerStartStop = 6018
const DryerStop = 6019
const DryerReboot = 6020

//const CoolingValve = 7005

// Input registers

const Model = 0

// const Firmware = 2                       //	Uint16	Firmware MAJOR and MINOR Version	Ex: 267 => 267 // 256 = 1, 267 % 256 => 11 (1.11)
// const Patch = 3                          // Uint16	Firmware PATCH Version	Ex: 3 => 3 (3)
// const Build = 4                          // Uint32	Firmware Build Number	e.g. 0x4E343471
const BoardSerial = 6 //	Uint128	Device Control Board Serial Number	9E25E695-A66A-61DD-6570-50DB4E73652D

const ChassisSerial = 14 //	Uint64	Chassis Serial Number	1 bits - reserved, must be 0 10 bits - Product Unicode, 11 bits - Year + Month, 5 bits - Day, 24 bits - Chassis Number, 5 bits - Order, 8 bits - Site
const SystemState = 18   // Uint16	System State	0 = Internal Error, System not Initialized yet; 1 = System in Operation; 2 = Error; 3 = System in Maintenance Mode; 4 = Fatal Error; 5 = System in Expert Mode.
// const LiveTime = 20                      // Uint32	Live time [seconds]	Total time during which a system is power up (not only time when stack is working).
// const UpTime = 22                        // Uint32	Uptime [seconds]	How long the system has been running
// const FreeMemory = 26                    // Uint32	Free memory = Memory which can be used
// const AvailableMemory = 28               // Uint32	Available memory = Memory which has not been allocated yet
// const FlashCardSpace = 30                // Uint32	Free space on flash-card	Space on flash-card where the configuration is

const WarningsArray = 768 // Array of 32 Warning Events Array	Warning Events Array represented by Error Codes. First Uint16 contains total quantity of Warning Events.
const ErrorsArray = 832   // Array of 32 Error Events Array	Error Events Array represented by Error Codes. First Uint16 contains total quantity of Error Events.
const ProductCode = 1000  // Uint32	Product Code
// const StackCycles = 1002                 // Uint32	Stack Start/Stop Cycles Quantity	How many Stack Start/Stop cycles
// const StackRuntime = 1004                // Uint32	Stack Total Runtime	seconds
// const StackTotalProduction = 1006 		// Float32	Stack Total H2 Production	NL
// const FlowRate = 1008 // Float32	H2 Flow Rate	NL/hour, NAN when not producing H2;
// const StackSerial = 1010 // Uint64	Stack Serial Number	1 bits - reserved, must be 0, 15 bits - Stack Type , 11 bits - Year + Month , 5 bits - Day, 24 bits - Stack Number, 8 bits - Site

const State = 1200 // Uint16	Electrolyser State	0 = Halted; 1= Maintenance mode; 2 = Idle; 3 = Steady; 4 = Stand-By (Max Pressure); 5 = Curve; 6 = BlowDown.
//const ConfigInProgress = 4000            // Boolean	Configuration Progress	1 = Configuration is in progress.
//const ConfigViaModbus = 4001             // Boolean	Configuration Source	1 = Configuration over Modbus.
//const LastConfigResult = 4002            // Int32	Last Configuration Result	0 = OK, Configuration was completed successfully; 1 = Permanent, The operation has failed (internal or general error); 2 = No Entry, Configuration was not started or interrupted; 5 = I/O, Data save error; 11 - Try again, Configuration needs to be tried again; 13 = Access Denied, Some changed registers are read-only; 16 = Busy, Another configuration was in progress; 22 = Invalid, The data has invalid or wrong type.
//const LastConfigWrongHolding = 4004      // Uint16	Last Configuration Wrong Holding	Keeps first invalid Holding register number which doesn't allow successful configuration commit.
//const HeatBeatTimeout = 4600             // Uint16	Heartbeat	Timeout for Modbus Heartbeat in seconds. 0 = disabled (default)

const DryerError = 6000 // Uint16	Dryer Error code (bitmask).
//const DryerWarning = 6001                // Uint16	Dryer Warning	Dryer warning code (bitmask).

const DryerTemp0 = 6002 // Float32	Dryer TT00	Temperature of heater element for cartridge 0 (first line).
// const DryerTemp1 = 6004                  // Float32	Dryer TT01	Temperature of heater element for cartridge 1 (second line).
// const DryerTemp2 = 6006                  // Float32	Dryer TT02	Temperature of heater element for cartridge 2 (first line).
// const DryerTemp3 = 6008                  // Float32	Dryer TT03	Temperature of heater element for cartridge 3 (second line).
// const DryerInputPressure = 6010          // Float32	Dryer PT00	Input pressure of the dryer.
// const DryerOutputPressure = 6012         // Float32	Dryer PT01	Output pressure of the dryer.
// const WifiOUI = 6100                     // Uint32	3 OUI octets for Wi-Fi MAC address	Ex: C8:2B:96
// const WifiMAC = 6102                     // Uint32	3 NIC octets for Wi-Fi MAC address	Ex: A8:F5:2C
const DryerNetworkStatus = 6104 // Boolean	Dryer Control Network Connection Status	1 = Online; 0 = Offline

const ElectrolyteHigh = 7000 // Boolean	High Electrolyte Level Switch (LSH102B_in)	1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
// const ElectrolyteVeryHigh = 7001         // Boolean	Very High Electrolyte Level Switch 1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
// const ElectrolyteLow = 7002              // Boolean	Low Electrolyte Level Switch 1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
// const ElectrolyteMedium = 7003           // Boolean	Medium Electrolyte Level Switch 1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
// const ElectrolytePressureHigh = 7004     // Boolean	Electrolyte Tank High Pressure Switch	1 = Pressure is too high; 0 = Pressure is normal.
// const ElectrolytePressureVeryHigh = 7005 // Boolean	Very High Hydrogen Pressure Switch	1 = Pressure is too high; 0 = Pressure is normal.
// const DownstreamHighTempSwitch = 7006    // Boolean	Downstream High Temperature Switch	1 = Temperature is too high. 0 = Temperature is normal.
// const ElectronicsHighTempSwitch = 7007   // Boolean	Electronic Compartment High Temperature Switch	1 = Temperature is too high. 0 = Temperature is normal.
// const ElectrolyteTempSwitch = 7008       // Boolean	Very Low Electrolyte Temperature Switch	1 = Temperature is too low. 0 = Temperature is normal.
// const ChassisWaterPresenceSwitch = 7009  // Boolean	Chassis Water Presence Switch	1 = Water is present on input; 0 = No water input.
// const DryContact = 7010                  // Boolean	Dry Contact	1 = OK (Closed); 0 = NOT OK (Opened)

const ElectrolyteCoolingFan = 7500 // Float32	Electrolyte Cooler Fan Speed (F103A_in_rpm)	[rpm]
//const AirCirculationSpeed = 7502         // Float32	Air Circulation Fan Speed (F104B_in_rpm)	[rpm]
//const ElectronicsCoolingSpeed = 7504     // Float32	Electronic Compartment Cooling Fan Speed (F108C_in_rpm)	[rpm]
//const ElectrolyteFlow = 7506             // Float32	Electrolyte Flow Meter (FM106_in_lpm)	[Liters per minute]

//const StackCurrent = 7508 // Float32	Stack Current	[Ampere]
//const PSUVoltage = 7510                  // Float32	PSU Voltage (Stack Voltage) (PSU_in_v)	[Volt]
//const InnerH2Pressure = 7512             // Float32	Inner Hydrogen Pressure (PT101A_in_bar)	[bar]
//const OuterH2Pressure = 7514             // Float32	Outer Hydrogen Pressure (PT101C_in_bar)	[bar]
//const WaterInletPressure = 7516          // Float32	Water Inlet Pressure (PT105_in_bar)	[bar]
//const ElectrolyteTemp = 7518             // Float32	Electrolyte Temperature (TT102A_in_c)	[°C]
//const DownStreamTemp = 7520              // Float32	Downstream Temperature (TT106_in_c)	[°C]
//const InnerH2RawPressure = 8000          // Float32	Inner Hydrogen Pressure Raw Sensor Value (PT101A_in_v)	Raw value, [Volt]
//const OuterH2RawPressure = 8002          // Float32	Outer Hydrogen Pressure Raw Sensor Value (PT101C_in_v)	Raw value, [Volt]
//const StackCurrentRaw = 8004

//const ElStandby = 4

/*
Query strings to return historical data
*/

const ElectrolyserDataByMinute = `select UNIX_TIMESTAMP(logged) as logged
						,avg(flow) as flow
						,avg(volts) as volts
						,avg(amps) as amps
						,avg(temperature) as temperature
						,avg(waterPressure) as waterPressure
						,avg(rate) as rate
						,avg(innerH2Pressure) as innerH2Pressure
						,avg(outerH2Pressure) as outerH2Pressure
						,avg(electrolyteLevel) as electrolyteLevel
						,avg(electrolyteFlow) as electrolyteFlow
						,avg(electrolyteFanSpeed) as electrolyteFanSpeed
						,avg(electronicFanSpeed) as electronicFanSpeed
						,avg(airFanSpeed) as airFanSpeed
						,avg(downstreamTemperature) as downstreamTemperature
					from Electrolyser
				   where name = ?
				     and logged between ? and ?
			    group by UNIX_TIMESTAMP(logged) div 60`

const ElectrolyserDataBySecond = `select UNIX_TIMESTAMP(logged) as logged
						,flow
						,volts
						,amps
						,temperature
						,waterPressure
						,rate
						,innerH2Pressure
						,outerH2Pressure
						,electrolyteLevel
						,electrolyteFlow
						,electrolyteFanSpeed
						,electronicFanSpeed
						,airFanSpeed
						,downstreamTemperature
					from Electrolyser
				   where name = ?
				     and logged between ? and ?`

type jsonFloat32 float32

func (value jsonFloat32) MarshalJSON() ([]byte, error) {
	if math.IsNaN(float64(value)) {
		log.Println("Tried to convert NaN float value to JSON")
		return json.Marshal(nil)
	} else {
		return json.Marshal(float32(value))
	}
}

type ElectrolyserEventsType struct {
	count uint16
	codes [31]uint16
}

type ElectrolyteLevelType byte

const (
	empty ElectrolyteLevelType = iota
	low
	medium
	high
	veryHigh
)

func (l ElectrolyteLevelType) String() string {
	switch l {
	case empty:
		return "Empty"
	case low:
		return "Low"
	case medium:
		return "Medium"
	case high:
		return "High"
	case veryHigh:
		return "Very High"
	}
	return "ERROR bad level"
}

type DryerStatusType struct {
	Temps          [4]jsonFloat32 `json:"temps"`
	InputPressure  jsonFloat32    `json:"inputPressure"`
	OutputPressure jsonFloat32    `json:"outputPressure"`
	Errors         uint16         `json:"errors"`
	Warnings       uint16         `json:"warnings"`
}

//	Firmware              string
//	DefaultProductionRate jsonFloat32 `json:"defaultRate"`        // H4396

type ElectrolyserType struct {
	status ElectrolyserStatusType
	//	OnOffTime          time.Time
	// OffDelayTime       time.Time
	OffRequested       *time.Timer
	Client             *modbus.ModbusClient
	clientConnected    bool
	connectErrorCount  int
	lastConnectAttempt time.Time
	failedConnections  uint8
	powerRelay         uint8
	poweredOn          time.Time
	hasDryer           bool // This is updated as the electrolysers are running. Only one should be in control of the dryer
	enabled            bool
	mu                 sync.Mutex
	buf                bytes.Buffer
	stopTime           time.Time
	startTime          time.Time
	MonitorTrigger     *time.Timer
}

// ElectrolysersType defines an array of electrolysers and provide a mutex to control access
type ElectrolysersType struct {
	Arr []*ElectrolyserType
	mu  sync.Mutex
}

func (el *ElectrolysersType) Init() {
	// Update the electrolysers or add new ones as necessary
	el.mu.Lock()
	defer el.mu.Unlock()

	// Sort the array by stack time
	els := currentSettings.Electrolysers[:]
	slices.SortStableFunc(els, func(a ElectrolyserSettingType, b ElectrolyserSettingType) int {
		return cmp.Compare(a.StackTime, b.StackTime)
	})
	//	log.Printf("Electrolysers: %s, %s, %s", els[0].Name, els[1].Name, els[2].Name)

	for _, el := range currentSettings.Electrolysers {
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
				elect.status.Enabled = el.Enabled
				elect.enabled = el.Enabled
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
				newEl.enabled = el.Enabled
				newEl.status.Serial = el.Serial
				Electrolysers.Arr = append(Electrolysers.Arr, newEl)
			}
		}
	}
	// Copy all the electrolysers that match one in the settings by relay
	newArr := make([]*ElectrolyserType, 0)
	for _, el := range Electrolysers.Arr {
		if currentSettings.FindElectrolyserByRelay(el.powerRelay) != nil {
			newArr = append(newArr, el)
		}
	}
	// If the length is different, then replace the array to get rid of ones we no longer have
	if len(newArr) != len(Electrolysers.Arr) {
		Electrolysers.Arr = newArr
	}
}

// FindByName returns a pointer to the electrolyser with the matching name or a nil pointer if not found
func (el *ElectrolysersType) FindByName(name string) *ElectrolyserType {
	for idx := range el.Arr {
		if strings.ToLower(el.Arr[idx].status.Name) == strings.ToLower(name) {
			return el.Arr[idx]
		}
	}
	if idx, err := strconv.Atoi(name); err == nil {
		if idx >= 0 && idx < len(el.Arr) {
			return el.Arr[idx]
		}
	}
	return nil
}

// FindByRelay returns a pointer to the electrolyser with the assigned relay that matches that given or a nil pointer if not found
func (el *ElectrolysersType) FindByRelay(relay uint8) *ElectrolyserType {
	for idx := range el.Arr {
		if el.Arr[idx].powerRelay == relay {
			return el.Arr[idx]
		}
	}
	return nil
}

// name, flow, volts, amps, temperature, errors, warnings, waterPressure, rate, innerH2Pressure,
// outerH2Pressure, electrolyteLevel, electrolyteFlow, electronicFanSpeed, airFanSpeed, electrolyteFanSpeed, downstreamTemperature

// RecordData writes the current status values to the database
func (el *ElectrolyserType) RecordData(stmt *sql.Stmt) error {
	_, err := stmt.Exec(el.status.Name,
		el.status.H2Flow,
		el.status.StackVoltage,
		el.status.StackCurrent,
		el.status.ElectrolyteTemp,
		el.GetErrorText(),
		el.GetWarningText(),
		el.status.WaterPressure,
		el.status.CurrentProductionRate,
		el.status.InnerH2Pressure,
		el.status.OuterH2Pressure,
		el.status.ElectrolyteLevel,
		el.status.ElectrolyteFlowMeter,
		el.status.ElectronicCompartmentCoolingFanSpeed,
		el.status.AirCirculationFanSpeed,
		el.status.ElectrolyteCoolerFanSpeed,
		el.status.DownstreamTemp)
	return err
}

func (el *ElectrolyserType) getJsonStatus() ([]byte, error) {
	var st ElectrolyserJSONStatusType
	el.mu.Lock()
	defer el.mu.Unlock()

	el.IsSwitchedOn()
	st.load(el.status)
	return json.Marshal(st)
}

func (el *ElectrolyserType) setClient(IP net.IP) {
	var logger *log.Logger

	if IP.Equal(net.IPv4zero) {
		if el.Client != nil {
			el.Client = nil
		}
		el.status.IP = net.IPv4zero
		return
	}
	var config modbus.ClientConfiguration
	config.Timeout = 1 * time.Second // 1 second timeout
	config.URL = "tcp://" + IP.String() + ":502"
	logger = log.New(&el.buf, "MODBUS", 0)
	config.Logger = logger
	el.status.IP = IP
	el.status.Serial = ""
	if Client, err := modbus.NewClient(&config); err != nil {
		log.Printf("%s - New modbus client error - %v", el.status.Name, err)
		return
	} else {
		el.Client = Client
		var err error
		for tries := 0; tries < 5; tries++ {
			if err = el.Client.Open(); err == nil {
				break
			}
			//			el.Client.Close()
			time.Sleep(time.Duration(5) * time.Second)
			if debugOutput {
				log.Printf("%d - New modbus client error %s - %v", tries, el.status.Name, err)
			}
		}
		if err != nil {
			log.Printf("%s error - %v", el.status.Name, err)
			el.Client = nil
			el.connectErrorCount++
			if el.connectErrorCount > 5 {
				if elConfig := currentSettings.findElByIP(IP.String()); elConfig != nil {
					log.Printf("Rescanning for %s", elConfig.Name)
					if ip, elType, err := el.rescan(0, elConfig.Serial); err != nil {
						log.Printf("%s error %v", el.status.Name, err)
						return
					} else {
						el.status.IP = ip
						log.Printf("%s electrolyser with serial number %s found at %s", elType, el.status.Serial, ip.String())
						currentSettings.findElByName(el.status.Name).IP = ip.String()
						if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
							log.Print(err)
						}
					}
					el.connectErrorCount = 0
				} else {
					log.Println("Failed to find electrolyser configuration for ", IP.String())
				}
			}
		} else {
			el.connectErrorCount = 0
		}
	}
}

func NewElectrolyser(ip net.IP) *ElectrolyserType {
	e := new(ElectrolyserType)
	//	e.OnOffTime = time.Now().Add(0 - (time.Minute * 30))
	// e.OffDelayTime = time.Now()
	e.OffRequested = nil

	if debugOutput {
		log.Printf("Adding an electrolyser at [%s]\n", ip)
	}
	//	e.setClient(ip)
	e.status.IP = ip
	return e
}

func (el *ElectrolyserType) GetRate() int {
	el.mu.Lock()
	defer el.mu.Unlock()
	r := int(el.status.CurrentProductionRate)
	if (el.OffRequested != nil) && (r == 60) {
		return 0
	} else {
		return r
	}
}

func (el *ElectrolyserType) BoardSerial() (string, error) {
	if !el.CheckConnected() {
		if debugOutput {
			log.Printf("%s not connected", el.status.Name)
		}
		return "", fmt.Errorf("%s not connected", el.status.Name)
	}
	if debugOutput {
		log.Printf("Reading %s Board Serial Number", el.status.Name)
	}
	if ID, err := el.Client.ReadUint64s(BoardSerial, 2, modbus.INPUT_REGISTER); err != nil {
		return "", err
	} else {
		return fmt.Sprintf("%08x%08x", ID[0], ID[1]), nil
	}
}

func DecodeStackSerialV1(serial uint64) string {
	var decoded struct {
		StackType string
		Year      int
		Month     int
		Day       int
		Number    int
		Site      string
	}
	st := (serial >> 48) & 0x7F
	switch st {
	case 1:
		decoded.StackType = "23E"
	case 2:
		decoded.StackType = "23D"
	case 3:
		decoded.StackType = "23V"
	default:
		decoded.StackType = fmt.Sprint(st)
	}
	ym := (serial >> 37) & 0x7FF
	decoded.Year = int((ym - 1) / 12)
	decoded.Month = int((ym-1)%12) + 1
	decoded.Day = int((serial >> 32) & 0x1F)
	decoded.Number = int((serial >> 8) & 0xFFFFFF)
	site := serial & 0xFF
	switch site {
	case 0:
		decoded.Site = "PI"
	case 1:
		decoded.Site = "SA"
	default:
		decoded.Site = fmt.Sprint(site)
	}
	//	fmt.Printf("type: %s | year: %d | month: %d | day: %d | number: %d | site: %s\n", decoded.StackType, decoded.Year, decoded.Month, decoded.Day, decoded.Number, decoded.Site)
	return fmt.Sprintf("%s%02d%02d%02d%02d%s", decoded.StackType, decoded.Year, decoded.Month, decoded.Day, decoded.Number, decoded.Site)
}

func DecodeChassisSerialV1(serial uint64) string {
	var codes struct {
		Site    string
		Order   string
		Chassis uint32
		Day     uint8
		Month   uint8
		Year    uint16
		Product string
	}
	//  1 bit - reserved, must be 0
	// 10 bits - Product Unicode
	// 11 bits - Year + Month
	//  5 bits - Day
	// 24 bits - Chassis Number
	//  5 bits - Order
	//  8 bits - Site

	Site := uint8(serial & 0xff)
	switch Site {
	case 0:
		codes.Site = "PI"
	case 1:
		codes.Site = "SA"
	default:
		codes.Site = "XX"
	}

	var Order [1]byte
	Order[0] = byte((serial>>8)&0x1f) + 64
	codes.Order = string(Order[:])

	codes.Chassis = uint32((serial >> 13) & 0xffffff)
	codes.Day = uint8((serial >> 37) & 0x1f)
	yearMonth := (serial >> 42) & 0x7ff
	codes.Year = uint16(yearMonth / 12)
	codes.Month = uint8(yearMonth % 12)
	if codes.Month == 0 {
		codes.Month = 12
		codes.Year--
	}
	Product := uint16((serial >> 53) & 0x3ff)

	var unicode [2]byte
	unicode[1] = byte(Product>>5) + 64
	unicode[0] = byte(Product&0x1f) + 64
	codes.Product = string(unicode[:])

	return fmt.Sprintf("%s%02d%02d%02d%02d%s%s", codes.Product, codes.Year, codes.Month, codes.Day, codes.Chassis, codes.Order, codes.Site)
}

func DecodeSerialV2(serial uint64) string {
	var decoded struct {
		Product string
		Year    int
		Month   int
		Day     int
		Number  int
		Site    string
	}
	prod := make([]byte, 1)
	prodSerial := (serial >> 38) & 0xFFFFFF
	prod[0] = byte((prodSerial % 26) + 'A')
	prodSerial = prodSerial / 26
	char := (prodSerial % 27)
	if char > 0 {
		prod = append(prod, byte(prodSerial%27)+'@')
	}
	for {
		prodSerial = prodSerial / 27
		char = (prodSerial % 27)
		if char > 0 {
			prod = append(prod, byte(prodSerial%27)+'@')
		} else {
			break
		}
	}

	decoded.Product = string(prod)

	ym := (serial >> 27) & 0x7FF
	decoded.Year = int((ym - 1) / 12)
	decoded.Month = int((ym-1)%12) + 1
	decoded.Day = int((serial >> 22) & 0x1F)

	decoded.Number = int(serial & 0xFFF)

	site := (serial >> 12) & 0x3FF
	siteCode := make([]byte, 2)
	siteCode[0] = byte(site%26) + 'A'
	site = site / 26
	siteCode[1] = byte(site%27) + '@'
	decoded.Site = string(siteCode)

	//	fmt.Printf("type: %s | year: %d | month: %d | day: %d | number: %d | site: %s\n", decoded.StackType, decoded.Year, decoded.Month, decoded.Day, decoded.Number, decoded.Site)
	return fmt.Sprintf("%s %02d %02d %02d %s %d", decoded.Product, decoded.Year, decoded.Month, decoded.Day, decoded.Site, decoded.Number)
}

// ReadSerialNumber reads and decodes the serial number
func (el *ElectrolyserType) ReadSerialNumber(IP ...net.IP) (string, error) {

	if len(IP) > 0 {
		el.setClient(IP[0])
	}

	if !el.CheckConnected() {
		if debugOutput {
			log.Printf("%s not connected", el.status.Name)
		}
		return "", fmt.Errorf("%s not connected", el.status.Name)
	}
	if debugOutput {
		log.Printf("Reading %s serial number", el.status.Name)
	}
	serialCode, err := el.Client.ReadUint64(ChassisSerial, modbus.INPUT_REGISTER)
	if err != nil {
		if strings.Contains(err.Error(), "broken pipe") {
			// We lost communication, so we should try to recreate the pipe
			if err := el.Client.Close(); err != nil {
				log.Printf("attempt to close modbus connection to %s returned %v", el.status.Name, err)
			}
			el.clientConnected = false
			if err := el.Client.Open(); err != nil {
				log.Printf("attempt to reopen modbus connection to %s returned %v", el.status.Name, err)
				el.Client = nil
				return "", fmt.Errorf("broken pipe to %s - failed to restablish connection", el.status.Name)
			} else {
				if serialCode2, err := el.Client.ReadUint64(ChassisSerial, modbus.INPUT_REGISTER); err != nil {
					log.Printf("Error getting serial number from %s after reconnect - %v", el.status.Name, err)
					return "", err
				} else {
					serialCode = serialCode2
				}
			}
		} else {
			log.Printf("Error getting serial number from %s - %v", el.status.Name, err)
			return "", err
		}
	}
	if debugOutput {
		log.Printf("Got serial number from %s - %0x", el.status.Name, serialCode)
	}
	if serialCode == 0 {
		if debugOutput {
			log.Printf("no serial number found for %s!", el.status.Name)
		}
		if ucm, err := el.BoardSerial(); err != nil {
			return "", err
		} else {
			return fmt.Sprintf("UCM-%s", ucm), nil
		}
		//		return fmt.Sprintf("No Serial Number %s", el.status.IP.String()), nil
	}

	if serialCode&0x8000000000000000 > 0 {
		return DecodeSerialV2(serialCode), nil
	} else {
		return DecodeChassisSerialV1(serialCode), nil
	}
}

// Is the electrolyser switched on? If the relay turns on, wait 20 seconds before returning true.
func (el *ElectrolyserType) IsSwitchedOn() bool {
	if !Relays.GetRelay(el.powerRelay) {
		el.poweredOn = time.Time{}
		el.status.Powered = false
		return false
	} else {
		if el.poweredOn.IsZero() {
			el.poweredOn = time.Now()
		}
	}
	if el.status.Powered {
		return true
	}
	if time.Since(el.poweredOn) > (time.Second * 20) {
		el.status.Powered = true
		if debugOutput {
			log.Printf("%s powered on", el.status.Name)
		}
	} else {
		if debugOutput {
			log.Printf("Waiting for electrolyser %s", el.status.Name)
		}
	}
	return el.status.Powered
}

func (el *ElectrolyserType) CheckConnected() bool {
	//if el.Client == nil {
	//	if debugOutput {
	//		log.Println("No client")
	//	}
	//	//		return false
	//	if IP, err := GetOurSubnet(); err != nil {
	//		log.Println(err)
	//		return false
	//	} else {
	//		IP[3] = 1
	//		el.setClient(IP)
	//	}
	//}
	if !el.IsSwitchedOn() {
		if debugOutput {
			log.Printf("%s is switched off", el.status.Name)
		}
		return false
	}
	if !el.clientConnected {
		if debugOutput {
			log.Printf("%s switched on but not yet connected", el.status.Name)
		}
		if time.Since(el.lastConnectAttempt) > time.Second*5 {
			err := fmt.Errorf("no client")
			if el.Client != nil {
				err = el.Client.Open()
			}
			if err != nil {
				log.Printf("%s modbus client.open error - %v", el.status.Name, err)
				el.clientConnected = false
				el.failedConnections++
				if el.failedConnections > 2 {
					//				if el.failedConnections > 10 {
					setting := currentSettings.FindElectrolyserByRelay(el.powerRelay)
					if setting == nil {
						log.Printf("%s not found in settings", el.status.Name)
						return false
					}
					log.Printf("seach for %s with serial number %s", el.status.Name, setting.Serial)
					if ip, elType, err := el.rescan(1, setting.Serial); err != nil {
						log.Println(err)
						el.clientConnected = false
						return false
					} else {
						el.status.IP = ip
						if ip.Equal(net.IPv4zero) {
							log.Printf("Failed to find %s", el.status.Name)
							el.clientConnected = false
							return false
						} else {
							log.Printf("Found %s - %s at %s", el.status.Name, elType, ip.String())
							currentSettings.findElByName(el.status.Name).IP = ip.String()
							if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
								err = errors.Join(fmt.Errorf("Error whilst trying to save the settings for %s", el.status.Name), err)
								log.Print(err)
							}
						}
					}
					el.failedConnections = 0
				} else {
					log.Printf("%s waiting for failed connections > 10 (%d)\n", el.status.Name, el.failedConnections)
				}
			} else {
				el.clientConnected = true
				if debugOutput {
					log.Printf("%s connected...", el.status.Name)
				}
			}
			el.lastConnectAttempt = time.Now()
		} else {
			if debugOutput {
				log.Printf("%s - Time since last connection attempt = %v", el.status.Name, time.Since(el.lastConnectAttempt))
			}
		}
	} else {
		el.failedConnections = 0
	}
	return el.clientConnected
}

func (el *ElectrolyserType) readEvents() {
	if !el.CheckConnected() {
		return
	}
	events, err := el.Client.ReadRegisters(WarningsArray, 32, modbus.INPUT_REGISTER)
	if err != nil {
		log.Printf("Modbus read register error %s - %v", el.status.Name, err)
		if err := el.Client.Close(); err != nil {
			log.Printf("Error closing modbus client to %s - %v", el.status.Name, err)
			log.Print(el.buf)
		}
		el.clientConnected = false
		return
	}

	el.status.Warnings.count = events[0]
	copy(el.status.Warnings.codes[:], events[1:])

	events, err = el.Client.ReadRegisters(ErrorsArray, 32, modbus.INPUT_REGISTER)
	if err != nil {
		log.Printf("%s modbus read register error - %v", el.status.Name, err)
		log.Print(el.buf)
		if err := el.Client.Close(); err != nil {
			log.Printf("Error closing modbus client to %s - %v", el.status.Name, err)
			log.Print(el.buf)
		}
		el.clientConnected = false
		return
	}
	el.status.Errors.count = events[0]
	copy(el.status.Errors.codes[:], events[1:])
}

func (el *ElectrolyserType) ReadModelNumber() error {
	modelNumber, err := el.Client.ReadUint32(0, modbus.INPUT_REGISTER)
	if err != nil {
		if debugOutput {
			log.Printf("No active device for %s found at - %s - %v", el.status.Name, el.status.IP.String(), err)
		}
		el.status.Model = ""
		return err
	}
	// Is this an EL21?
	switch modelNumber {
	case 0x454C3231:
		el.status.Model = "EL-21"
	case 0x45533430:
		el.status.Model = "ES-40"
	case 0x45533431:
		el.status.Model = "ES-41"
	default:
		el.status.Model = ""
		if debugOutput {
			log.Printf("%s is not a known electrolyser type (0x%X)", el.status.Name, modelNumber)
		}
		return fmt.Errorf("%s is not a known electrolyser type (0x%X)", el.status.Name, modelNumber)
	}
	return nil
}

// ReadValues calls out to the electrolyser using ModbusTCP and gathers the current data
func (el *ElectrolyserType) ReadValues() error {
	// Add a modbus client if we do not have one assigned
	if debugOutput {
		log.Printf("Read values from %s - %s", el.status.Name, el.status.IP.String())
	}
	if el.Client == nil {
		if debugOutput {
			log.Println("Adding a modbus client")
		}
		el.setClient(el.status.IP)
	}

	if !el.CheckConnected() {
		if debugOutput {
			log.Println("Electrolyser " + el.status.Name + " is not connected")
		}
		el.status.ClearErrors()
		el.status.ClearWarnings()
		el.status.Clear()
		return fmt.Errorf("electrolyser %s is not connected", el.status.Name)
	}
	el.mu.Lock()
	defer el.mu.Unlock()

	// Get the model if we do not already have it
	if el.status.Model == "" {
		if err := el.ReadModelNumber(); err != nil {
			return err
		}
	}

	if serial, err := el.ReadSerialNumber(); err != nil {
		return err
	} else {
		if debugOutput {
			log.Printf("%s Serial number = %s : ip= %s", el.status.Name, serial, el.status.IP.String())
		}
		if el.status.Serial == "" {
			log.Printf("New serial number for %s = %s", el.status.Name, serial)
			for _, electrolyser := range Electrolysers.Arr {
				if electrolyser.status.Serial == serial {
					err := fmt.Errorf("%s is already assigned to %s. Trying to find %s", serial, electrolyser.status.Name, el.status.Name)
					log.Print(err)
					return err
				}
			}
			el.status.Serial = serial
			if elSetting := currentSettings.findElByName(el.status.Name); elSetting != nil {
				elSetting.IP = el.status.IP.String()
				elSetting.Serial = serial
				//if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
				//	log.Println("failed to save the settings. ", err)
				//} else {
				//	log.Printf("New ip assigned to %s : %s", el.status.Name, el.status.IP.String())
				//}
			} else {
				log.Println("Settings for electrolyser " + el.status.Name + " not found.")
			}
		} else {
			if el.status.Serial != serial {
				log.Printf("electrolyser at %s has a different serial number (%s). Rescanning...", el.GetIPString(), serial)
				if ip, elType, err := el.rescan(2, el.status.Serial); err != nil {
					log.Println(err)
					return err
				} else {
					el.status.IP = ip
					log.Printf("%s electrolyser with serial number %s found at %s", elType, el.status.Serial, ip.String())
					currentSettings.findElByName(el.status.Name).IP = ip.String()
					//					if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
					//						log.Print(err)
					//					}
				}
			}
		}
	}

	// Get the stack current and voltage, innerH2 pressure, outerH2 pressure, water pressure and electrolyte temperature.
	if values, err := el.Client.ReadFloat32s(ElectrolyteCoolingFan, 11, modbus.INPUT_REGISTER); err != nil {
		log.Print("Modbus reading float32 values - ", err)
		log.Print(el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.ElectrolyteCoolerFanSpeed = jsonFloat32(values[0])
		el.status.AirCirculationFanSpeed = jsonFloat32(values[1])
		el.status.ElectronicCompartmentCoolingFanSpeed = jsonFloat32(values[2])
		el.status.ElectrolyteFlowMeter = jsonFloat32(values[3])
		el.status.StackCurrent = jsonFloat32(values[4])
		el.status.StackVoltage = jsonFloat32(values[5])
		el.status.InnerH2Pressure = jsonFloat32(values[6])
		el.status.OuterH2Pressure = jsonFloat32(values[7])
		el.status.WaterPressure = jsonFloat32(values[8])
		el.status.ElectrolyteTemp = jsonFloat32(values[9])
		el.status.DownstreamTemp = jsonFloat32(values[10])
	}

	//	log.Println("Electrolyte")
	if values, err := el.Client.ReadRegisters(ElectrolyteHigh, 11, modbus.INPUT_REGISTER); err != nil {
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.ElectrolyteLevel = empty
		if values[1] != 0 {
			el.status.ElectrolyteLevel = veryHigh
		} else if values[0] != 0 {
			el.status.ElectrolyteLevel = high
		} else if values[3] != 0 {
			el.status.ElectrolyteLevel = medium
		} else if values[2] != 0 {
			el.status.ElectrolyteLevel = low
		}
		el.status.ElectrolyteTankPressureTooHigh = values[4] != 0
		el.status.HydrogenPressureTooHigh = values[5] != 0
		el.status.DownstreamHighTemperature = values[6] != 0
		el.status.ElectronicCompartmentHighTemp = values[7] != 0
		el.status.VeryLowElectrolyteTemp = values[8] != 0
		el.status.ChassisWaterPresent = values[9] != 0
		el.status.DryContact = values[10] != 0
	}

	//	get the maximum tank and restart pressure settings if we don't have them
	//	log.Println("Restart Pressure")
	if el.status.MaxTankPressure == 0 || el.status.RestartPressure == 0 {
		p, err := el.Client.ReadFloat32s(MaxPressure, 2, modbus.HOLDING_REGISTER)
		if err != nil {
			log.Print("Modbus reading max tank and restart pressure - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		}
		el.status.MaxTankPressure = jsonFloat32(p[0])
		el.status.RestartPressure = jsonFloat32(p[1])
	}
	if state, err := el.Client.ReadRegister(SystemState, modbus.INPUT_REGISTER); err != nil {
		log.Print("System state error - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.SystemState = state
	}
	if state, err := el.Client.ReadRegister(State, modbus.INPUT_REGISTER); err != nil {
		log.Printf("%s State - %v", el.status.Name, err)
		if err := el.Client.Close(); err != nil {
			log.Printf("Error closing modbus client for %s - %v", el.status.Name, err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.State = state
	}
	if state, err := el.Client.ReadRegister(DryerNetworkStatus, modbus.INPUT_REGISTER); err != nil {
		log.Printf("%s Dryer Network Status error - %v", el.status.Name, err)
		if err := el.Client.Close(); err != nil {
			log.Printf("Error closing modbus client for %s - %v", el.status.Name, err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.DryerNetworkEnabled = state != 0
	}

	//	log.Println("Product Code")
	if buffer, err := el.Client.ReadBytes(ProductCode, 28, modbus.INPUT_REGISTER); err != nil {
		log.Printf("%s product code error - %v", el.status.Name, err)
		if err := el.Client.Close(); err != nil {
			log.Printf("Error closing modbus client for %s - %v", el.status.Name, err)
		}
		el.clientConnected = false
		return err
	} else {
		buf := bytes.NewReader(buffer)
		var values struct {
			ProductCode          uint32  // Product Code
			StackStartStopCycles uint32  // 1002
			StackTotalRuntime    uint32  // 1004
			StackTotalProduction float32 // 1006
			H2Flow               float32 // 1008
			StackSerial          uint64  // 1010
		}
		if err := binary.Read(buf, binary.BigEndian, &values); err != nil {
			log.Println("error reading stack values from %s - %v", el.status.Name, err)
		}
		el.status.ProductCode = values.ProductCode
		el.status.StackStartStopCycles = values.StackStartStopCycles
		el.status.StackTotalRunTime = values.StackTotalRuntime
		el.status.StackTotalProduction = jsonFloat32(values.StackTotalProduction)
		if math.IsNaN(float64(values.H2Flow)) {
			el.status.H2Flow = 0.0
		} else {
			el.status.H2Flow = jsonFloat32(values.H2Flow)
		}
		if values.StackSerial&0x8000000000000000 > 0 {
			el.status.StackSerialNumber = DecodeSerialV2(values.StackSerial)
		} else {
			el.status.StackSerialNumber = DecodeStackSerialV1(values.StackSerial)
		}
	}

	if rate, err := el.Client.ReadFloat32(RATE, modbus.HOLDING_REGISTER); err != nil {
		log.Printf("%s current production rate error - %v", el.status.Name, err)
		if err := el.Client.Close(); err != nil {
			log.Printf("Error closing modbus client for %s - %v", el.status.Name, err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.CurrentProductionRate = jsonFloat32(rate)
	}

	//	log.Println("Events")
	el.readEvents()

	//	log.Println("Dryer")
	if el.hasDryer && !currentSettings.acquiringElectrolysers {
		dryer, err := el.Client.ReadFloat32s(DryerTemp0, 6, modbus.INPUT_REGISTER)
		if err != nil {
			log.Printf("%s dryer error - %v", el.status.Name, err)
			//			if err := el.Client.Close(); err != nil {
			//				log.Print("Error closing modbus client - ", err)
			//			}
			el.status.DryerFailure = err.Error() // Log the dryer communication failure
			//		el.clientConnected = false
			//			return err
			dryer = make([]float32, 6)
		}
		if el.status.Dryer == nil {
			el.status.Dryer = new(DryerStatusType)
		}

		if el.status.Dryer != nil {
			el.status.DryerFailure = "" // Clear any previous dryer failure error
			el.status.Dryer.Temps[0] = jsonFloat32(dryer[0])
			el.status.Dryer.Temps[1] = jsonFloat32(dryer[1])
			el.status.Dryer.Temps[2] = jsonFloat32(dryer[2])
			el.status.Dryer.Temps[3] = jsonFloat32(dryer[3])
			el.status.Dryer.InputPressure = jsonFloat32(dryer[4])
			el.status.Dryer.OutputPressure = jsonFloat32(dryer[5])
			if dryerErrors, err := el.Client.ReadRegisters(DryerError, 2, modbus.INPUT_REGISTER); err != nil {
				log.Printf("Error reading dryer errors from %s - %v", el.status.Name, err)
				if debugOutput {
					if el.CheckConnected() {
						log.Printf("%s connected", el.status.Name)
					} else {
						log.Printf("%s is NOT connected", el.status.Name)
					}
				}
				//if err := el.Client.Close(); err != nil {
				//	log.Print("Error closing modbus client - ", err)
				//}
				//el.clientConnected = false
				//return err
			} else {
				el.status.Dryer.Errors = dryerErrors[0]
				el.status.Dryer.Warnings = dryerErrors[1]
			}
		}
	} else {
		el.status.DryerFailure = "No Dryer"
	}
	//	log.Println("Electrolyser Errors")
	if !el.status.monitored {
		go el.MonitorElectrolyserErrors()
	}
	if debugOutput {
		log.Printf("%s values read...", el.status.Name)
	}
	return nil
}

func (el *ElectrolyserType) clearSerial() {
	el.status.Serial = ""
}

func (el *ElectrolyserType) getStatus() *ElectrolyserStatusType {
	el.mu.Lock()
	defer el.mu.Unlock()

	log.Printf("%s get status", el.status.Name)
	status := new(ElectrolyserStatusType)
	log.Println("Test Electrolyser")
	el.IsSwitchedOn()

	*status = el.status
	status.Errors.count = el.status.Errors.count
	for idx, st := range el.status.Errors.codes {
		status.Errors.codes[idx] = st
	}
	status.Warnings.count = el.status.Warnings.count
	for idx, st := range el.status.Warnings.codes {
		status.Warnings.codes[idx] = st
	}
	return status
}

func (el *ElectrolyserType) GetSystemState() string {
	el.mu.Lock()
	defer el.mu.Unlock()

	switch el.status.SystemState {
	case 0:
		return "Internal Error, System not Initialized yet"
	case 1:
		return "System in Operation"
	case 2:
		return "Error"
	case 3:
		return "System in Maintenance Mode"
	case 4:
		return "Fatal Error"
	case 5:
		return "System in Expert Mode"
	default:
		return "Unknown state"
	}
}

func decodeMessage(w uint16) string {
	dec := map[uint16]string{0x0FFF: "Internal error",
		0x108A: "Pump broken",
		0x1114: "Inner hydrogen pressure reading is below the expected value",
		0x118A: "Electrolyte level is too high",
		0x1194: "Electrolyte level is too low",
		0x11A8: "Refilling unsuccessful",
		0x11B2: "Conflict between water level sensors (low and medium)",
		0x11B3: "Conflict between water level sensors (medium and high)",
		0x11B4: "Conflict between water level sensors (high and very high)",
		0x1201: "Broken PSU",
		0x120A: "Broken membrane",
		0x120B: "Steady-state leak check failed",
		0x120C: "Insufficient stack current",
		0x128A: "Electrolyte temperature is too high",
		0x130C: "Insufficient stack current",
		0x1402: "Water presence detected",
		0x1403: "PSU broken",
		0x1404: "Stack current is too high",
		0x1405: "Backflow temperature is too high",
		0x1407: "Control board temperature is too high",
		0x1408: "Electrolyte tank pressure is too high",
		0x1409: "Electrolyte temperature is too low",
		0x140A: "Hydrogen pressure is too high",
		0x140B: "Control Board MCU temperature is too high",
		0x140C: "Outer hydrogen pressure is too high",
		0x140D: "High hydrogen presence detected",
		0x141E: "Water inlet pressure transmitter broken",
		0x141F: "Electrolyte tank temperature transmitter broken",
		0x1420: "Electrolyte flow meter broken",
		0x1421: "Electrolyte backflow temperature transmitter broken",
		0x1422: "Inner hydrogen pressure transmitter broken",
		0x1423: "Outer hydrogen pressure transmitter broken",
		0x1424: "Chassis circulation fan broken",
		0x1425: "Electronic compartment cooling fan broken",
		0x1426: "Electronic board temperature transmitter broken",
		0x1427: "Stack current sensor broken",
		0x1428: "Dry contact triggered",
		0x1429: "Water level sensor broken",
		0x142A: "Common trip",
		0x142B: "Insufficient electrolyte flow",
		0x148A: "Frozen pipes",
		0x1501: "Inner hydrogen pressure reading is below the expected value",
		0x159E: "Hydrogen Purge line obstruction or blowdown failure",
		0x170A: "Recombiner undertemperature",
		0x170B: "Recombiner overtemperature",
		0x1714: "Recombiner overcooling",
		0x1715: "Recombiner underheating",
		0x171E: "Recombiner frozen",
		0x171F: "Recombiner heater malfunction",
		0x178A: "INVALID COMMUNICATION PACKET",
		0x1794: "DRY CONTACT",
		0x1795: "COMMUNICATION FAILURE",
		0x1796: "SAFE STATE REQUESTED BY CONTROL BOARD",
		0x1797: "INCOMPATIBLE CONTROL BOARD VERSION",
		0x1798: "RELAY BROKEN",
		0x1799: "PROOF-TEST CANNOT BE STARTED",
		0x179A: "PROOF-TEST FAILED",
		0x179B: "INNER HYDROGEN PRESSURE TRANSMITTER BROKEN",
		0x179C: "INCORRECT CONTROL BOARD",
		0x179D: "RECOMBINER CHAMBER TEMPERATURE SENSOR BROKEN",
		0x179E: "TACHOMETER OR FAN BROKEN",
		0x179F: "SAFETY BOARD TEMPERATURE SENSOR BROKEN",
		0x17A0: "CONTROL BOARD AND SAFETY BOARD PAIRED",
		0x17A1: "Safety Board brownout",
		0x17B3: "SIF1: HYDROGEN STACK OVERPRESSURE",
		0x17B4: "SIF2: RECOMBINER CHAMBER OVERTEMPERATURE",
		0x17B5: "SIF3: WATER LEAKAGE DETECTION",
		0x17B6: "SIF4: TANK OVERPRESSURE",
		0x17B7: "SIF5: DILUTION FAN FOR HYDROGEN CONCENTRATION",
		0x17B8: "SIF6: SAFETY ELECTRONIC BOARD TEMPERATURE",
		0x1F81: "Brownout detected",
		0x1F82: "New configuration parameters added",
		0x1F83: "Broken periphery",
		0x1F86: "Insufficient resources for DCN / IDCN",
		0x228A: "Electrolyte temperature is too low",
		0x2314: "Target current could not be reached.",
		0x2401: "Inner hydrogen pressure is too high",
		0x318A: "Water inlet pressure is too high",
		0x3194: "Water inlet pressure is too low",
		0x3195: "Refilling timeout",
		0x3196: "Refilling failure",
		0x3197: "Draining timeout",
		0x31B3: "Drain completely",
		0x31B4: "Refill to high level",
		0x31B5: "Drain to high level",
		0x31B6: "Refill to medium level",
		0x3214: "Standby mode",
		0x3215: "Drift in inner hydrogen pressure sensor",
		0x3216: "Refilling is not occurring",
		0x321E: "Replace electrolyte",
		0x321F: "Derating due to temperature",
		0x3220: "Derating due to voltage",
		0x3294: "Electrolyte cooling fan broken",
		0x3295: "Slow electrolyte heating",
		0x330A: "Gas-side pressure is not atmospheric",
		0x330B: "Electrolyte level insufficient for start-up",
		0x3314: "Ramp-up leak check failed",
		0x340E: "Hydrogen presence detected",
		0x340F: "Hydrogen sensor regeneration",
		0x3410: "Electrolyte large temperature discrepancy",
		0x3432: "Inner hydrogen pressure check disabled",
		0x3433: "Water presence check disabled",
		0x3434: "Power supply unit check is disabled",
		0x3435: "Stack current check disabled",
		0x3436: "Electrolyte backflow temperature check disabled",
		0x3437: "Control board temperature check disabled",
		0x3438: "Electrolyte tank pressure check disabled",
		0x3439: "Low electrolyte temperature check disabled",
		0x343A: "Inner overpressure check disabled",
		0x343B: "Water inlet pressure check disabled",
		0x343C: "Electrolyte tank temperature check disabled",
		0x343D: "Electrolyte flow meter check disabled",
		0x343E: "Electrolyte cooling fan check disabled",
		0x343F: "Electrolyte backflow temperature check disabled",
		0x3440: "Outer hydrogen pressure check disabled",
		0x3441: "Chassis circulation fan check disabled",
		0x3442: "Electronic compartment cooling fan check disabled",
		0x3443: "Dry contact check disabled",
		0x3444: "Water level check disabled",
		0x3445: "Control Board temperature check disabled",
		0x3446: "Hydrogen sensor check disabled",
		0x3447: "Electrolyte flow check disabled",
		0x3448: "Electrolyte sensors discrepancy check disabled",
		0x348A: "Electrolyte anti-freeze routine is disabled",
		0x350A: "Insufficient pressure drop",
		0x358A: "Outer hydrogen pressure is too high for blowdown",
		0x3594: "Blowdown routine is active",
		0x360A: "Lost Modbus safety heartbeat communication",
		0x360B: "Lost Enapter Gateway safety heartbeat communication",
		0x360C: "Lost UCM communication",
		0x368A: "Polarization curve start failed",
		0x3701: "Recombiner anti-freeze routine is disabled",
		0x37B2: "SIF-test has been started",
		0x3F84: "Stuck power button",
		0x3F85: "Low battery voltage",
	}
	if txt := dec[w]; txt != "" {
		return txt
	}
	return fmt.Sprintf("Unknown error/warning : %x", w)
}

func (el *ElectrolyserType) GetSerial() string {

	el.mu.Lock()
	defer el.mu.Unlock()

	return el.status.Serial
}

func (el *ElectrolyserType) GetIPString() string {

	el.mu.Lock()
	defer el.mu.Unlock()

	return el.status.IP.String()
}

func (el *ElectrolyserType) GetWarnings() []string {
	el.mu.Lock()
	defer el.mu.Unlock()

	return el.status.GetWarnings()
}

func (el *ElectrolyserType) GetWarningText() string {
	return strings.Join(el.GetWarnings(), "\n")
}

func (el *ElectrolyserType) GetErrors() []string {
	el.mu.Lock()
	defer el.mu.Unlock()

	return el.status.GetErrors()
}

func (el *ElectrolyserType) GetErrorText() string {
	return strings.Join(el.GetErrors(), "\n")
}

func (el *ElectrolyserType) getState() string {

	el.mu.Lock()
	defer el.mu.Unlock()

	if el.status.Powered {
		switch el.status.State {
		case 0:
			return "Halted"
		case 1:
			return "Maintenance mode"
		case 2:
			return "Idle"
		case 3:
			return "Steady"
		case 4:
			return "Stand-By"
		case 5:
			return "Curve"
		case 6:
			return "Blow down"
		default:
			return "Unknown State"
		}
	} else {
		return "Off"
	}
}

func decodeDryerMessage(code uint16) []string {
	var e []string
	for b := uint16(1); b < 0b1000000000000000; b <<= 1 {
		if (code & b) > 0 {
			switch b {
			case 0b1:
				e = append(e, "TT00 has invalid value (sensor provides unexpected values)")
			case 0b10:
				e = append(e, "TT01 has invalid value (sensor provides unexpected values)")
			case 0b100:
				e = append(e, "TT02 has invalid value (sensor provides unexpected values)")
			case 0b1000:
				e = append(e, "TT03 has invalid value (sensor provides unexpected values)")
			case 0b10000:
				e = append(e, "TT00 value growth is not enough (heating mechanism does not work properly)")
			case 0b100000:
				e = append(e, "TT01 value growth is not enough (heating mechanism does not work properly)")
			case 0b1000000:
				e = append(e, "TT02 value growth is not enough (heating mechanism does not work properly)")
			case 0b10000000:
				e = append(e, "TT03 value growth is not enough (heating mechanism does not work properly)")
			case 0b100000000:
				e = append(e, "PS00 (pressure switch on line 0) is triggered")
			case 0b1000000000:
				e = append(e, "PS01 (pressure switch on line 1) is triggered")
			case 0b10000000000:
				e = append(e, "F100 has invalid RPM speed (fan between line 0 and line 1)")
			case 0b100000000000:
				e = append(e, "F101 has invalid RPM speed (fan on line 0)")
			case 0b1000000000000:
				e = append(e, "F102 has invalid RPM speed (fan on line 1)")
			case 0b10000000000000:
				e = append(e, "PT00 (Input pressure) has invalid value (sensor provides unexpected values)")
			case 0b100000000000000:
				e = append(e, "PT01 (Output pressure) has invalid value (sensor provides unexpected values)")
			}
		}
	}
	return e
}

func (el *ElectrolyserType) GetDryerTemp(idx int) interface{} {
	if math.IsNaN(float64(el.status.Dryer.Temps[idx])) {
		return nil
	} else {
		return el.status.Dryer.Temps[idx]
	}
}

func (el *ElectrolyserType) GetDryerErrors() []string {
	if el != nil {
		if el.status.Dryer != nil {
			return decodeDryerMessage(el.status.Dryer.Errors)
		}
	}
	return nil
}

func (el *ElectrolyserType) GetDryerWarnings() []string {
	if el != nil {
		if el.status.Dryer != nil {
			return decodeDryerMessage(el.status.Dryer.Warnings)
		}
	}
	return nil
}

func (el *ElectrolyserType) SendRateToElectrolyser(rate float32) error {
	err := el.Client.WriteFloat32(RATE, rate)
	if err != nil {
		log.Print("Error setting production rate - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
	}
	return err
}

// SetProduction sets the electrolyser to the rate given 0, 60..100
func (el *ElectrolyserType) SetProduction(rate uint8) {
	if debugOutput {
		log.Printf("Set electrolyser %s to %d", el.status.Name, rate)
	}

	if !el.CheckConnected() {
		return
	}
	if rate < 60 || rate > 100 {
		log.Printf("Invalid rate (%d) requested", rate)
		return
	}
	// 60% or more we should send the rate and clear the off timer
	if err := el.SendRateToElectrolyser(float32(rate)); err != nil {
		log.Println(err)
	}
	// If there is a pending delayed stop then kill the timer
	if el.OffRequested != nil {
		el.OffRequested.Stop()
		el.OffRequested = nil
	}
	// If the electrolyser is in Idle then start it.
	//if e.status.State == ElIdle {
	//	log.Println("Electrolyser is idle so sending a start command.")
	//	e.Start(false)
	//}
}

func (el *ElectrolyserType) SetRestartPressure(pressure float32) error {
	if !el.CheckConnected() {
		return fmt.Errorf("electrolyser is not turned on")
	}

	// Check configuration status
	status, err := el.Client.ReadRegister(BeginConfig, modbus.INPUT_REGISTER)
	if err != nil {
		log.Println("Cannot establish configuration status - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return fmt.Errorf("unable to set the restart pressure for the electrolyser. See the log file for more detail")
	}
	if status != 0 {
		if debugOutput {
			log.Println("configuration is already in progress")
		}
		return fmt.Errorf("configuration is already in progress")
	}

	//Begin configuration
	err = el.Client.WriteRegister(BeginConfig, 1)
	if err != nil {
		log.Println("Cannot start configuration - ", err, el.buf)
		return fmt.Errorf("start configuration failed")
	}
	status, err = el.Client.ReadRegister(BeginConfig, modbus.INPUT_REGISTER)
	if err != nil {
		log.Println("Cannot establish configuration status after configuration start - ", err)
		return fmt.Errorf("unable to set the restart pressure for the electrolyser. See the log file for more detail")
	}
	if status == 0 {
		log.Println("Configuration did not start.")
		return fmt.Errorf("configuration failed to start")
	}

	err = el.Client.WriteFloat32(RestartPressure, pressure)
	if err != nil {
		log.Print("Error setting electrolyser restart pressure - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return fmt.Errorf("unable to set the restart pressure for the electrolyser. See the log file for more detail")
	}

	err = el.Client.WriteRegister(CommitConfig, 1)
	if err != nil {
		log.Println("Commit configuration changes failed - ", err, el.buf)
		return fmt.Errorf("unable to commit the restart pressure change for the electrolyser. See the log file for more detail")
	}
	// Force a reread of the pressure the next time the values are read from the electrolyser
	el.mu.Lock()
	defer el.mu.Unlock()

	el.status.RestartPressure = 0

	return nil
}

// Start will Attempt to start the electrolyser - return an httpStatus. 200 if successful
func (el *ElectrolyserType) Start() (int, error) {
	if el.CheckConnected() {
		if time.Since(el.stopTime) > time.Minute*time.Duration(currentSettings.ElectrolyserStopToStartTime) {
			if !el.status.IsRunning() {
				el.startTime = time.Now()
			}
			if err := el.Client.WriteRegister(StartStop, 1); err != nil {
				log.Print("Error starting Electrolyser - ", err, el.buf)
				if err := el.Client.Close(); err != nil {
					log.Print("Error closing modbus client - ", err)
				}
				el.clientConnected = false
			} else {
				log.Printf("Start command sent to %s", el.status.Name)
				return http.StatusOK, nil
			}
		} else {
			return http.StatusConflict, fmt.Errorf("too soon after last stop - %s. Start allowed after %s", el.stopTime.Format(time.Kitchen), el.stopTime.Add(time.Minute*time.Duration(currentSettings.ElectrolyserStopToStartTime)).Format(time.Kitchen))
		}
	}
	return http.StatusBadRequest, fmt.Errorf("electrolyser is not connected")
}

/*
Stop -  Attempt to stop the electrolyser - returns an http.Status. 200 if successful
*/
func (el *ElectrolyserType) Stop() (int, error) {
	if el.CheckConnected() {
		if time.Since(el.startTime) > time.Minute*time.Duration(currentSettings.ElectrolyserStartToStopTime) {
			if el.status.IsRunning() {
				el.stopTime = time.Now()
			}
			// Send the stop command
			if err := el.Client.WriteRegister(StartStop, 0); err != nil {
				log.Print("Error stopping electrolyser - ", err, el.buf)
				if err := el.Client.Close(); err != nil {
					log.Print("Error closing modbus client - ", err)
				}
				el.clientConnected = false
			} else {
				log.Printf("Stop command sent to %s", el.status.Name)
				currentSettings.findElByName(el.status.Name).StackTime = uint32(int32(el.status.StackTotalRunTime) - el.status.elm.StackTimeOffset)
				if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
					log.Print("Error saving settings - ", err)
				}
				return http.StatusOK, nil
			}
		} else {
			return http.StatusConflict, fmt.Errorf("too soon after start command - %s. Stop allowed after %s", el.startTime.Format(time.Kitchen), el.startTime.Add(time.Minute*time.Duration(currentSettings.ElectrolyserStartToStopTime)).Format(time.Kitchen))
		}
	}
	return http.StatusBadRequest, fmt.Errorf("electrolyser is not connected")
}

// Preheat will start the pre-heat cycle
func (el *ElectrolyserType) Preheat() error {
	if el.CheckConnected() {
		if el.status.ElectrolyteTemp < 26 {
			err := el.Client.WriteRegister(PREHEAT, 1)
			if err != nil {
				log.Print("Preheat Request failed - ", err, el.buf)
				if err := el.Client.Close(); err != nil {
					log.Print("Error closing modbus client - ", err)
				}
				el.clientConnected = false
				return err
			} else {
				log.Printf("Preheat command sent to %s", el.status.Name)
			}
		} else {
			return fmt.Errorf("preheat request ignored as temperature is already %f C", el.status.ElectrolyteTemp)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// Reboot attempts to reboot the electrolyser
func (el *ElectrolyserType) Reboot() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(REBOOT, 1)
		if err != nil {
			log.Print("reboot Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Reboot command sent to %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// Locate starts the location cycle which flashes the LEDs on the front panel
func (el *ElectrolyserType) Locate() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(LOCATE, 1)
		if err != nil {
			log.Print("Locate Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Locate command sent to %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// EnableMaintenance puts the electrolyser into maintenance mode
func (el *ElectrolyserType) EnableMaintenance() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(MAINTENANCE, 1)
		if err != nil {
			log.Print("enable maintenance request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Enable maintenance command sent to %s", el.status.Name)
			return nil
		}
	}
	return fmt.Errorf("electrolyser is not connected")
}

// DisableMaintenance will stop the maintenance cycle
func (el *ElectrolyserType) DisableMaintenance() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(MAINTENANCE, 0)
		if err != nil {
			log.Print("disable maintenance request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Disable maintenance command sent to %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// BlowDown starts the blow-down process
func (el *ElectrolyserType) BlowDown() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(BlowDown, 1)
		if err != nil {
			log.Print("Blow down Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Blowdown command sent to %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// Refill starts the refill process
func (el *ElectrolyserType) Refill() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(REFILL, 1)
		if err != nil {
			log.Print("Refill Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("%s refill command sent with level at %s", el.status.Name, el.status.ElectrolyteLevel)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// StartDryer will start a connected dryer
func (el *ElectrolyserType) StartDryer() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(DryerStartStop, 1)
		if err != nil {
			log.Print("Start Dryer Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Dryer star command sent via %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// StopDryer will stop a connected dryer
func (el *ElectrolyserType) StopDryer() error {
	if el.CheckConnected() {
		err := el.Client.WriteRegister(DryerStop, 1)
		if err != nil {
			log.Print("Stop Dryer Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Dryer stop command sent via %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// RebootDryer will send a reboot command to a connected dryer
func (el *ElectrolyserType) RebootDryer() error {
	if el.CheckConnected() {
		if err := el.Client.WriteRegister(DryerReboot, 1); err != nil {
			log.Printf("Reboot Dryer via %s Request failed - ", el.status.Name, err)
			if err := el.Client.Close(); err != nil {
				log.Printf("Error closing modbus client to %s - %v", el.status.Name, err)
			}
			el.clientConnected = false
			return err
		} else {
			log.Printf("Dryer reboot command sent via %s", el.status.Name)
		}
		return nil
	}
	return fmt.Errorf("%s is not connected", el.status.Name)
}

// GetDryerErrorsHTML returns any errors from the connected dryer in an HTML table
func (el *ElectrolyserType) GetDryerErrorsHTML() string {
	var htmlString = "<table>"

	for _, err := range el.GetDryerErrors() {
		htmlString += "<tr><td>" + html.EscapeString(err) + "</td></tr>"
	}
	htmlString += "</table>"
	return htmlString
}

// GetDryerErrorText returns any errors from the connected dryer
func (el *ElectrolyserType) GetDryerErrorText() string {
	return strings.Join(el.GetDryerErrors(), "\n")
}

// GetDryerWarningsHTML returns any errors from the connected dryer in an  HTML table
func (el *ElectrolyserType) GetDryerWarningsHTML() string {
	var htmlString = "<table>"

	for _, warning := range el.GetDryerWarnings() {
		htmlString += "<tr><td>" + html.EscapeString(warning) + "</td></tr>"
	}
	htmlString += "</table>"
	return htmlString
}

// GetDryerWarningText returns any errors from the connected dryer
func (el *ElectrolyserType) GetDryerWarningText() string {
	return strings.Join(el.GetWarnings(), "\n")
}

// Acquire searches for an electrolyser
// Search is conducted in the current subnet.
func (el *ElectrolyserType) Acquire() error {
	// Make sure all electrolysers are powered down.
	for _, el := range currentSettings.Electrolysers {
		if Relays.GetRelay(el.PowerRelay) {
			return fmt.Errorf("all electrolysers must be turned off before performing a search")
		}
	}
	if debugOutput {
		log.Print("turning on the electrolyser")
	}
	// Turn on the relay and give it 15 seconds to come online.
	Relays.SetRelay(el.powerRelay, true)
	defer Relays.SetRelay(el.powerRelay, false)

	time.Sleep(time.Second * 30)

	// Try and find the electrolyser
	if debugOutput {
		log.Print("Searching...")
	}
	if el.Client != nil {
		//		e.Client.Close()
		el.Client = nil
		if debugOutput {
			log.Println("Client closed")
		}
	}
	ip, elType, err := scan(1)

	if err == nil {
		// 5 second delay
		log.Println("Waiting for 5 seconds...")
		time.Sleep(time.Second * 5)

		el.mu.Lock()
		log.Println("Set the client for ", ip.String())
		el.setClient(ip)
		el.status.Model = elType
		el.mu.Unlock()

		if debugOutput {
			log.Println("Read the current values")
		}
		el.status.Serial = ""
		return el.ReadValues()
	} else {
		return err
	}
}

// GetOurSubnet returns the subnet for this computer
func GetOurSubnet() (net.IP, error) {
	//conn, err := net.Dial("udp", "8.8.8.8:80")
	//if err != nil {
	//	return nil, err
	//}
	//defer func() {
	//	if err := conn.Close(); err != nil {
	//		log.Print(err)
	//	}
	//}()
	//
	//localAddr := conn.LocalAddr().(*net.UDPAddr)
	//return localAddr.IP, nil

	ip := net.ParseIP(currentSettings.Subnet)
	return ip, nil
}

// scan searches from StartIP until it finds a recognisable electrolyser
func scan(StartIP byte) (net.IP, string, error) {
	var subnetIP net.IP
	if subnet, err := GetOurSubnet(); err != nil {
		log.Println(err)
	} else {
		subnetIP = subnet
	}
	currentSettings.scanningElectrolysers = true
	defer func() { currentSettings.scanningElectrolysers = false }()

	log.Println("Subnet = ", subnetIP.String())

	for ip := StartIP; ip < 255; ip++ {
		IP := subnetIP.To4()
		IP[3] = ip
		if debugOutput {
			log.Print(IP.String())
		}
		if tryConnect(IP, 502) == nil {
			// Something is there and responding on the Modbus port.
			if elType, err := CheckForElectrolyser(IP); err == nil {
				// It is an electrolyser so return its IP
				if debugOutput {
					log.Println("Electrolyser", elType, "found at ", IP)
				}
				return IP, elType, nil
			}
		}
	}
	return net.IPv4zero, "Not Found", fmt.Errorf("no electrolyser found")
}

// rescan searches from StartIP until it finds a recognisable electrolyser with the same serial number
func (el *ElectrolyserType) rescan(StartIP byte, knownSerial string) (net.IP, string, error) {
	var subnetIP net.IP
	var serial string

	if el.clientConnected {
		if err := el.Client.Close(); err != nil {
			log.Println(err)
		}
		el.clientConnected = false
	}

	if subnet, err := GetOurSubnet(); err != nil {
		log.Println(err)
	} else {
		subnetIP = subnet
	}
	if debugOutput {
		log.Println("Subnet = ", subnetIP.String())
	}

	currentSettings.scanningElectrolysers = true
	defer func() { currentSettings.scanningElectrolysers = false }()

	el.status.IP = net.IPv4zero
	for tries := 0; tries < 5; tries++ {
		for ip := StartIP; ip < 255; ip++ {
			IP := subnetIP.To4()
			IP[3] = ip
			if debugOutput {
				log.Print(IP.String())
			}
			if tryConnect(IP, 502) == nil {
				// Something is there and responding on the Modbus port.
				if elType, err := CheckForElectrolyser(IP); err == nil {
					// It is an electrolyser so check the serial number
					time.Sleep(time.Second * 5)
					if s, err := el.ReadSerialNumber(IP); err != nil {
						return IP, "", err
					} else {
						serial = s
					}
					if debugOutput {
						log.Println("Looking for ", knownSerial)
						log.Println("Found       ", serial)
					}
					if knownSerial == serial || strings.TrimSpace(knownSerial) == "" {
						// Looks like we have our device, so we should update the IP address.
						if debugOutput {
							log.Println("Electrolyser", elType, "found at ", IP)
						}
						return IP, elType, nil
					}
				} else {
					if debugOutput {
						log.Println(err)
					}
					//				return IP, elType, err
				}
			}
		}
		time.Sleep(time.Second * 10)
	}
	return net.IPv4zero, "Not Found", fmt.Errorf("no electrolyser found")
}

// tryConnect attempts to connect to the IP on the given port. err is nil if it succeeds
func tryConnect(host net.IP, port int) error {
	timeout := time.Millisecond * 25

	fmt.Print(net.JoinHostPort(host.String(), fmt.Sprint(port)), "\r")
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host.String(), fmt.Sprint(port)), timeout)
	if err != nil {
		return err
	}
	if conn != nil {
		defer func() {
			if err := conn.Close(); err != nil {
				log.Print(err)
			}
		}()
		return nil
	}
	return fmt.Errorf("unknown error")
}

// CheckForElectrolyser tests for an electrolyser.
// Given that we can connect on the Modbus port, test to see if this looks like an electrolyser by
// checking for the serial number.
func CheckForElectrolyser(ip net.IP) (string, error) {
	var config modbus.ClientConfiguration
	config.Timeout = 1 * time.Second // 1 second timeout
	config.URL = "tcp://" + ip.String() + ":502"
	if Client, err := modbus.NewClient(&config); err == nil {
		if err := Client.Open(); err != nil {
			return "", err
		}
		defer func() {
			if err := Client.Close(); err != nil {
				log.Print(err)
			}
		}()
		if debugOutput {
			log.Printf("Checking for model number on %s", ip.String())
		}
		model, err := Client.ReadUint32(Model, modbus.INPUT_REGISTER)
		if err != nil {
			log.Println("Not an electrolyser - ", err)
			return "", err
		}
		// Is this a EL21, ES40 or ES41?
		if debugOutput {
			log.Printf("Electrolyser %0x found at %s", model, ip.String())
		}
		switch model {
		case 0x454C3231:
			return "EL-21", nil
		case 0x45533430:
			return "ES-40", nil
		case 0x45533431:
			return "ES-41", nil
		default:
			return "", fmt.Errorf("not an EL21, ES40 or ES41 (%0x)", model)
		}
	} else {
		if debugOutput {
			log.Printf("Failed to get a new Modbus client - %v", err)
		}
		return "", err
	}
}

// MonitorElectrolyserErrors will watch for electrolyser errors and reboot a maximum of 5 times.
func (el *ElectrolyserType) MonitorElectrolyserErrors() {
	if !el.status.monitored {
		el.status.monitored = true
		elTicker := time.NewTicker(time.Minute)
		numErrors := 0 // Errors is incremented each time we reboot until we power cycle.
		numLoops := 0  // Ensures that a reboot is only if we have seen multiple halt conditions
		if debugOutput {
			log.Print("Starting electrolyser monitor for ", el.status.Name)
		}
		for {
			select {
			case <-elTicker.C:
				// Drop out if the electrolyser is switched off
				if !el.IsSwitchedOn() {
					log.Print("Stopping electrolyser monitor for ", el.status.Name)
					return
				}
				// Every minute, check for errors if we have seen less than 5 errors since turning the electrolyser on
				if el.status.State == 0 && numErrors < 5 {
					numLoops++
					if numLoops > 2 {
						log.Print("Rebooting ", el.status.Name)
						if err := el.Reboot(); err != nil {
							log.Print(err)
						} else {
							numErrors++
						}
						numLoops = 0
					}
				}
			}
		}
	}
}

// MonitorDryerErrors will watch for a dryer error and if it occurs it will reboot the dryer a maximum of 5 times.
func (el *ElectrolyserType) MonitorDryerErrors() {
	dryerTicker := time.NewTicker(time.Second * 5) // Check every five seconds
	numErrors := 0
	numSeconds := 0
	for {
		select {
		case <-dryerTicker.C:
			// Drop out if the electrolyser is switched off or does not have a dryer attached
			//			log.Printf("%s Monitor dryer errors", el.status.Name)
			if !el.IsSwitchedOn() || !el.hasDryer {
				return
			}
			// Every minute, check for errors if we have seen less than 5 errors since turning the dryer on
			if numSeconds > 60 {
				if len(el.GetDryerErrors()) > 0 && numErrors < 5 {
					if err := el.RebootDryer(); err != nil {
						log.Print(err)
					} else {
						numErrors++
					}
				}
				numSeconds = 0
			} else {
				numSeconds++
			}
		}
	}
}
