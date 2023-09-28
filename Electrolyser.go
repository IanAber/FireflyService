package main

import (
	"encoding/json"
	"fmt"
	"github.com/simonvetter/modbus"
	"html"
	"log"
	"math"
	"net"
	"strings"
	"sync"
	"time"
)

//const ElIdle = 2
//const ElSteady = 3

// Modbus registers defined at https://handbook.enapter.com/electrolyser/el21_firmware/1.9.3/modbus_tcp_communication_interface.html#references
// Holding Registers
//const TIME = 0
const REBOOT = 4
const LOCATE = 5

const MAINTENANCE = 6
const STARTSTOP = 1000
const RATE = 1002
const BLOWDOWN = 1010
const REFILL = 1011

//const MAINTENANCE = 1013
const PREHEAT = 1014
const BEGINCONFIG = 4000
const COMMITCONFIG = 4001

//const SETIP = 4020
//const SETIPMASK = 4022
//const SETIPGATEWAY = 4024
//const SERIAL = 4026
//const CLOUDLOGGINGENABLE = 4038
//const CLOUDLOGGINGDISABLE = 4042
//const SYSLOGIP = 4044
//const SYSLOGPORT = 4046
//const ALTITUDE = 4142
const MAXPRESSURE = 4308
const RESTARTPRESSURE = 4310

//const STACKSERIAL = 4376
//const DEFAULTRATE = 4396
//const COOLINGTYPE = 4494
//const HEARTBEAT = 4600
//const HEARTBEATGATEWAYTIMEOUT = 4602
//const HEARTBEATUCMTIMEOUT = 4604
//const WARMUPPERIOD = 4900
//const MINSTACKCURRENT = 4910
//const STACKCURRENTCHECKTHRESHOLD = 4912
//const STACKCURRENTPERIOD = 4914
//const MEMBRANEPERIOD = 4916
//const MEMBRANEPRESSURETHRESHOLD = 4926
//const MEMBRANEVOLTAGETHRESHOLD = 4928
//const DRYERSTARTTHRESHOLD = 6014
//const DRYERSTANDBYTHRESHOLD = 6016
const DRYERSTARTSTOP = 6018
const DRYERSTOP = 6019
const DRYERREBOOT = 6020

//const COOLINGVALVE = 7005

// Input registers
const MODEL = 0

//const FIRMWARE = 2                       //	Uint16	Firmware MAJOR and MINOR Version	Ex: 267 => 267 // 256 = 1, 267 % 256 => 11 (1.11)
//const PATCH = 3                          // Uint16	Firmware PATCH Version	Ex: 3 => 3 (3)
//const BUILD = 4                          // Uint32	Firmware Build Number	e.g. 0x4E343471
//const BOARDSERIAL = 6                    //	Uint128	Device Control Board Serial Number	9E25E695-A66A-61DD-6570-50DB4E73652D
const CHASSISSERIAL = 14 //	Uint64	Chassis Serial Number	1 bits - reserved, must be 0 10 bits - Product Unicode, 11 bits - Year + Month, 5 bits - Day, 24 bits - Chassis Number, 5 bits - Order, 8 bits - Site
const SYSTEMSTATE = 18   // Uint16	System State	0 = Internal Error, System not Initialized yet; 1 = System in Operation; 2 = Error; 3 = System in Maintenance Mode; 4 = Fatal Error; 5 = System in Expert Mode.
//const LIVETIME = 20                      // Uint32	Live time [seconds]	Total time during which a system is power up (not only time when stack is working).
//const UPTIME = 22                        // Uint32	Uptime [seconds]	How long the system has been running
//const FREEMEMORY = 26                    // Uint32	Free memory	Memory which can be used
//const AVAILABLEMEMORY = 28               // Uint32	Available memory	Memory which has not been allocated yet
//const CLASHCARDSPACE = 30                // Uint32	Free space on flash-card	Space on flash-card where the configuration is
//const WARNINGS = 768                     // Array of 32 Warning Events	Warning Events Array	Warning Events Array represented by Error Codes. First Uint16 contains total quantity of Warning Events.
//const ERRORS = 832                       // Array of 32 Error Events	Error Events Array	Error Events Array represented by Error Codes. First Uint16 contains total quantity of Error Events.
//const PRODUCTCODE = 1000                 // Uint32	Product Code
//const STACKCYCLES = 1002                 // Uint32	Stack Start/Stop Cycles Quantity	How many Stack Start/Stop cycles
//const STACKRUNTIME = 1004                // Uint32	Stack Total Runtime	seconds
//const STACKTOTALPRODUCTION = 1006        // Float32	Stack Total H2 Production	NL
const FLOWRATE = 1008 // Float32	H2 Flow Rate	NL/hour, NAN when not producing H2;
//const STACKSERIAL = 1010                 // Uint64	Stack Serial Number	1 bits - reserved, must be 0, 15 bits - Stack Type , 11 bits - Year + Month , 5 bits - Day, 24 bits - Stack Number, 8 bits - Site
const STATE = 1200 // Uint16	Electrolyser State	0 = Halted; 1= Maintenance mode; 2 = Idle; 3 = Steady; 4 = Stand-By (Max Pressure); 5 = Curve; 6 = Blowdown.
//const CONFIGINPROGRESS = 4000            // Boolean	Configuration Progress	1 = Configuration is in progress.
//const CONFIGVIAMODBUS = 4001             // Boolean	Configuration Source	1 = Configuration over Modbus.
//const LASTCONFIGRESULT = 4002            // Int32	Last Configuration Result	0 = OK, Configuration was completed successfully; 1 = Permanent, The operation has failed (internal or general error); 2 = No Entry, Configuration was not started or interrupted; 5 = I/O, Data save error; 11 - Try again, Configuration needs to be tried again; 13 = Access Denied, Some changed registers are read-only; 16 = Busy, Another configuration was in progress; 22 = Invalid, The data has invalid or wrong type.
//const LASTCONFIGWRONGHOLDING = 4004      // Uint16	Last Configuration Wrong Holding	Keeps first invalid Holding register number which doesn't allow successful configuration commit.
//const HEATBEATTIMEOUT = 4600             // Uint16	Heartbeat	Timeout for Modbus Heartbeat in seconds. 0 = disabled (default)
const DRYERERROR = 6000 // Uint16	Dryer Error	Dryer error code (bitmask).
//const DRYERWARNING = 6001                // Uint16	Dryer Warning	Dryer warning code (bitmask).
const DRYERTEMP0 = 6002 // Float32	Dryer TT00	Temperature of heater element for cartridge 0 (first line).
//const DRYERTEMP1 = 6004                  // Float32	Dryer TT01	Temperature of heater element for cartridge 1 (second line).
//const DRYERTEMP2 = 6006                  // Float32	Dryer TT02	Temperature of heater element for cartridge 2 (first line).
//const DRYERTEMP3 = 6008                  // Float32	Dryer TT03	Temperature of heater element for cartridge 3 (second line).
//const DRYERINPUTPRESSURE = 6010          // Float32	Dryer PT00	Input pressure of the dryer.
//const DRYEROUTPUTPRESSURE = 6012         // Float32	Dryer PT01	Output pressure of the dryer.
//const WIFIOUI = 6100                     // Uintе32	3 OUI octets for Wi-Fi MAC address	Ex: C8:2B:96
//const WIFIMAC = 6102                     // Uintе32	3 NIC octets for Wi-Fi MAC address	Ex: A8:F5:2C
//const DRYERNETWORKSTATUS = 6104          // Boolean	Dryer Control Network Connection Status	1 = Online; 0 = Offline
const ELECTROLYTEHIGH = 7000 // Boolean	High Electrolyte Level Switch (LSH102B_in)	1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
//const ELECTROLYTEVERYHIGH = 7001         // Boolean	Very High Electrolyte Level Switch (LSHH102A_in)	1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
//const ELECTROLYTELOW = 7002              // Boolean	Low Electrolyte Level Switch (LSL102D_in)	1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
//const ELECTROLYTEMEDIUM = 7003           // Boolean	Medium Electrolyte Level Switch (LSM102C_in)	1 = Electrolyte level over sensor; 0 = Electrolyte level below sensor.
//const ELECTROLYTEPRESSUREHIGH = 7004     // Boolean	Electrolyte Tank High Pressure Switch (PSH102_in)	1 = Pressure is too high; 0 = Pressure is normal.
//const ELECTROLYTEPRESSUREVERYHIGH = 7005 // Boolean	Very High Hydrogen Pressure Switch (PSHH101B_in)	1 = Pressure is too high; 0 = Pressure is normal.
//const DOWNSTREAMHIGHTEMPSWITCH = 7006    // Boolean	Downstream High Temperature Switch (TSH106_in)	1 = Temperature is too high. 0 = Temperature is normal.
//const ELECTRONICSHIGHTEMPSWITCH = 7007   // Boolean	Electronic Compartment High Temperature Switch (TSH108_in)	1 = Temperature is too high. 0 = Temperature is normal.
//const ELECTROLYTETEMPSWITCH = 7008       // Boolean	Very Low Electrolyte Temperature Switch (TSLL102B_in)	1 = Temperature is too low. 0 = Temperature is normal.
//const CHASSISWATERPERSENCESWITCH = 7009  // Boolean	Chassis Water Presence Switch (WPS104_in)	1 = Water is present on input; 0 = No water input.
//const DRYCONTACT = 7010                  // Boolean	Dry Contact	1 = OK (Closed); 0 = NOT OK (Opened)
//const ELECTROLYTECOOLINGFAN = 7500       // Float32	Electrolyte Cooler Fan Speed (F103A_in_rpm)	[rpm]
//const AIRCIRCULATIONSPEED = 7502         // Float32	Air Circulation Fan Speed (F104B_in_rpm)	[rpm]
//const ELECTRONICSCOOLINGSPEED = 7504     // Float32	Electronic Compartment Cooling Fan Speed (F108C_in_rpm)	[rpm]
//const ELECTROLYTEFLOW = 7506             // Float32	Electrolyte Flow Meter (FM106_in_lmin)	[Liters per minute]
const STACKCURRENT = 7508 // Float32	Stack Current (HASS_in_a)	[Ampere]
//const PSUVOLTAGE = 7510                  // Float32	PSU Voltage (Stack Voltage) (PSU_in_v)	[Volt]
//const INNERH2PRESSURE = 7512             // Float32	Inner Hydrogen Pressure (PT101A_in_bar)	[bar]
//const OUTERH2PRESSURE = 7514             // Float32	Outer Hydrogen Pressure (PT101C_in_bar)	[bar]
//const WATERINLETPRESSURE = 7516          // Float32	Water Inlet Pressure (PT105_in_bar)	[bar]
//const ELECTROLYTETEMP = 7518             // Float32	Electrolyte Temperature (TT102A_in_c)	[°C]
//const OWNSTREAMTEMP = 7520               // Float32	Downstream Temperature (TT106_in_c)	[°C]
//const INNERH2RAWPRESSURE = 8000          // Float32	Inner Hydrogen Pressure Raw Sensor Value (PT101A_in_v)	Raw value, [Volt]
//const OUTERH2RAWPRESSURE = 8002          // Float32	Outer Hydrogen Pressure Raw Sensor Value (PT101C_in_v)	Raw value, [Volt]
//const STACKCURRENTRAW = 8004

//const ElStandby = 4

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

type ElectrolyserStatusType struct {
	Device                uint8                  `json:"device"`
	Name                  string                 `json:"name"`
	Powered               bool                   `json:"on"` // Relay is turned on
	Model                 string                 `json:"model"`
	Serial                string                 `json:"serial"`           // 14
	SystemState           uint16                 `json:"systemState"`      // 18
	H2Flow                jsonFloat32            `json:"h2Flow"`           // 1008
	State                 uint16                 `json:"state"`            // 1200
	ElectrolyteLevel      ElectrolyteLevelType   `json:"electrolyteLevel"` // (7000 - 7003 four booleans)
	StackCurrent          jsonFloat32            `json:"stackCurrent"`     // 7508
	StackVoltage          jsonFloat32            `json:"stackVoltage"`     // 7510
	InnerH2Pressure       jsonFloat32            `json:"innerH2"`          // 7512
	OuterH2Pressure       jsonFloat32            `json:"outerH2"`          // 7514
	WaterPressure         jsonFloat32            `json:"waterPressure"`    // 7516
	ElectrolyteTemp       jsonFloat32            `json:"temp"`             // 7518
	CurrentProductionRate jsonFloat32            `json:"rate"`             // H1002
	MaxTankPressure       jsonFloat32            `json:"maxPressure"`      // H4308
	RestartPressure       jsonFloat32            `json:"restartPressure"`  // H4310
	Warnings              ElectrolyserEventsType `json:"warnings"`         // 768
	Errors                ElectrolyserEventsType `json:"errors"`           // 832
	Dryer                 *DryerStatusType       `json:"dryer"`
	IP                    net.IP                 `json:"ip"`
	PowerRelay            uint8                  `json:"powerRelay"`
	//	mu                    sync.Mutex
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
	powerRelay         uint8
	hasDryer           bool
	mu                 sync.Mutex
}

/**
ElectrolysersType defines an array of electrolyses and provide a mutex to control access
*/
type ElectrolysersType struct {
	Arr []*ElectrolyserType
	mu  sync.Mutex
}

/**
FindByName returns a pointer to the electrolyser with the matching name or a nil pointer if not found
*/
func (el *ElectrolysersType) FindByName(name string) *ElectrolyserType {
	for idx := range el.Arr {
		if strings.ToLower(el.Arr[idx].status.Name) == strings.ToLower(name) {
			return el.Arr[idx]
		}
	}
	return nil
}

/**
FindByRelay returns a pointer to the electrolyser with the assigned relay that matches that given or a nil pointer if not found
*/
func (el *ElectrolysersType) FindByRelay(relay uint8) *ElectrolyserType {
	for idx := range el.Arr {
		if el.Arr[idx].powerRelay == relay {
			return el.Arr[idx]
		}
	}
	return nil
}

func (el *ElectrolyserType) getJsonStatus() ([]byte, error) {
	return json.Marshal(el.getStatus())
}

func (el *ElectrolyserType) setClient(IP net.IP) {
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

	//if el.Client != nil {
	//	if err := el.Client.Close(); err != nil {
	//		log.Print(err)
	//	}
	//}
	el.status.IP = IP
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
					el.rescan(0, elConfig.Serial)
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

func (e *ElectrolyserType) GetRate() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	r := int(e.status.CurrentProductionRate)
	if (e.OffRequested != nil) && (r == 60) {
		return 0
	} else {
		return r
	}
}

// ReadSerialNumber reads and decodes the serial number
func (e *ElectrolyserType) ReadSerialNumber(IP ...net.IP) string {
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
		e.setClient(IP[0])
	}

	if !e.CheckConnected() {
		if debugOutput {
			log.Println("Not connected")
		}
		return ""
	}
	if debugOutput {
		log.Println("Reading the serial number")
	}
	serialCode, err := e.Client.ReadUint64(CHASSISSERIAL, modbus.INPUT_REGISTER)
	if err != nil {
		log.Println("Error getting serial number - ", err)
		return ""
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

	return fmt.Sprintf("%s%02d%02d%02d%02d%s%s", codes.Product, codes.Year, codes.Month, codes.Day, codes.Chassis, codes.Order, codes.Site)
}

// AA 21 06 4 %!s(uint32=1) C%!(EXTRA string=PI)

func (e *ElectrolyserType) IsSwitchedOn() bool {
	e.status.Powered = Relays.GetRelay(e.powerRelay)
	return e.status.Powered
}

func (e *ElectrolyserType) CheckConnected() bool {
	if e.Client == nil {
		if debugOutput {
			log.Println("No client")
		}
		return false
	}
	if !e.IsSwitchedOn() {
		if debugOutput {
			log.Println("Client Switched Off")
		}
		return false
	}
	if !e.clientConnected {
		if time.Since(e.lastConnectAttempt) > time.Second*5 {
			if err := e.Client.Open(); err != nil {
				log.Print("Modbus client.open error - ", err)
			} else {
				e.clientConnected = true
				if debugOutput {
					log.Println("Connected...")
				}
			}
			e.lastConnectAttempt = time.Now()
		} else {
			if debugOutput {
				log.Printf("Time since last connection attempt = %v", time.Since(e.lastConnectAttempt))
			}
		}
	}
	return e.clientConnected
}

func (e *ElectrolyserType) readEvents() {
	if !e.CheckConnected() {
		return
	}
	events, err := e.Client.ReadRegisters(768, 32, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("Modbus read register error - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}

	e.status.Warnings.count = events[0]
	copy(e.status.Warnings.codes[:], events[1:])
	events, err = e.Client.ReadRegisters(832, 32, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("Modbus read register error - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}
	e.status.Errors.count = events[0]
	copy(e.status.Errors.codes[:], events[1:])
}

func (e *ElectrolyserType) ReadModelNumber() {
	modelNumber, err := e.Client.ReadUint32(0, modbus.INPUT_REGISTER)
	if err != nil {
		if debugOutput {
			log.Println("No active device found at - "+e.status.IP.String(), err)
		}
		e.status.Model = ""
		return
	}
	// Is this an EL21?
	switch modelNumber {
	case 0x454C3231:
		e.status.Model = "EL-21"
	case 0x45533430:
		e.status.Model = "ES-40"
	default:
		e.status.Model = ""
		if debugOutput {
			log.Println("not an EL21 or ES40")
		}
	}
	return
}

func (e *ElectrolyserType) ReadValues() {
	if e.Client == nil {
		if debugOutput {
			log.Println("Adding a modbus client")
		}
		e.setClient(e.status.IP)
	}
	if !e.CheckConnected() {
		if debugOutput {
			log.Println("Electrolyser " + e.status.Name + " is not connected")
		}
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	// Get the model if we do not already have it
	if e.status.Model == "" {
		e.ReadModelNumber()
	}
	// Get the serial number if we do not already have it
	if e.status.Serial == "" {
		e.status.Serial = e.ReadSerialNumber()
	}

	// Get the stack current and voltage, innerH2 pressure, outerH2 pressure, water pressure and electrolyte temperature.
	//	log.Println("STACKCURRENT")
	values, err := e.Client.ReadFloat32s(STACKCURRENT, 6, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("Modbus reading float32 values - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}

	e.status.StackCurrent = jsonFloat32(values[0])
	e.status.StackVoltage = jsonFloat32(values[1])
	e.status.InnerH2Pressure = jsonFloat32(values[2])
	e.status.OuterH2Pressure = jsonFloat32(values[3])
	e.status.WaterPressure = jsonFloat32(values[4])
	e.status.ElectrolyteTemp = jsonFloat32(values[5])

	//	get the maximum tank and restart pressure settings if we don't have them
	if e.status.MaxTankPressure == 0 || e.status.RestartPressure == 0 {
		p, err := e.Client.ReadFloat32s(MAXPRESSURE, 2, modbus.HOLDING_REGISTER)
		if err != nil {
			log.Print("Modbus reading max tank and restart pressure - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return
		}
		e.status.MaxTankPressure = jsonFloat32(p[0])
		e.status.RestartPressure = jsonFloat32(p[1])
	}

	e.status.SystemState, err = e.Client.ReadRegister(SYSTEMSTATE, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("System state error - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}
	//	log.Println("FLOWRATE")
	//	log.Println("Reading 1008(2)...")
	h2f, err := e.Client.ReadFloat32(FLOWRATE, modbus.INPUT_REGISTER)
	e.status.H2Flow = jsonFloat32(h2f)
	if err != nil {
		log.Print("H2Flow error - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}
	// Flow will return NaN if the electrolyser is not producing.
	if math.IsNaN(float64(e.status.H2Flow)) {
		e.status.H2Flow = 0
	}
	//	log.Println("STATE")
	//	log.Println("Reading 1208(2)...")
	e.status.State, err = e.Client.ReadRegister(STATE, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("ElState - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}
	//	log.Println("ELECTROLYTE LEVEL")
	//	log.Println("Reading 7008(2)...")
	level, err := e.Client.ReadRegisters(ELECTROLYTEHIGH, 4, modbus.INPUT_REGISTER)
	if err != nil {
		log.Print("Electrolyte Level - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}
	switch {
	case level[2] == 0:
		e.status.ElectrolyteLevel = empty
	case level[3] == 0:
		e.status.ElectrolyteLevel = low
	case level[0] == 0:
		e.status.ElectrolyteLevel = medium
	case level[1] == 0:
		e.status.ElectrolyteLevel = high
	default:
		e.status.ElectrolyteLevel = veryHigh
	}
	//	log.Println("RATE")
	rate, err := e.Client.ReadFloat32(RATE, modbus.HOLDING_REGISTER)
	//	log.Println("Current rate = ", rate)
	e.status.CurrentProductionRate = jsonFloat32(rate)
	if err != nil {
		log.Print("Current Production error - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return
	}
	e.readEvents()

	// log.Println("Reading DRYERTEMP (6 registers)...")
	if e.hasDryer {
		dryer, err := e.Client.ReadFloat32s(DRYERTEMP0, 6, modbus.INPUT_REGISTER)
		if err != nil {
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			log.Println("Dryer error", err)
			return
		}
		if e.status.Dryer == nil {
			e.status.Dryer = new(DryerStatusType)
		}

		if e.status.Dryer != nil {
			e.status.Dryer.Temps[0] = jsonFloat32(dryer[0])
			e.status.Dryer.Temps[1] = jsonFloat32(dryer[1])
			e.status.Dryer.Temps[2] = jsonFloat32(dryer[2])
			e.status.Dryer.Temps[3] = jsonFloat32(dryer[3])

			e.status.Dryer.InputPressure = jsonFloat32(dryer[4])
			e.status.Dryer.OutputPressure = jsonFloat32(dryer[5])

			dryerErrors, err := e.Client.ReadRegisters(DRYERERROR, 2, modbus.INPUT_REGISTER)
			if err != nil {
				log.Print("Error reading dryer errors - ", err)
				if err := e.Client.Close(); err != nil {
					log.Print("Error closing modbus client - ", err)
				}
				e.clientConnected = false
				return
			}
			e.status.Dryer.Errors = dryerErrors[0]
			e.status.Dryer.Warnings = dryerErrors[1]
		}
	}
	//	log.Printf("Electrolyser %s - Serial # %s - IP %s", e.status.Name, e.status.Serial, e.status.IP.String())
}

func (e *ElectrolyserType) getStatus() *ElectrolyserStatusType {
	e.mu.Lock()
	defer e.mu.Unlock()

	status := new(ElectrolyserStatusType)

	*status = e.status
	return status
}

func (e *ElectrolyserType) GetSystemState() string {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch e.status.SystemState {
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

func (e *ElectrolyserType) GetSerial() string {

	e.mu.Lock()
	defer e.mu.Unlock()

	return e.status.Serial
}

func (e *ElectrolyserType) GetIPString() string {

	e.mu.Lock()
	defer e.mu.Unlock()

	return e.status.IP.String()
}

func (e *ElectrolyserType) GetWarnings() []string {
	var s []string

	e.mu.Lock()
	defer e.mu.Unlock()

	for w := uint16(0); w < e.status.Warnings.count; w++ {
		s = append(s, decodeMessage(e.status.Warnings.codes[w]))
	}
	return s
}

func (e *ElectrolyserType) GetErrors() []string {
	var s []string

	e.mu.Lock()
	defer e.mu.Unlock()

	for err := uint16(0); err < e.status.Errors.count; err++ {
		s = append(s, decodeMessage(e.status.Errors.codes[err]))
	}
	return s
}

func (e *ElectrolyserType) getState() string {

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.status.Powered {
		switch e.status.State {
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

func (e *ElectrolyserType) GetDryerErrors() []string {
	return decodeDryerMessage(e.status.Dryer.Errors)
}

func (e *ElectrolyserType) GetDryerWarnings() []string {
	return decodeDryerMessage(e.status.Dryer.Warnings)
}

func (e *ElectrolyserType) SendRateToElectrolyser(rate float32) error {
	err := e.Client.WriteFloat32(RATE, rate)
	if err != nil {
		log.Print("Error setting production rate - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
	}
	return err
}

// SetProduction sets the electrolyser to the rate given 0, 60..100
func (e *ElectrolyserType) SetProduction(rate uint8) {
	if debugOutput {
		log.Printf("Set electrolyser %s to %d", e.status.Name, rate)
	}

	if !e.CheckConnected() {
		return
	}
	if rate < 60 || rate > 100 {
		log.Printf("Invalid rate (%d) requested", rate)
		return
	}
	// 60% or more we should send the rate and clear the off timer
	if err := e.SendRateToElectrolyser(float32(rate)); err != nil {
		log.Println(err)
	}
	// If there is a pending delayed stop then kill the timer
	if e.OffRequested != nil {
		e.OffRequested.Stop()
		e.OffRequested = nil
	}
	// If the electrolyser is in Idle then start it.
	//if e.status.State == ElIdle {
	//	log.Println("Electrolyser is idle so sending a start command.")
	//	e.Start(false)
	//}
}

func (e *ElectrolyserType) SetRestartPressure(pressure float32) error {
	if !e.CheckConnected() {
		return fmt.Errorf("electrolyser is not turned on")
	}

	// Check configuration status
	status, err := e.Client.ReadRegister(BEGINCONFIG, modbus.INPUT_REGISTER)
	if err != nil {
		log.Println("Cannot establish configuration status - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return fmt.Errorf("unable to set the restart pressure for the electrolyser. See the log file for more detail")
	}
	if status != 0 {
		if debugOutput {
			log.Println("configuration is already in progress")
		}
		return fmt.Errorf("configuration is already in progress")
	}

	//Begin configuration
	err = e.Client.WriteRegister(BEGINCONFIG, 1)
	if err != nil {
		log.Println("Cannot start configuration - ", err)
		return fmt.Errorf("start configuration failed")
	}
	status, err = e.Client.ReadRegister(BEGINCONFIG, modbus.INPUT_REGISTER)
	if err != nil {
		log.Println("Cannot establish configuration status after configuration start - ", err)
		return fmt.Errorf("unable to set the restart pressure for the electrolyser. See the log file for more detail")
	}
	if status == 0 {
		log.Println("Configuration did not start.")
		return fmt.Errorf("configuration failed to start")
	}

	err = e.Client.WriteFloat32(RESTARTPRESSURE, pressure)
	if err != nil {
		log.Print("Error setting electrolyser restart pressure - ", err)
		if err := e.Client.Close(); err != nil {
			log.Print("Error closing modbus client - ", err)
		}
		e.clientConnected = false
		return fmt.Errorf("unable to set the restart pressure for the electrolyser. See the log file for more detail")
	}

	err = e.Client.WriteRegister(COMMITCONFIG, 1)
	if err != nil {
		log.Println("Commit configuration changes failed - ", err)
		return fmt.Errorf("unable to commit the restart pressure change for the electrolyser. See the log file for more detail")
	}
	// Force a reread of the pressure the next time the values are read from the electrolyser
	e.mu.Lock()
	defer e.mu.Unlock()

	e.status.RestartPressure = 0

	return nil
}

/**
Start will Attempt to start the electrolyser - return true if successful
*/
func (e *ElectrolyserType) Start() bool {
	if e.CheckConnected() {
		if err := e.Client.WriteRegister(STARTSTOP, 1); err != nil {
			log.Print("Error starting Electrolyser - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
		} else {
			if debugOutput {
				log.Printf("Electrolyser %s started", e.status.Name)
			}
			return true
		}
	}
	return false
}

/*
Stop -  Attempt to stop the electrolyser - return true if successful
*/
func (e *ElectrolyserType) Stop() bool {
	if e.CheckConnected() {
		// Send the stop command
		if err := e.Client.WriteRegister(STARTSTOP, 0); err != nil {
			log.Print("Error stopping electrolyser - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
		} else {
			return true
		}
	}
	return false
}

/**
Preheat will start the preheat cycle
*/
func (e *ElectrolyserType) Preheat() error {
	if e.CheckConnected() {
		if e.status.ElectrolyteTemp < 26 {
			err := e.Client.WriteRegister(PREHEAT, 1)
			if err != nil {
				log.Print("Preheat Request failed - ", err)
				if err := e.Client.Close(); err != nil {
					log.Print("Error closing modbus client - ", err)
				}
				e.clientConnected = false
				return err
			}
		} else {
			return fmt.Errorf("Preheat request ignored as temperature is already %f C", e.status.ElectrolyteTemp)
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
Reboot attempts to reboot the electrolyser
*/
func (e *ElectrolyserType) Reboot() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(REBOOT, 1)
		if err != nil {
			log.Print("Reboot Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
Locate starts the locate cycle which flashes the LEDs on the fron panel
*/
func (e *ElectrolyserType) Locate() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(LOCATE, 1)
		if err != nil {
			log.Print("Locate Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
EnableMaintenance puts the electrolyser into maintenance mode
*/
func (e *ElectrolyserType) EnableMaintenance() error {
	if !e.CheckConnected() {
		err := e.Client.WriteRegister(MAINTENANCE, 1)
		if err != nil {
			log.Print("Enable Maintenance Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
DisbleMaintenance stops the maintenanace cycle
*/
func (e *ElectrolyserType) DisableMaintenance() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(MAINTENANCE, 0)
		if err != nil {
			log.Print("Disable Maintenance Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
BlowDown starts the blowdown process
*/
func (e *ElectrolyserType) BlowDown() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(BLOWDOWN, 1)
		if err != nil {
			log.Print("Blow down Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
Refill starts the refill process
*/
func (e *ElectrolyserType) Refill() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(REFILL, 1)
		if err != nil {
			log.Print("Refill Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
StartDryer will start a connected dryer
*/
func (e *ElectrolyserType) StartDryer() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(DRYERSTARTSTOP, 1)
		if err != nil {
			log.Print("Start Dryer Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
StopDryer will stop a connected dryer
*/
func (e *ElectrolyserType) StopDryer() error {
	if e.CheckConnected() {
		err := e.Client.WriteRegister(DRYERSTOP, 1)
		if err != nil {
			log.Print("Stop Dryer Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
RebootDryer will send a reboot command to a connted dryer
*/
func (e *ElectrolyserType) RebootDryer() error {
	if e.CheckConnected() {
		if err := e.Client.WriteRegister(DRYERREBOOT, 1); err != nil {
			log.Print("Reboot Dryer Request failed - ", err)
			if err := e.Client.Close(); err != nil {
				log.Print("Error closing modbus client - ", err)
			}
			e.clientConnected = false
			return err
		}
		return nil
	}
	return fmt.Errorf("Electrolyser is not connected")
}

/**
GetDryerErrorsHTML returns any errors from the connected dryer in an HTML table
*/
func (e *ElectrolyserType) GetDryerErrorsHTML() string {
	var htmlString = "<table>"

	for _, err := range e.GetDryerErrors() {
		htmlString += "<tr><td>" + html.EscapeString(err) + "</td></tr>"
	}
	htmlString += "</table>"
	return htmlString
}

/**
GetDryerErrorText returns any errors from the connected dryer
*/
func (e *ElectrolyserType) GetDryerErrorText() string {
	var s = ""

	for _, err := range e.GetDryerErrors() {
		if s != "" {
			s += "\n"
		}
		s += err
	}
	return s
}

/**
GetDryerWarningsHTML returns any errors from the connected dryer in an  HTML table
*/
func (e *ElectrolyserType) GetDryerWarningsHTML() string {
	var htmlString = "<table>"

	for _, warning := range e.GetDryerWarnings() {
		htmlString += "<tr><td>" + html.EscapeString(warning) + "</td></tr>"
	}
	htmlString += "</table>"
	return htmlString
}

/**
GetDryerWarningText returns any errors from the connected dryer
*/
func (e *ElectrolyserType) GetDryerWarningText() string {
	var s = ""

	for _, warning := range e.GetDryerWarnings() {
		if s != "" {
			s += "\n"
		}
		s += warning
	}
	return s
}

/**
Acquire searches for an electrolyser
Search is conducted in the current subnet.
*/
func (e *ElectrolyserType) Acquire() error {
	// Make sure all electrolysers are powered down.
	for _, el := range currentSettings.Electrolysers {
		if Relays.Relays[el.PowerRelay].On {
			return fmt.Errorf("All electrolysers must be turned off before performing a search")
		}
	}
	if debugOutput {
		log.Print("Turning on the electrolyser")
	}
	// Turn on the relay and give it 15 seconds to come on line.
	Relays.SetRelay(e.powerRelay, true)
	defer Relays.SetRelay(e.powerRelay, false)

	time.Sleep(time.Second * 15)

	// Try and find the electrolyser
	if debugOutput {
		log.Print("Searching...")
	}
	if e.Client != nil {
		//		e.Client.Close()
		e.Client = nil
		if debugOutput {
			log.Println("Client closed")
		}
	}
	ip, elType, err := scan(1)

	if err == nil {
		e.mu.Lock()
		e.setClient(ip)
		e.status.Model = elType
		e.mu.Unlock()

		if debugOutput {
			log.Println("Read the current values")
		}
		e.status.Serial = ""
		e.ReadValues()

		return nil
	} else {
		return err
	}
}

/**
GetOurIP returns the main IP address for this computer
*/
func GetOurSubnet() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP, nil
}

/**
scan searches from StartIP until it finds a recognisable electrolyser
*/
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
	return net.IPv4zero, "Not Found", fmt.Errorf("No electrolyser found")
}

/**
rescan searches from StartIP until it finds a recognisable electrolyser with the same serial number
*/
func (e *ElectrolyserType) rescan(StartIP byte, knownSerial string) (net.IP, string, error) {
	var subnetIP net.IP
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
				serial := e.ReadSerialNumber(IP)
				if debugOutput {
					log.Println("Looking for ", knownSerial)
					log.Println("Found       ", serial)
				}
				if knownSerial == serial {
					// Looks like we have our device so we should update the IP address.
					if debugOutput {
						log.Println("Electrolyser", elType, "found at ", IP)
					}
					return IP, elType, nil
				}
			} else {
				if debugOutput {
					log.Println(err)
				}
			}
		}
	}
	return net.IPv4zero, "Not Found", fmt.Errorf("No electrolyser found")
}

/**
tryConnect attempts to connect to the IP on the given port. err is nil if it succeeds
*/
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

/**
CheckForElectrolyser tests for an electrolyser.
Given that we can connect on the Modbus port, test to see if this looks like an electrolyser by
checking for the serial number.
*/
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
		model, err := Client.ReadUint32(MODEL, modbus.INPUT_REGISTER)
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
