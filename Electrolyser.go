package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/simonvetter/modbus"
	"html"
	"log"
	"math"
	"net"
	"net/http"
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

//const Firmware = 2                       //	Uint16	Firmware MAJOR and MINOR Version	Ex: 267 => 267 // 256 = 1, 267 % 256 => 11 (1.11)
//const Patch = 3                          // Uint16	Firmware PATCH Version	Ex: 3 => 3 (3)
//const Build = 4                          // Uint32	Firmware Build Number	e.g. 0x4E343471
//const BoardSerial = 6                    //	Uint128	Device Control Board Serial Number	9E25E695-A66A-61DD-6570-50DB4E73652D

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

const StackTotalProduction = 1006 // Float32	Stack Total H2 Production	NL
// const FlowRate = 1008 // Float32	H2 Flow Rate	NL/hour, NAN when not producing H2;

const StackSerial = 1010 // Uint64	Stack Serial Number	1 bits - reserved, must be 0, 15 bits - Stack Type , 11 bits - Year + Month , 5 bits - Day, 24 bits - Stack Number, 8 bits - Site

const State = 1200 // Uint16	Electrolyser State	0 = Halted; 1= Maintenance mode; 2 = Idle; 3 = Steady; 4 = Stand-By (Max Pressure); 5 = Curve; 6 = BlowDown.
//const ConfigInProgress = 4000            // Boolean	Configuration Progress	1 = Configuration is in progress.
//const ConfigViaModbus = 4001             // Boolean	Configuration Source	1 = Configuration over Modbus.
//const LastConfigResult = 4002            // Int32	Last Configuration Result	0 = OK, Configuration was completed successfully; 1 = Permanent, The operation has failed (internal or general error); 2 = No Entry, Configuration was not started or interrupted; 5 = I/O, Data save error; 11 - Try again, Configuration needs to be tried again; 13 = Access Denied, Some changed registers are read-only; 16 = Busy, Another configuration was in progress; 22 = Invalid, The data has invalid or wrong type.
//const LastConfigWrongHolding = 4004      // Uint16	Last Configuration Wrong Holding	Keeps first invalid Holding register number which doesn't allow successful configuration commit.
//const HeatBeatTimeout = 4600             // Uint16	Heartbeat	Timeout for Modbus Heartbeat in seconds. 0 = disabled (default)

const DryerError = 6000 // Uint16	Dryer Error code (bitmask).
//const DryerWarning = 6001                // Uint16	Dryer Warning	Dryer warning code (bitmask).

const DryerTemp0 = 6002 // Float32	Dryer TT00	Temperature of heater element for cartridge 0 (first line).
//const DryerTemp1 = 6004                  // Float32	Dryer TT01	Temperature of heater element for cartridge 1 (second line).
//const DryerTemp2 = 6006                  // Float32	Dryer TT02	Temperature of heater element for cartridge 2 (first line).
//const DryerTemp3 = 6008                  // Float32	Dryer TT03	Temperature of heater element for cartridge 3 (second line).
//const DryerInputPressure = 6010          // Float32	Dryer PT00	Input pressure of the dryer.
//const DryerOutputPressure = 6012         // Float32	Dryer PT01	Output pressure of the dryer.
//const WifiOUI = 6100                     // Uint32	3 OUI octets for Wi-Fi MAC address	Ex: C8:2B:96
//const WifiMAC = 6102                     // Uint32	3 NIC octets for Wi-Fi MAC address	Ex: A8:F5:2C
//const DryerNetworkStatus = 6104          // Boolean	Dryer Control Network Connection Status	1 = Online; 0 = Offline

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
	if value != value {
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
	status             ElectrolyserStatusType
	OnOffTime          time.Time
	OffDelayTime       time.Time
	OffRequested       *time.Timer
	Client             *modbus.ModbusClient
	clientConnected    bool
	connectErrorCount  int
	lastConnectAttempt time.Time
	failedConnections  uint8
	powerRelay         uint8
	hasDryer           bool // This is updated as the electrolysers are running. Only one should be in control of the dryer
	enabled            bool
	mu                 sync.Mutex
	buf                bytes.Buffer
	stopTime           time.Time
	startTime          time.Time
}

// ElectrolysersType defines an array of electrolysers and provide a mutex to control access
type ElectrolysersType struct {
	Arr []*ElectrolyserType
	mu  sync.Mutex
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
		if err != nil {
			log.Print("New modbus client error - ", err)
			return
		}
	} else {
		el.Client = Client
		if err := el.Client.Open(); err != nil {
			log.Print(err)
			el.Client = nil
			el.connectErrorCount++
			if el.connectErrorCount > 20 {
				if elConfig := currentSettings.findElByIP(IP.String()); elConfig != nil {
					log.Println("Rescanning for ", elConfig.Name)
					if ip, elType, err := el.rescan(0, elConfig.Serial); err != nil {
						log.Print(err)
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
	e.OnOffTime = time.Now().Add(0 - (time.Minute * 30))
	e.OffDelayTime = time.Now()
	e.OffRequested = nil

	if debugOutput {
		log.Printf("Adding an electrolyser at [%s]\n", ip)
	}
	e.setClient(ip)
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

// ReadSerialNumber reads and decodes the serial number
func (el *ElectrolyserType) ReadSerialNumber(IP ...net.IP) (string, error) {
	type Codes struct {
		Site    string
		Order   string
		Chassis uint32
		Day     uint8
		Month   uint8
		Year    uint16
		Product string
	}

	var codes Codes

	if len(IP) > 0 {
		el.setClient(IP[0])
	}

	if !el.CheckConnected() {
		if debugOutput {
			log.Println("Not connected")
		}
		return "", fmt.Errorf("not conencted")
	}
	if debugOutput {
		log.Println("Reading the serial number")
	}
	serialCode, err := el.Client.ReadUint64(ChassisSerial, modbus.INPUT_REGISTER)
	if err != nil {
		if strings.Contains(err.Error(), "broken pipe") {
			// We lost communication, so we should try to recreate the pipe
			if err := el.Client.Close(); err != nil {
				log.Println("attempt to close modbus connection returned ", err)
			}
			el.clientConnected = false
			if err := el.Client.Open(); err != nil {
				log.Println("attempt to reopen modbus connection returned ", err)
				el.Client = nil
				return "", fmt.Errorf("broken pipe - failed to restablish connection")
			} else {
				if serialCode2, err := el.Client.ReadUint64(ChassisSerial, modbus.INPUT_REGISTER); err != nil {
					log.Println("Error getting serial number after reconnect - ", err)
					return "", err
				} else {
					serialCode = serialCode2
				}
			}
		} else {
			log.Println("Error getting serial number - ", err)
			return "", err
		}
	}
	if debugOutput {
		log.Println("Got serial number")
	}

	//  1 bit - reserved, must be 0
	// 10 bits - Product Unicode
	// 11 bits - Year + Month
	//  5 bits - Day
	// 24 bits - Chassis Number
	//  5 bits - Order
	//  8 bits - Site

	Site := uint8(serialCode & 0xff)
	switch Site {
	case 0:
		codes.Site = "PI"
	case 1:
		codes.Site = "SA"
	default:
		codes.Site = "XX"
	}

	var Order [1]byte
	Order[0] = byte((serialCode>>8)&0x1f) + 64
	codes.Order = string(Order[:])

	codes.Chassis = uint32((serialCode >> 13) & 0xffffff)
	codes.Day = uint8((serialCode >> 37) & 0x1f)
	yearMonth := (serialCode >> 42) & 0x7ff
	codes.Year = uint16(yearMonth / 12)
	codes.Month = uint8(yearMonth % 12)
	Product := uint16((serialCode >> 53) & 0x3ff)

	var unicode [2]byte
	unicode[1] = byte(Product%32) + 64
	unicode[0] = byte(Product/32) + 64
	codes.Product = string(unicode[:])

	return fmt.Sprintf("%s%02d%02d%02d%02d%s%s", codes.Product, codes.Year, codes.Month, codes.Day, codes.Chassis, codes.Order, codes.Site), nil
}

func (el *ElectrolyserType) IsSwitchedOn() bool {
	el.status.Powered = Relays.GetRelay(el.powerRelay)
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
			log.Println("Client Switched Off")
		}
		return false
	}
	if !el.clientConnected {
		if time.Since(el.lastConnectAttempt) > time.Second*5 {
			err := fmt.Errorf("no client")
			if el.Client != nil {
				err = el.Client.Open()
			}
			if err != nil {
				log.Print("modbus client.open error - ", err)
				el.clientConnected = false
				el.failedConnections++
				if el.failedConnections > 2 {
					//				if el.failedConnections > 10 {
					setting := currentSettings.FindElectrolyserByRelay(el.powerRelay)
					if setting == nil {
						log.Println("Electrolyser not found in settings")
						return false
					}
					log.Printf("seach for electrolyser with serial number %s\n", setting.Serial)
					if ip, elType, err := el.rescan(1, setting.Serial); err != nil {
						el.status.IP = ip
						if ip.Equal(net.IPv4zero) {
							log.Printf("Failed to find %s", el.status.Name)
							el.clientConnected = false
						} else {
							log.Printf("Found %s - %s at %s", el.status.Name, elType, ip.String())
							currentSettings.findElByName(el.status.Name).IP = ip.String()
							if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
								log.Print(err)
							}
						}
					}
					el.failedConnections = 0
				} else {
					log.Printf("Waiting for failed connections > 10 (%d)\n", el.failedConnections)
				}
			} else {
				el.clientConnected = true
				if debugOutput {
					log.Println("Connected...")
				}
			}
			el.lastConnectAttempt = time.Now()
		} else {
			if debugOutput {
				log.Printf("Time since last connection attempt = %v", time.Since(el.lastConnectAttempt))
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
		log.Print("Modbus read register error - ", err)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
			log.Print(el.buf)
		}
		el.clientConnected = false
		return
	}

	el.status.Warnings.count = events[0]
	copy(el.status.Warnings.codes[:], events[1:])

	events, err = el.Client.ReadRegisters(ErrorsArray, 32, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("Modbus read register error - ", err)
		log.Print(el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
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
			log.Println("No active device found at - "+el.status.IP.String(), err)
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
	default:
		el.status.Model = ""
		if debugOutput {
			log.Println("not an EL21 or ES40")
		}
		return fmt.Errorf("not an EL21 or ES40")
	}
	return nil
}

// ReadValues calls out to the electrolyser using ModbusTCP and gathers the current data
func (el *ElectrolyserType) ReadValues() error {
	// Add a modbus client if we do not have one assigned
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
					if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
						log.Print(err)
					}
				}
			} else {
				if debugOutput {
					log.Printf("serial number read from %s = %s", el.status.Name, serial)
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

	//	log.Println("System State")
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
	//	log.Println("State")
	if state, err := el.Client.ReadRegister(State, modbus.INPUT_REGISTER); err != nil {
		log.Print("ElState - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.State = state
	}

	//	log.Println("Product Code")
	if stack, err := el.Client.ReadUint32s(ProductCode, 3, modbus.INPUT_REGISTER); err != nil {
		log.Print("Product Code error - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {

		el.status.ProductCode = stack[0]
		el.status.StackStartStopCycles = stack[1]
		el.status.StackTotalRunTime = stack[2]
	}

	//	log.Println("Stack Total Production")
	if stack, err := el.Client.ReadFloat32s(StackTotalProduction, 2, modbus.INPUT_REGISTER); err != nil {
		log.Print("Current Production error - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.StackTotalProduction = jsonFloat32(stack[0])
		if math.IsNaN(float64(stack[1])) {
			el.status.H2Flow = 0.0
		} else {
			el.status.H2Flow = jsonFloat32(stack[1])
		}
	}
	//	log.Println("Stack Serial")
	if stack, err := el.Client.ReadUint64(StackSerial, modbus.INPUT_REGISTER); err != nil {
		log.Print("Stack Serial error - ", err, el.buf)
		if err := el.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		el.clientConnected = false
		return err
	} else {
		el.status.StackSerialNumber = stack
	}
	//	log.Println("Rate")
	if rate, err := el.Client.ReadFloat32(RATE, modbus.HOLDING_REGISTER); err != nil {
		log.Print("current production rate error - ", err)
		if err := el.Client.Close(); err != nil {
			log.Print("error closing modbus client - ", err)
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
			log.Println("Dryer error", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			el.status.DryerFailure = err.Error() // Log the dryer communication failure
			return err
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
			dryerErrors, err := el.Client.ReadRegisters(DryerError, 2, modbus.INPUT_REGISTER)
			if err != nil {
				log.Print("Error reading dryer errors - ", err, el.buf)
				if err := el.Client.Close(); err != nil {
					log.Print("Error closing modbus client - ", err)
				}
				el.clientConnected = false
				return err
			}
			el.status.Dryer.Errors = dryerErrors[0]
			el.status.Dryer.Warnings = dryerErrors[1]
		}
	} else {
		el.status.DryerFailure = "No Dryer"
	}
	//	log.Println("Electrolyser Errors")
	if !el.status.monitored {
		go el.MonitorElectrolyserErrors()
	}
	return nil
}

func (el *ElectrolyserType) getStatus() *ElectrolyserStatusType {
	el.mu.Lock()
	defer el.mu.Unlock()

	status := new(ElectrolyserStatusType)

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
	switch w {
	case 0:
		return "No error"
	case 0x0FFF:
		return "Hardware failure : Unexpected error"
	case 0x1F81:
		return "Voltage < 2.9V : Brownout detected"
	case 0x1F82:
		return "Updated firmware has new mandatory settings : New parameters have been added to the configuration"
	case 0x1F83:
		return "Hardware failure : Broken periphery"
	case 0x3F84:
		return "Power Button pressed for longer than 5sec	Sticky button : Power button is pushed."
	case 0x3F85:
		return "Too low battery. : Main board battery charge is to low."
	case 0x108A:
		return "Pump broken. : The electrolyte pump may be damaged."
	case 0x1114:
		return "Pressure drop > 2% : Possible hydrogen leak"
	case 0x318A:
		return "Pressure > 5 Bar : Water inlet pressure too high"
	case 0x3194:
		return "Pressure < 1.0 Bar : Water inlet pressure too low	Please provide water input pressure to the water inlet."
	case 0x118A:
		return "Water level is over very high level switch : Electrolyte level is too high. Please switch the electrolyser into maintenance mode and decrease the electrolyte level."
	case 0x1194:
		return "Water level is below low level switch	Electrolyte level too low. Please switch the electrolyser into maintenance mode, drain fully and then fill the electrolyte tank with fresh electrolyte solution."
	case 0x11B2:
		return "Conflict between water level sensors (low and medium level)"
	case 0x11B3:
		return "Conflict between water level sensors (medium and high level)"
	case 0x11B4:
		return "Conflict between water level sensors (high and very high level)"
	case 0x11A8:
		return "Refilling unsuccessful."
	case 0x3195:
		return "Refilling timeout	Please reboot device and ensure water inlet requirements are met."
	case 0x3196:
		return "Refilling failure	The refilling failed. Check water water supply system."
	case 0x31B3:
		return "Available only in Maintenance Mode	Drain completely	Electrolyte level is below minimum level. Electrolyser is ready for refill."
	case 0x31B4:
		return "Available only in Maintenance Mode	Refill to high level	Please continue filling the electrolyte."
	case 0x31B5:
		return "Electrolyte level is very high, drain to high level."
	case 0x1201:
		return "PSU bad current. PSU might be broken."
	case 0x120A:
		return "Broken membrane. Membrane inside the stack might be broken."
	case 0x3215:
		return "Pressure spike > 2%	Drifting PT101A. Pressure mismatch towards stack status has been detected."
	case 0x3216:
		return "System works with electrolyte level less than medium one and can not refill (during pressure limit and etc)	Refilling not happening	Please check the water supply - otherwise, the hydrogen production will stop soon."
	case 0x321E:
		return "Stack voltage is too high	Replace electrolyte	Replace electrolyte. If the error persists."
	case 0x128A:
		return "Temperature > 58°C	Electrolyte temperature too high	Please make sure that air ventilation is unobstructed or cooling liquid cooling loop operating and that ambient temperatures do not exceed device specifications"
	case 0x3294:
		return "Rotation < 600rpm	Electrolyte cooling fan broken. The electrolyte cooling fan should be checked."
	case 0x228A:
		return "Temperature < 6°C	Electrolyte temperature too low	Please make sure that room temperature is at least 6°C. Keep the EL powered to ensure the heating routine continues to protect the device internals."
	case 0x330A:
		return "Pressure is > atmospheric pressure + 10%	Gas side pressure is not atmospheric	Purge line pressure detected. Ramp-Up is not possible. Please check that the purge line is unobstructed."
	case 0x230A:
		return "Cannot start the heater because the water level in the internal electrolyser tank is too low.	Not enough warmup water	Heater can't be started due to a low electrolyte level. Refill electrolyser, restart and try again."
	case 0x1401:
		return "Pressure > 37bar	Hydrogen inner pressure too high. The hydrogen inner pressure exceeded 37 Bar (nominal, but high)."
	case 0x1402:
		return "Water sensor is wet	Water presence. Water is leaking inside the electrolyser. Please remove the water supply and power from the system and drain immediately."
	case 0x1403:
		return "No voltage from PSU	PSU broken. PSU failure detected. No voltage on stack."
	case 0x1404:
		return "Current > 58A	Stack current too high. Stack over current detected."
	case 0x1405:
		return "Back flow temperature too high. The stack outlet temperature is too high."
	case 0x1407:
		return "Temperature > 75°C	Electronic board temperature too high	The electronic board temperature is too high. Please check and clean ventilation openings."
	case 0x1408:
		return "vent line obstruction	Electrolyte tank pressure too high	Please make sure that O2 vent line is not blocked."
	case 0x1409:
		return "Electrolyte temperature too low	Please make sure that room temperature is at least 6°C. Keep the EL powered to ensure the heating routine continues to protect the device internals."
	case 0x140A:
		return "Hydrogen pressure too high. pressure transmitter calibration needs to be verified."
	case 0x140B:
		return "Temperature Sensor	Temperature > 75°C	Control Board MCU temperature too high	Please make sure that room temperature below 45°C."
	case 0x141E:
		return "Water inlet pressure transmitter broken. The water inlet pressure cannot be measured or bad water inlet pressure."
	case 0x141F:
		return "Electrolyte tank temperature transmitter broken. The electrolyte tank temperature cannot be measured."
	case 0x1420:
		return "Electrolyte flow meter broken. The electrolyte flow cannot be measured."
	case 0x1421:
		return "Electrolyte back flow temperature transmitter broken. The electrolyte back flow temperature cannot be measured."
	case 0x1422:
		return "Hydrogen inner pressure transmitter broken. The hydrogen inner pressure cannot be measured."
	case 0x1423:
		return "Outer hydrogen pressure transmitter broken. The outer hydrogen pressure cannot be measured."
	case 0x1424:
		return "Rotation < 3000rpm	Chassis circulation fan broken. The chassis air circulation fan speed cannot be measured."
	case 0x1425:
		return "Rotation < 3000rpm	Electronic compartment cooling fan broken. The electronic compartment cooling fan speed cannot be measured."
	case 0x1426:
		return "Electronic board temperature transmitter broken. The electronic board temperature cannot be measured."
	case 0x1427:
		return "Current sensor broken. The stack current cannot be measured."
	case 0x1428:
		return "Dry contact error	Dry contact triggered system stop. Please check your system to understand what triggered the dry contact."
	case 0x3432:
		return "Hydrogen inner pressure check disabled."
	case 0x3433:
		return "Water presence check disabled."
	case 0x3434:
		return "PSU check disabled."
	case 0x3435:
		return "Stack current check disabled."
	case 0x3436:
		return "Back flow temperature check disabled."
	case 0x3437:
		return "Electronic board temperature check disabled"
	case 0x3438:
		return "Electrolyte tank pressure check disabled."
	case 0x3439:
		return "Low electrolyte temperature check disabled."
	case 0x343B:
		return "Inlet pressure check disabled."
	case 0x343C:
		return "Electrolyte tank temperature check disabled."
	case 0x343D:
		return "Electrolyte flow meter check disabled."
	case 0x343E:
		return "Electrolyte cooling check disabled."
	case 0x343F:
		return "Electrolyte back flow temperature check disabled."
	case 0x3440:
		return "Hydrogen outer pressure check disabled."
	case 0x3441:
		return "Chassis circulation fan check disabled."
	case 0x3442:
		return "Electronic compartment cooling fan check disabled."
	case 0x3443:
		return "External switch		Dry contact check disabled."
	case 0x3445:
		return "MCU Temperature Sensor		Control Board MCU temperature check disabled."
	case 0x148A:
		return "Frozen pipes. Electrolyte flow outside pump control limits."
	case 0x1501:
		return "Possible hydrogen leak detected. Pressure readings below nominal values. The device needs to be checked or repaired."
	case 0x350A:
		return "Insufficient pressure drop	Insufficient pressure drop. Check that purge line from the electrolyser is not obstructed."
	case 0x358A:
		return "Pressure > 25 Bar Outer pressure is too high to run blow down routine Please reduce outlet pressure to below 25 bar in order to run the blow down routine."
	case 0x3594:
		return "The Blow down procedure will be started at H2 production start Blow down Routine Active. Please make sure that purge line is properly connected and leads to a safe area."
	case 0x159E:
		return "The purge line is obstructed"
	case 0x360A:
		return "ModBus Heartbeat Packet was not received in time : Lost ModBus safety heartbeat communication : Please check ModBus communication between Electrolyser and controller. Please check if Ethernet cable is properly installed and connection is established."
	case 0x360B:
		return "Gateway	Heartbeat Packet was not received in time : Lost Gateway safety heartbeat communication : Please check communication between Gateway and Electrolyser (UCM). Please check if WiFi connection is stable."
	case 0x360C:
		return "Heartbeat Packet was not received in time : Lost UCM safety heartbeat communication"
	case 0x368A:
		return "Polarization curve cannot be started."
	default:
		return "Unknown Error/Warning"
	}
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
	return decodeDryerMessage(el.status.Dryer.Errors)
}

func (el *ElectrolyserType) GetDryerWarnings() []string {
	return decodeDryerMessage(el.status.Dryer.Warnings)
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
func (el *ElectrolyserType) Start() int {
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
				if debugOutput {
					log.Printf("Electrolyser %s started", el.status.Name)
				}
				return http.StatusOK
			}
		} else {
			return http.StatusConflict
		}
	}
	return http.StatusBadRequest
}

/*
Stop -  Attempt to stop the electrolyser - returns an http.Status. 200 if successful
*/
func (el *ElectrolyserType) Stop() int {
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
				currentSettings.findElByName(el.status.Name).StackTime = el.status.StackTotalRunTime
				if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
					log.Print("Error saving settings - ", err)
				}
				return http.StatusOK
			}
		} else {
			return http.StatusConflict
		}
	}
	return http.StatusBadRequest
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
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// EnableMaintenance puts the electrolyser into maintenance mode
func (el *ElectrolyserType) EnableMaintenance() error {
	if !el.CheckConnected() {
		err := el.Client.WriteRegister(MAINTENANCE, 1)
		if err != nil {
			log.Print("enable maintenance request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		}
		return nil
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
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
}

// RebootDryer will send a reboot command to a connected dryer
func (el *ElectrolyserType) RebootDryer() error {
	if el.CheckConnected() {
		if err := el.Client.WriteRegister(DryerReboot, 1); err != nil {
			log.Print("Reboot Dryer Request failed - ", err, el.buf)
			if err := el.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			el.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("electrolyser is not connected")
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
		if Relays.Relays[el.PowerRelay].On {
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
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Print(err)
		}
	}()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

// scan searches from StartIP until it finds a recognisable electrolyser
func scan(StartIP byte) (net.IP, string, error) {
	var subnetIP net.IP
	if subnet, err := GetOurSubnet(); err != nil {
		log.Println(err)
	} else {
		subnetIP = subnet
	}
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

	if subnet, err := GetOurSubnet(); err != nil {
		log.Println(err)
	} else {
		subnetIP = subnet
	}
	if debugOutput {
		log.Println("Subnet = ", subnetIP.String())
	}

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
				if knownSerial == serial {
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
		model, err := Client.ReadUint32(Model, modbus.INPUT_REGISTER)
		if err != nil {
			log.Println("Not an electrolyser - ", err)
			return "", err
		}
		// Is this an EL21 or an ES40?
		switch model {
		case 0x454C3231:
			return "EL-21", nil
		case 0x45533430:
			return "ES-40", nil
		default:
			return "", fmt.Errorf("not an EL21 or ES40")
		}
	} else {
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
		for {
			select {
			case <-elTicker.C:
				// Drop out if the electrolyser is switched off
				if !el.IsSwitchedOn() {
					return
				}
				// Every minute, check for errors if we have seen less than 5 errors since turning the electrolyser on
				if el.status.State == 0 && numErrors < 5 {
					numLoops++
					if numLoops > 2 {
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
	dryerTicker := time.NewTicker(time.Second) // Check every second
	numErrors := 0
	numSeconds := 0
	for {
		select {
		case <-dryerTicker.C:
			// Drop out if the electrolyser is switched off or does not have a dryer attached
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
