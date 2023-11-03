package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go.einride.tech/can"
	"log"
	"net/http"
	"os"
	"time"
	//	"log"
	"sync"
)

const FuelCellDCDCVoltagesBySecond = `select UNIX_TIMESTAMP(logged) as logged
                       ,(DCDCOutVolts / 10) as voltage
                       ,(DCDCOutAmps / 100) as current
					   ,((DCDCOutVolts * DCDCOutAmps) / 1000) as power
					   ,(PowerRequested * 100) as requestedPower
					   ,(MaxBattVolts / 10) as MaxBattVolts
					   ,(MinBattVolts / 10) as MinBattVolts
                   from firefly.PANFuelCell
                  where logged between ? and ?`

const FuelCellDCDCVoltagesByMinute = `select min(UNIX_TIMESTAMP(logged)) as logged
                       ,(avg(DCDCOutVolts) / 10) as voltage
                       ,(avg(DCDCOutAmps) / 100) as current
					   ,(avg(DCDCOutVolts * DCDCOutAmps) / 1000) as power
					   ,(avg(PowerRequested) * 100) as requestedPower
					   ,(avg(MaxBattVolts) / 10) as MaxBattVolts
					   ,(avg(MinBattVolts) / 10) as MinBattVolts
                   from firefly.PANFuelCell
                  where logged between ? and ?
	              group by UNIX_TIMESTAMP(logged) div 60`

const FuelCellStackVoltagesBySecond = `select UNIX_TIMESTAMP(logged) as logged
						, (StackVoltage / 10) as stackVolts
						, (StackCurrent / 10) as stackCurrent
					    , Cell00Volts as cell00
					    , Cell01Volts as cell01
					    , Cell02Volts as cell02
					    , Cell03Volts as cell03
					    , Cell04Volts as cell04
					    , Cell05Volts as cell05
					    , Cell06Volts as cell06
					    , Cell07Volts as cell07
					    , Cell08Volts as cell08
					    , Cell09Volts as cell09
					    , Cell10Volts as cell10
					    , Cell11Volts as cell11
					    , Cell12Volts as cell12
					    , Cell13Volts as cell13
					    , Cell14Volts as cell14
					    , Cell15Volts as cell15
					    , Cell16Volts as cell16
					    , Cell17Volts as cell17
					    , Cell18Volts as cell18
					    , Cell19Volts as cell19
					    , Cell20Volts as cell20
					    , Cell21Volts as cell21
					    , Cell22Volts as cell22
					    , Cell23Volts as cell23
					    , Cell24Volts as cell24
					    , Cell25Volts as cell25
					    , Cell26Volts as cell26
					    , Cell27Volts as cell27
					    , Cell28Volts as cell28
					    , Cell29Volts as cell29
					    , Cell30Volts as cell30
                   from firefly.PANFuelCell
                  where logged between ? and ?`

const FuelCellStackVoltagesByMinute = `select min(UNIX_TIMESTAMP(logged)) as logged
						, avg(StackVoltage) / 10 as stackVolts
						, avg(StackCurrent) / 10 as stackCurrent
					    , avg(Cell00Volts) as cell00
					    , avg(Cell01Volts) as cell01
					    , avg(Cell02Volts) as cell02
					    , avg(Cell03Volts) as cell03
					    , avg(Cell04Volts) as cell04
					    , avg(Cell05Volts) as cell05
					    , avg(Cell06Volts) as cell06
					    , avg(Cell07Volts) as cell07
					    , avg(Cell08Volts) as cell08
					    , avg(Cell09Volts) as cell09
					    , avg(Cell10Volts) as cell10
					    , avg(Cell11Volts) as cell11
					    , avg(Cell12Volts) as cell12
					    , avg(Cell13Volts) as cell13
					    , avg(Cell14Volts) as cell14
					    , avg(Cell15Volts) as cell15
					    , avg(Cell16Volts) as cell16
					    , avg(Cell17Volts) as cell17
					    , avg(Cell18Volts) as cell18
					    , avg(Cell19Volts) as cell19
					    , avg(Cell20Volts) as cell20
					    , avg(Cell21Volts) as cell21
					    , avg(Cell22Volts) as cell22
					    , avg(Cell23Volts) as cell23
					    , avg(Cell24Volts) as cell24
					    , avg(Cell25Volts) as cell25
					    , avg(Cell26Volts) as cell26
					    , avg(Cell27Volts) as cell27
					    , avg(Cell28Volts) as cell28
					    , avg(Cell29Volts) as cell29
					    , avg(Cell30Volts) as cell30
                   from firefly.PANFuelCell
                  where logged between ? and ?
	              group by UNIX_TIMESTAMP(logged) div 60`

const FuelCellPressuresBySecond = `select UNIX_TIMESTAMP(logged) as logged
                       ,(PANFuelCell.HydrogenPressure / 10) as hydrogen
                       ,(PANFuelCell.AirPressure / 100) as air
                   from firefly.PANFuelCell
                  where logged between ? and ?`

const FuelCellPressuresByMinute = `select min(UNIX_TIMESTAMP(logged)) as logged
                       ,(avg(PANFuelCell.HydrogenPressure) / 10) as hydrogen
                       ,(avg(PANFuelCell.AirPressure) / 100) as air
                   from firefly.PANFuelCell
                  where logged between ? and ?
	              group by UNIX_TIMESTAMP(logged) div 60`

const FuelCellCoolantBySecond = `select UNIX_TIMESTAMP(PANFuelCell.logged) as logged
                       ,(PANFuelCell.CoolantInTemp / 10) - 40 as inTemp
                       ,(PANFuelCell.CoolantOutTemp / 10) - 40 as outTemp
					   ,(PANFuelCell.CoolantFanSpeed) as fanSpeed
					   ,(PANFuelCell.CoolantPumpSpeed) as pumpSpeed
					   ,(PANFuelCell.CoolantPumpAmps) as pumpAmps
					   ,(PANFuelCell.CoolantPumpVolts) as pumpVolts
                   from firefly.PANFuelCell
                  where logged between ? and ?`

const FuelCellCoolantByMinute = `select min(UNIX_TIMESTAMP(logged)) as logged
                       ,(avg(PANFuelCell.CoolantInTemp / 10) - 40) as inTemp
                       ,(avg(PANFuelCell.CoolantOutTemp / 10) - 40) as outTemp
					   ,(avg(PANFuelCell.CoolantFanSpeed)) as fanSpeed
					   ,(avg(PANFuelCell.CoolantPumpSpeed)) as pumpSpeed
					   ,(avg(PANFuelCell.CoolantPumpAmps)) as pumpAmps
					   ,(avg(PANFuelCell.CoolantPumpVolts)) as pumpVolts
                   from firefly.PANFuelCell
                  where logged between ? and ?
	              group by UNIX_TIMESTAMP(logged) div 60`

const NUMCOMMANDREPEATS = 10 // Send Start/Stop 10 times

type PanSettingsType struct {
	TargetPower       float64 // Power output requested
	TargetBatteryHigh float64 // High voltage target value
	TargetBatteryLow  float64 // Low voltage target value
	FuelCellOn        bool    // Flag to tell the unit to turn on
	Exhaust           bool    // Flag to indicate that an Exhaust command is requested
	PumpActive        bool    // Flag to show if the water pump is running
	//	PumpTimer         *time.Timer // The timer to detect that no water pump messages have been received
}

var heartbeat uint16
var returnedHeartbeat uint16

type RunCommandType byte

const (
	FCNoCommand RunCommandType = iota
	FCStartUp
	FCShutDown
)

type ExhaustModeType byte

const (
	ExhaustClosed = iota
	ExhaustOpen
)

const CanOutputControlMsg = 0x161088C1

type OutputControlType struct {
	FuelCellRunEnable RunCommandType  // Startup / Shutdown
	PowerDemand       uint8           // kW x 10
	ExhaustMode       ExhaustModeType // Open / Closed
	repeats           uint8
}

func (t *OutputControlType) GetPowerDemand() float64 {
	return float64(t.PowerDemand) / 10.0
}

func (t *OutputControlType) UpdateFuelCell(bus *CANBus) error {
	var frame can.Frame

	frame.ID = CanOutputControlMsg
	frame.IsExtended = true
	frame.Length = 8
	if t.repeats > 0 {
		frame.Data[0] = byte(t.FuelCellRunEnable)
	} else {
		frame.Data[0] = 0
	}
	frame.Data[1] = t.PowerDemand
	frame.Data[2] = byte(t.ExhaustMode)
	t.repeats--
	return bus.Publish(frame)
}

const CanBatteryVoltageLimitsMsg = 0x161088C2

type BatteryVoltageLimitsType struct {
	BMSHighVoltage uint16 //Battery high voltage setpoint
	BMSLowVoltage  uint16 //Battery low voltage setpoint
	IsoFlag        bool   //Set true to suppress stack isolation tests.
	ClearFaults    bool   //Set true to send 10 frames with the fault clear flag set
	repeats        uint8  // Number of times to send the same frame
}

// UpdateFuelCell sends the frame to the CAN bus
func (t *BatteryVoltageLimitsType) UpdateFuelCell(bus *CANBus) error {
	var frame can.Frame

	frame.ID = CanBatteryVoltageLimitsMsg
	frame.IsExtended = true
	frame.Length = 8
	binary.LittleEndian.PutUint16(frame.Data[0:2], t.BMSHighVoltage)
	binary.LittleEndian.PutUint16(frame.Data[2:4], t.BMSLowVoltage)
	if t.IsoFlag {
		frame.Data[4] = 1
	} else {
		frame.Data[4] = 0
	}
	if t.ClearFaults {
		frame.Data[5] = 2
	} else {
		frame.Data[5] = 0
	}
	if t.repeats > 0 {
		t.repeats--
	} else {
		t.ClearFaults = false
	}
	return bus.Publish(frame)
}

type PowerModeStateType byte

//goland:noinspection GoUnusedConst
const (
	PMOff = iota
	PMInit
	PMH2Purge
	PMStartup
	PMAirPurge
	PMH2LeakCheck
	PMManual
	PMEmergencyShut
	PMFault
	PM_Shutdown
)

func (pm PowerModeStateType) String() string {
	modeStates := [...]string{"Off", "Standby", "Hydrogen intake", "Start", "AirPurge", "Hydrogen leak check", "manual", "emergency stop", "fault", "shutdown"}
	return modeStates[pm]
}

const CanPowerModeMsg = 0x161088A1

type PowerModeType struct {
	PowerModeState PowerModeStateType
	FaultLevel     byte
	FaultCode      uint16
	RunStage       byte
}

func (t *PowerModeType) Load(data [8]byte) {
	t.PowerModeState = PowerModeStateType(data[0])
	t.FaultLevel = data[1]
	t.FaultCode = binary.LittleEndian.Uint16(data[2:4])
	t.RunStage = data[4]
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.PowerModeState = data[0]
	dbRecord.RunStage = data[4]
	dbRecord.FaultLevel = data[1]
}

const CanPressuresMsg = 0x161088A2

type PressuresType struct {
	H2Pressure        uint16 // Hydrogen pressure
	AirPressure       uint16 // Air pressure
	CoolantPressure   uint16 // Coolant pressure
	H2AirPressureDiff uint16 // Hydrogen air pressure difference
}

func (t *PressuresType) Load(data [8]byte) {
	t.H2Pressure = binary.LittleEndian.Uint16(data[0:2])
	t.AirPressure = binary.LittleEndian.Uint16(data[2:4])
	t.CoolantPressure = binary.LittleEndian.Uint16(data[4:6])
	t.H2AirPressureDiff = binary.LittleEndian.Uint16(data[6:8])
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.HydrogenPressure = t.H2Pressure
	dbRecord.AirPressure = t.AirPressure
	dbRecord.CoolantPressure = t.CoolantPressure
}

func (t *PressuresType) GetH2Pressure() float64 {
	return float64(t.H2Pressure) / 10.0
}

func (t *PressuresType) GetAirPressure() float64 {
	return float64(t.AirPressure) / 10.0
}

func (t *PressuresType) GetCoolantPressure() float64 {
	return float64(t.CoolantPressure) / 10.0
}

func (t *PressuresType) GetH2AirPressureDiff() float64 {
	return float64(t.H2AirPressureDiff) / 10.0
}

const CanStackCoolantMsg = 0x161088A3

type StackCoolantType struct {
	CoolantInTemp  uint16 // Coolant temperature at the inlet of the stack
	CoolantOutTemp uint16 // Coolant temperature at the outlet of the stack
	AirTemp        uint16 // Air temperature at the inlet of the stack
	AmbientTemp    uint16 // Ambient temperature
}

func (t *StackCoolantType) Load(data [8]byte) {
	t.CoolantInTemp = binary.LittleEndian.Uint16(data[0:2])
	t.CoolantOutTemp = binary.LittleEndian.Uint16(data[2:4])
	t.AirTemp = binary.LittleEndian.Uint16(data[4:6])
	t.AmbientTemp = binary.LittleEndian.Uint16(data[6:8])
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.CoolantInlTemp = t.CoolantInTemp
	dbRecord.CoolantOutTemp = t.CoolantOutTemp
	dbRecord.AirinletTemp = t.AirTemp
	dbRecord.AmbientTemp = t.AmbientTemp
}

func (t *StackCoolantType) GetCoolantInTemp() float64 {
	return (float64(t.CoolantInTemp) / 10.0) - 40
}

func (t *StackCoolantType) GetCoolantOutTemp() float64 {
	return (float64(t.CoolantOutTemp) / 10.0) - 40
}

func (t *StackCoolantType) GetAirTemp() float64 {
	return (float64(t.AirTemp) / 10.0) - 40
}

func (t *StackCoolantType) GetAmbientTemp() float64 {
	return (float64(t.AmbientTemp) / 10.0) - 40
}

const CanAirFlowMsg = 0x161088A4

type AirFlowType struct {
	Flow uint16 //Air flow Lpm * 10
}

func (t *AirFlowType) Load(data [8]byte) {
	t.Flow = binary.LittleEndian.Uint16(data[4:6])
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.AirFlow = t.Flow
}

func (t *AirFlowType) GetFlow() float64 {
	return float64(t.Flow) / 10.0
}

const CanAlarmsMsg = 0x161088A5

type AlarmsType struct {
	bitMap uint32
}

func (al *AlarmsType) Load(data [8]byte) {
	al.bitMap = binary.LittleEndian.Uint32(data[0:4])
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.Alarms = al.bitMap
}

const AlarmVoltageLow = 0b00000000000000000000000000000001
const AlarmH2concentration = 0b00000000000000000000000000000010
const AlarmCoolantTempOutDiff = 0b00000000000000000000000000000100
const AlarmCoolantTempHigh = 0b00000000000000000000000000001000
const AlarmWaterPumpFault = 0b00000000000000000000000000010000
const AlarmH2CirculatingPumpFault = 0b00000000000000000000000000100000
const AlarmH2PressureHigh = 0b00000000000000000000000001000000
const AlarmH2PressureSensorFault = 0b00000000000000000000000010000000
const AlarmDcdcCommunicationFault = 0b00000000000000000000000100000000
const AlarmDcdcFault = 0b00000000000000000000001000000000
const AlarmPtcFault = 0b00000000000000000000010000000000
const AlarmH2TankTemp = 0b00000000000000000000100000000000
const AlarmH2TankHighPressure = 0b00000000000000000001000000000000
const AlarmH2TankMidPressure = 0b00000000000000000010000000000000
const AlarmH2TankLowPressure = 0b00000000000000000100000000000000
const AlarmFcuToVcuFault = 0b00000000000000001000000000000000
const AlarmTempSensorFault = 0b00000000000000010000000000000000
const AlarmH2SPCheckFault = 0b00000000000000100000000000000000
const AlarmH2SOCLow = 0b00000000000001000000000000000000
const AlarmH2OutPressureLow = 0b00000000000010000000000000000000
const AlarmAirPressureLow = 0b00000000000100000000000000000000
const AlarmAirPressureHigh = 0b00000000001000000000000000000000
const AlarmAirTempHigh = 0b00000000010000000000000000000000
const AlarmCoolantPressureHigh = 0b00000000100000000000000000000000
const AlarmCellVoltageHigh = 0b00000001000000000000000000000000
const AlarmIsoLow = 0b00000010000000000000000000000000
const AlarmH2AirDiffHighMinus = 0b00000100000000000000000000000000
const AlarmH2AirDiffHighPlus = 0b00001000000000000000000000000000
const AlarmStartUploss = 0b00010000000000000000000000000000
const AlarmH2Leakageloss = 0b00100000000000000000000000000000

func (al *AlarmsType) Text() []string {
	alarmText := make([]string, 0)
	if (al.bitMap & AlarmAirPressureLow) != 0 {
		alarmText = append(alarmText, "Abnormal low air pressure")
	}
	if (al.bitMap & AlarmAirPressureHigh) != 0 {
		alarmText = append(alarmText, "Abnormal high air pressure")
	}
	if (al.bitMap & AlarmAirTempHigh) != 0 {
		alarmText = append(alarmText, "Abnormally high air temperature")
	}
	if (al.bitMap & AlarmCoolantTempOutDiff) != 0 {
		alarmText = append(alarmText, "Abnormal temperature difference between inlet and outlet")
	}
	if (al.bitMap & AlarmCoolantTempHigh) != 0 {
		alarmText = append(alarmText, "Abnormally high outlet water temperature")
	}
	if (al.bitMap & AlarmCoolantPressureHigh) != 0 {
		alarmText = append(alarmText, "Abnormal high cooling water pressure")
	}
	if (al.bitMap & AlarmCellVoltageHigh) != 0 {
		alarmText = append(alarmText, "Stack cell high voltage abnormality")
	}
	if (al.bitMap & AlarmDcdcFault) != 0 {
		alarmText = append(alarmText, "DC to DC Converter Fault")
	}
	if (al.bitMap & AlarmDcdcCommunicationFault) != 0 {
		alarmText = append(alarmText, "DC to DC Converter Communication Fault")
	}
	if (al.bitMap & AlarmFcuToVcuFault) != 0 {
		alarmText = append(alarmText, "FCU communication abnormal")
	}
	if (al.bitMap & AlarmH2AirDiffHighMinus) != 0 {
		alarmText = append(alarmText, "Abnormal large hydrogen-air pressure difference (negative direction)")
	}
	if (al.bitMap & AlarmH2AirDiffHighPlus) != 0 {
		alarmText = append(alarmText, "Abnormal hydrogen-air pressure difference (forward direction)")
	}
	if (al.bitMap & AlarmH2concentration) != 0 {
		alarmText = append(alarmText, "Abnormal hydrogen concentration in the module")
	}
	if (al.bitMap & AlarmH2CirculatingPumpFault) != 0 {
		alarmText = append(alarmText, "Abnormal hydrogen pump")
	}
	if (al.bitMap & AlarmH2Leakageloss) != 0 {
		alarmText = append(alarmText, "Hydrogen leak check failed")
	}
	if (al.bitMap & AlarmH2OutPressureLow) != 0 {
		alarmText = append(alarmText, "H2 Outlet Pressure Low")
	}
	if (al.bitMap & AlarmH2PressureHigh) != 0 {
		alarmText = append(alarmText, "Abnormally high hydrogen pressure")
	}
	if (al.bitMap & AlarmH2PressureSensorFault) != 0 {
		alarmText = append(alarmText, "The hydrogen outlet pressure sensor is abnormal")
	}
	if (al.bitMap & AlarmH2SOCLow) != 0 {
		alarmText = append(alarmText, "Hydrogen tank SOC is too low")
	}
	if (al.bitMap & AlarmH2SPCheckFault) != 0 {
		alarmText = append(alarmText, "Hydrogen pressure sensor self-test is abnormal")
	}
	if (al.bitMap & AlarmH2TankLowPressure) != 0 {
		alarmText = append(alarmText, "Abnormal low pressure of hydrogen tank")
	}
	if (al.bitMap & AlarmH2TankMidPressure) != 0 {
		alarmText = append(alarmText, "Abnormal pressure in the hydrogen tank")
	}
	if (al.bitMap & AlarmH2TankHighPressure) != 0 {
		alarmText = append(alarmText, "Abnormal high pressure of hydrogen tank")
	}
	if (al.bitMap & AlarmH2TankTemp) != 0 {
		alarmText = append(alarmText, "Abnormal temperature of hydrogen tank")
	}
	if (al.bitMap & AlarmIsoLow) != 0 {
		alarmText = append(alarmText, "Abnormal low insulation")
	}
	if (al.bitMap & AlarmPtcFault) != 0 {
		alarmText = append(alarmText, "Heater failure")
	}
	if (al.bitMap & AlarmStartUploss) != 0 {
		alarmText = append(alarmText, "Low starting hydrogen pressure (below 20KPA)")
	}
	if (al.bitMap & AlarmTempSensorFault) != 0 {
		alarmText = append(alarmText, "Abnormal temperature sensor")
	}
	if (al.bitMap & AlarmVoltageLow) != 0 {
		alarmText = append(alarmText, "Single cell voltage undervoltage")
	}
	if (al.bitMap & AlarmWaterPumpFault) != 0 {
		alarmText = append(alarmText, "Water pump failure")
	}
	return alarmText
}

const CanStackOutputMsg = 0x161088A7

type StackOutputType struct {
	Voltage uint16 //Stack voltage
	Current uint16 //Stack current
	Power   uint32 //Stack power
}

func (t *StackOutputType) Load(data [8]byte) {
	t.Voltage = binary.LittleEndian.Uint16(data[0:2])
	t.Current = binary.LittleEndian.Uint16(data[2:4])
	// Power is actually a 24 bit value
	t.Power = uint32(data[4]) | uint32(data[5])<<8 | uint32(data[6])<<16

	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.StackVoltage = t.Voltage
	dbRecord.StackCurrent = t.Current
}

func (t *StackOutputType) GetVoltage() float64 {
	return float64(t.Voltage) / 10.0
}

func (t *StackOutputType) GetCurrent() float64 {
	return float64(t.Current) / 10.0
}

func (t *StackOutputType) GetPower() float64 {
	return float64(t.Power) / 10.0
}

const CanCff1Msg = 0x8CFF1C91

type CffMsgType struct {
	GasConcentration uint8
	MSBSide          byte
	CycleCounter     uint8
	SensorFaultCode  byte
	LSBCheckSumq     byte
}

func (t *CffMsgType) Load(data [8]byte) {
	t.GasConcentration = data[0]
	t.MSBSide = data[1]
	t.CycleCounter = data[2] & 0x0f
	t.SensorFaultCode = (data[2] & 0x30) >> 4
	t.LSBCheckSumq = data[4]

	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.HydrogenConcentration = t.GasConcentration
}

func (t *CffMsgType) GetGasConcentration() int16 {
	return (int16(t.GasConcentration) * 500) - 5500
}

const CanInsulationMsg = 0x18FEA3B2

type InsulationType struct {
	InsulationStatusCode byte
	InsulationStatus     byte
	InsulationResistance uint16
	IsolationBattVolt    uint16
	IsolationLife        uint8
}

func (t *InsulationType) Load(data [8]byte) {
	t.InsulationStatusCode = data[0] & 0x0f
	t.InsulationStatus = (data[0] & 0x30) >> 4
	t.InsulationResistance = binary.LittleEndian.Uint16(data[1:3])
	t.IsolationBattVolt = binary.LittleEndian.Uint16(data[3:5])
	t.IsolationLife = data[7]

	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.InsulationResistance = t.InsulationResistance
}

func (t *InsulationType) getStatus() string {
	if t.InsulationStatus == 0 {
		return ""
	}
	switch t.InsulationStatusCode {
	case 0x0010:
		return "Normal Operation"
	case 0b0100:
		return "Wiring Fault"
	case 0b0101:
		return "The high voltage positive electrode has a small insulation resistance to the ground"
	case 0b0110:
		return "The high voltage negative electrode has a small insulation resistance to the ground"
	default:
		return "Unknown Status"
	}
}

func (t *InsulationType) getFault() string {
	switch t.InsulationStatus {
	case 0b11:
		return "Device Fault"
	case 0b01:
		return "level 1 alarm(resistance<100K)"
	case 0b10:
		return "level 2 alarm(resistance is between 100K-500K)"
	default:
		return "Normal"
	}
}

const CanStackCellsID1to4Msg = 0x1810A7B1
const CanStackCellsID5to8Msg = 0x1811A7B1
const CanStackCellsID9to12Msg = 0x1812A7B1
const CanStackCellsID13to16Msg = 0x1813A7B1
const CanStackCellsID17to20Msg = 0x1814A7B1
const CanStackCellsID21to24Msg = 0x1815A7B1
const CanStackCellsID25to28Msg = 0x1816A7B1
const CanStackCellsID29to32Msg = 0x1817A7B1
const CanMaxMinCellsMsg = 0x1801A7B1
const CanTotalStackVoltageMsg = 0x1802A7B1

type StackCellsType struct {
	StackCellVoltage           [5][32]uint16
	TotakStackVoltage          uint16
	StdDeviation               uint16
	Temperature                uint16
	StackControllerFaultStatus byte
	LifeSignal                 byte
	MinCellVolts               uint16
	MaxCellVolts               uint16
	AvgCellVolts               uint16
	IndexMaxVoltsCell          uint8
	IndexMinVoltsCell          uint8
	loop                       uint8
}

func (t *StackCellsType) Load(id uint32, data [8]byte) {
	switch id {
	case CanStackCellsID1to4Msg:
		t.loop++
		if t.loop > 4 {
			t.loop = 0
		}
		t.StackCellVoltage[t.loop][0] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][1] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][2] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][3] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID5to8Msg:
		t.StackCellVoltage[t.loop][4] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][5] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][6] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][7] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID9to12Msg:
		t.StackCellVoltage[t.loop][8] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][9] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][10] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][11] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID13to16Msg:
		t.StackCellVoltage[t.loop][12] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][13] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][14] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][15] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID17to20Msg:
		t.StackCellVoltage[t.loop][16] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][17] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][18] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][19] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID21to24Msg:
		t.StackCellVoltage[t.loop][20] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][21] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][22] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][23] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID25to28Msg:
		t.StackCellVoltage[t.loop][24] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][25] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][26] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][27] = binary.LittleEndian.Uint16(data[6:8])
	case CanStackCellsID29to32Msg:
		t.StackCellVoltage[t.loop][28] = binary.LittleEndian.Uint16(data[0:2])
		t.StackCellVoltage[t.loop][29] = binary.LittleEndian.Uint16(data[2:4])
		t.StackCellVoltage[t.loop][30] = binary.LittleEndian.Uint16(data[4:6])
		t.StackCellVoltage[t.loop][31] = binary.LittleEndian.Uint16(data[6:8])
	case CanMaxMinCellsMsg:
		t.MaxCellVolts = binary.LittleEndian.Uint16(data[0:2])
		t.MinCellVolts = binary.LittleEndian.Uint16(data[2:4])
		t.AvgCellVolts = binary.LittleEndian.Uint16(data[4:6])
		t.IndexMaxVoltsCell = data[6]
		t.IndexMinVoltsCell = data[7]
		dbRecord.mu.Lock()
		defer dbRecord.mu.Unlock()
		dbRecord.MaxCellVolts = t.MaxCellVolts
		dbRecord.MinCellVolts = t.MinCellVolts
		dbRecord.IdxMinCell = t.IndexMinVoltsCell
		dbRecord.IdxMaxCell = t.IndexMaxVoltsCell
		if t.loop == 0 {
			for i := 0; i < len(t.StackCellVoltage[0]); i++ {
				dbRecord.CellVoltages[i] = t.GetStackCellVoltage(i)
			}
		}
	case CanTotalStackVoltageMsg:
		t.TotakStackVoltage = binary.LittleEndian.Uint16(data[0:2])
		t.StdDeviation = binary.LittleEndian.Uint16(data[2:4])
		t.Temperature = binary.LittleEndian.Uint16(data[4:6])
		t.StackControllerFaultStatus = data[6]
		t.LifeSignal = data[7]
	}
}

func (t *StackCellsType) GetStackCellVoltage(cell int) int16 {
	var volts = int32(0)
	for idx := 0; idx < 5; idx++ {
		volts += int32(t.StackCellVoltage[idx][cell])
	}
	return int16(volts/5) - 5000
}

func (t *StackCellsType) GetMaxCellVoltage() int16 {
	return int16(t.MaxCellVolts) - 5000
}

func (t *StackCellsType) GetMinCellVoltage() int16 {
	return int16(t.MinCellVolts) - 5000
}

func (t *StackCellsType) GetAvgCellVoltage() int16 {
	return int16(t.AvgCellVolts) - 5000
}

const CanATSCoolingFanMsg = 0x19BBB701

type ATSCoolingFanType struct {
	Enable uint16
	Speed  uint16
}

func (t *ATSCoolingFanType) Load(data [8]byte) {
	t.Enable = binary.LittleEndian.Uint16(data[0:2])
	t.Speed = binary.LittleEndian.Uint16(data[2:4])

	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.CoolantFanSpeed = t.Speed
}

const CanWaterPumpMsg = 0x18FAC503

type WaterPumpType struct {
	Speed   uint16
	Voltage uint8
	Current uint8
}

func (t *WaterPumpType) Load(data [8]byte) {
	t.Speed = binary.LittleEndian.Uint16(data[0:2])
	t.Voltage = data[2]
	t.Current = data[3]

	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.CoolantPumpSpeed = t.Speed
	dbRecord.CoolantPumpAmps = t.Current
	dbRecord.CoolantPumpVolts = t.Voltage
}

func (t *WaterPumpType) getVoltage() float64 {
	return float64(t.Voltage) * 0.2
}

func (t *WaterPumpType) getCurrent() float64 {
	return float64(t.Current) * 0.2
}

const CanDCDCConverterMsg = 0x1029FF00

type DCDCConverterType struct {
	InputCurrent  uint16
	InputVoltage  uint16
	OutputCurrent uint16
	OutputVoltage uint16
}

func (t *DCDCConverterType) Load(data [8]byte) {
	t.InputCurrent = binary.LittleEndian.Uint16(data[0:2])
	t.InputVoltage = binary.LittleEndian.Uint16(data[2:4])
	t.OutputCurrent = binary.LittleEndian.Uint16(data[4:6])
	t.OutputVoltage = binary.LittleEndian.Uint16(data[6:8])
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.DCDCInVolts = t.InputVoltage
	dbRecord.DCDCOutVolts = t.OutputVoltage
	dbRecord.DCDCInAmps = t.InputCurrent
	dbRecord.DCDCOutAmps = t.OutputCurrent
}

func (t *DCDCConverterType) GetInputCurrent() float64 {
	return float64(t.InputCurrent) / 10.0
}

func (t *DCDCConverterType) GetOutputCurrent() float64 {
	return float64(t.OutputCurrent) / 10.0
}

func (t *DCDCConverterType) GetInputVoltage() float64 {
	return float64(t.InputVoltage) / 100.0
}

func (t *DCDCConverterType) GetOutputVoltage() float64 {
	return float64(t.OutputVoltage) / 100.0
}

const CanDCOutputMsg = 0x18FFB587

type DCOutputType struct {
	Temp         uint8
	Status       uint8
	FaultLevel   uint8
	ErrorCode    byte
	OutVolltage  uint8
	OutCurrent   uint8
	InputVoltage uint8
	InternalTest uint8
	LIFE         uint8
}

func (t *DCOutputType) Load(data [8]byte) {
	t.Temp = data[0]
	t.Status = data[1] & 0x0f
	t.FaultLevel = (data[1] & 0xf0) >> 4
	t.ErrorCode = data[2]
	t.OutVolltage = data[3]
	t.OutCurrent = data[4]
	t.InputVoltage = data[5]
	t.InternalTest = data[6]
	t.LIFE = data[7]
	dbRecord.mu.Lock()
	defer dbRecord.mu.Unlock()
	dbRecord.DCDCTemp = t.Temp
}

func (t *DCOutputType) GetTemperature() int16 {
	return int16(t.Temp) - 40
}

func (t *DCOutputType) GetStatus() string {
	switch t.Status {
	case 0:
		return "Stop"
	case 1:
		return "Running"
	case 2:
		return "discharge/soft off state"
	default:
		return "fault"
	}
}

func (t *DCOutputType) GetFaultCode() string {
	switch t.ErrorCode {
	case 0x01:
		return "Communication failure 1 (master-slave 1)"
	case 0x02:
		return "Communication failure 2 (master-slave 2)"
	case 0x03:
		return "Customer order input error"
	case 0x04:
		return "output overcurrent"
	case 0x06:
		return "output voltage overvoltage"
	case 0x07:
		return "output voltage undervoltage"
	case 0x08:
		return "Input total current overcurrent"
	case 0x09:
		return "input overvoltage"
	case 0x0A:
		return "input undervoltage"
	case 0x0B:
		return "overheating"
	case 0x0C:
		return "Voltage relationship protection"
	case 0x0D:
		return "Maximum power protection"
	case 0x0E:
		return "Bus communication failure"
	case 0x0F:
		return "negative current protection"
	case 0x11:
		return "input precharge failed 1"
	case 0x12:
		return "input precharge failed 2"
	case 0x13:
		return "Output precharge failed 1"
	case 0x14:
		return "Output precharge failed 2"
	case 0x15:
		return "Input short circuit fault"
	case 0x16:
		return "Output short circuit fault"
	case 0xFF:
		return "Auxiliary electrical failure"
	case 0x18:
		return "BUS overvoltage"
	default:
		return ""
	}
}

const CanKeyOnMsg = 0x161088AD
const CanRunTimeMsg = 0x1610AAAB

type SystemInfoType struct {
	Run              bool
	ExhaustFlag      bool
	Hours            uint8
	Mins             uint8
	exhaustFlagTimer *time.Timer
	exhaustLastValue bool
}

func (t *SystemInfoType) SetRunFlag(data byte) {
	t.Run = data != 0
}

func (t *SystemInfoType) SetRunTime(hours byte, mins byte) {
	t.Hours = hours
	t.Mins = mins
}

func (t *SystemInfoType) SetExhaustFlag() {
	log.Println("Set Exhaust")
	t.ExhaustFlag = true
	t.exhaustFlagTimer.Reset(time.Second)
}

const CanBMSSettingsMsg = 0x1610AAAA

type BMSSettingsType struct {
	TargetPowerLevel uint8
	BMSHigh          uint16
	BMSLow           uint16
	CurrentPower     uint8
}

func (t *BMSSettingsType) Load(data [8]byte) {
	t.BMSHigh = binary.LittleEndian.Uint16(data[0:2])
	t.BMSLow = binary.LittleEndian.Uint16(data[2:4])
	t.TargetPowerLevel = data[4]
	t.CurrentPower = data[5]
}

type PANFuelCell struct {
	mu            sync.Mutex
	bus           *CANBus
	SystemInfo    SystemInfoType
	PowerMode     PowerModeType
	Pressures     PressuresType
	StackCoolant  StackCoolantType
	AirFlow       AirFlowType
	Alarms        AlarmsType
	StackOutput   StackOutputType
	CffMsg        CffMsgType
	Insulation    InsulationType
	StackCells    StackCellsType
	ATSCoolingFan ATSCoolingFanType
	WaterPump     WaterPumpType
	DCDCConverter DCDCConverterType
	DCOutput      DCOutputType
	BMSSettings   BMSSettingsType
	Control       PanSettingsType
}

func (fc *PANFuelCell) init(canBus *CANBus) {
	fc.bus = canBus
	fc.SystemInfo.exhaustFlagTimer = time.AfterFunc(time.Second, func() { fc.SystemInfo.ExhaustFlag = false })
}

func (fc *PANFuelCell) getJSON() (string, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	jsonBytes, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return "", err
	} else {
		return string(jsonBytes), nil
	}
}

func (fc *PANFuelCell) setTargetPower(kw float64) error {
	if (kw <= 10.0) && (kw >= 0) {
		fc.Control.TargetPower = kw
		dbRecord.mu.Lock()
		defer dbRecord.mu.Unlock()
		dbRecord.PowerRequested = uint8(kw * 10)
		//		log.Println("Power = ", kw, dbRecord.PowerRequested)
		return nil
	}
	return fmt.Errorf("valid range for target power is 0kW to 10kW. %01fkW was requested", kw)
}

func (fc *PANFuelCell) setTargetBattHigh(volts float64) error {
	if (volts >= 35) && (volts <= 70) && (volts >= fc.Control.TargetBatteryLow) {
		fc.Control.TargetBatteryHigh = volts
		currentSettings.FuelCellSettings.HighBatterySetpoint = volts
		dbRecord.MaxBattVolts = int16(volts * 10)
		if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
			log.Println(err)
		}
		return nil
	}
	return fmt.Errorf("valid range for battery voltage high is 35V to 70V and must be above or equal to battery voltage low. %01fV was requested", volts)
}

func (fc *PANFuelCell) setTargetBattLow(volts float64) error {
	if (volts >= 35) && (volts <= 70) {
		fc.Control.TargetBatteryLow = volts
		if volts > fc.Control.TargetBatteryHigh {
			fc.Control.TargetBatteryHigh = fc.Control.TargetBatteryLow
		}
		currentSettings.FuelCellSettings.LowBatterySetpoint = volts
		dbRecord.MinBattVolts = int16(volts * 10)
		if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
			log.Println(err)
		}
		return nil
	}
	return fmt.Errorf("valid range for battery voltage low is 35V to 70V and must be below or equal to battery voltage high. %01fV was requested", volts)
}

/**
start sends the start command (1) NUMCOMMANDREPEATS times
*/
func (fc *PANFuelCell) start() {
	fc.Control.FuelCellOn = true
	if file, err := os.Create("/FireflyService/Recordings/fc_running"); err != nil {
		log.Println(err)
	} else {
		defer func() {
			if err := file.Close(); err != nil {
				log.Println(err)
			}
		}()
	}
	output.FuelCellRunEnable = FCStartUp
	output.repeats = NUMCOMMANDREPEATS
	if err := fc.updateOutput(); err != nil {
		log.Println(err)
	}
	log.Println("Start the fuel cell")
}

/**
start sends the stop command (2) NUMCOMMANDREPEATS times
*/
func (fc *PANFuelCell) stop() {
	fc.Control.FuelCellOn = false
	output.FuelCellRunEnable = FCShutDown
	output.repeats = NUMCOMMANDREPEATS
	if err := fc.updateOutput(); err != nil {
		log.Println(err)
	}
	log.Println("Stop the fuel cell")

	if err := os.Remove("/FireflyService/Recordings/fc_running"); err != nil {
		log.Println(err)
	}
}

func (fc *PANFuelCell) exhaustOpen() {
	fc.Control.Exhaust = true
	if err := fc.updateOutput(); err != nil {
		log.Println(err)
	}
	log.Println("Exhaust is open")
}

func (fc *PANFuelCell) exhaustClose() {
	fc.Control.Exhaust = false
	if err := fc.updateOutput(); err != nil {
		log.Println(err)
	}
	log.Println("Exhaust is closed")
}

/**
updateSettings sends the can control messages to the fuel cell
*/
var limits BatteryVoltageLimitsType

func (fc *PANFuelCell) updateSettings() error {

	// Only send settings if the fuel cell is enabled
	if currentSettings.IsFuelCellEnabled() {
		limits.BMSHighVoltage = uint16(fc.Control.TargetBatteryHigh * 10)
		limits.BMSLowVoltage = uint16(fc.Control.TargetBatteryLow * 10)
		limits.IsoFlag = currentSettings.FuelCellSettings.IgnoreIsoLow
		if fc.bus != nil {
			return limits.UpdateFuelCell(fc.bus)
		} else {
			return fmt.Errorf("no active CAN bus found for the fuel cell")
		}
	}
	return nil
}

var output OutputControlType

func (fc *PANFuelCell) updateOutput() error {
	// Only send commands if the fuel cell is enabled
	if currentSettings.IsFuelCellEnabled() {
		if fc.Control.FuelCellOn {
			output.PowerDemand = uint8(fc.Control.TargetPower * 10)
		} else {
			output.PowerDemand = 0
		}
		if fc.Control.Exhaust {
			output.ExhaustMode = ExhaustOpen
		} else {
			output.ExhaustMode = ExhaustClosed
		}
		if fc.bus != nil {
			if output.repeats > 0 {
				// If we are sending a stop command, wait for the run status goes away before counting down the repeats
				if output.FuelCellRunEnable != FCShutDown || fc.PowerMode.PowerModeState != PMStartup {
					output.repeats--
				}
			} else {
				output.FuelCellRunEnable = FCNoCommand
			}
			return output.UpdateFuelCell(fc.bus)
		} else {
			return fmt.Errorf("no active CAN bus found for the fuel cell")
		}
	}
	return nil
}

type PanStatus struct {
	System               string
	Version              string
	RunTimeHours         uint16
	RunTimeMinutes       uint8
	RunState             bool
	H2Pressure           float64 // Hydrogen pressure
	AirPressure          float64 // Air pressure
	CoolantPressure      float64 // Coolant pressure
	H2AirPressureDiff    float64 // Hydrogen air pressure difference
	CoolantInletTemp     float64
	CoolantOutletTemp    float64
	AirTemp              float64
	AmbientTemp          float64
	AirFlow              float64
	StackVolts           float64
	StackCurrent         float64
	StackPower           float64
	DCInVolts            float64
	DCInAmps             float64
	DCOutVolts           float64
	DCOutAmps            float64
	BMSPower             float64
	BMSHigh              float64
	BMSLow               float64
	BMSCurrentPower      float64
	BMSTargetPower       float64
	BMSTargetHigh        float64
	BMSTargetLow         float64
	RunStatus            string
	Alarms               []string
	DCOutputStatus       string
	DCOutputFaultCode    string
	Start                bool
	ExhaustOpen          bool
	Enable               bool
	InsulationResistance uint16
	InsulationStatus     string
	InsulationFault      string
	WaterPumpSpeed       uint16
	WaterPumpActive      bool
	CoolingFanSpeed      uint16
	ClearFaultsActive    bool
}

/*
GetStatus sends a status block from the fuel cell
*/
func (fc *PANFuelCell) GetStatus() *PanStatus {
	if !currentSettings.FuelCell {
		return nil
	}
	var status PanStatus
	fc.mu.Lock()
	defer fc.mu.Unlock()

	status.ClearFaultsActive = limits.ClearFaults
	status.System = currentSettings.Name
	status.Version = version
	status.RunTimeHours = uint16(fc.SystemInfo.Hours)
	status.RunTimeMinutes = fc.SystemInfo.Mins
	status.ExhaustOpen = fc.SystemInfo.ExhaustFlag
	status.RunState = fc.SystemInfo.Run
	status.Enable = currentSettings.FuelCellSettings.Enabled
	status.H2Pressure = (float64(fc.Pressures.H2Pressure) - 500) / 10.0
	status.AirPressure = (float64(fc.Pressures.AirPressure) - 500) / 10.0
	status.CoolantPressure = (float64(fc.Pressures.CoolantPressure) - 500) / 10.0
	status.H2AirPressureDiff = (float64(fc.Pressures.H2AirPressureDiff) - 50) / 10.0
	status.CoolantInletTemp = ((float64(fc.StackCoolant.CoolantInTemp)) / 10.0) - 40.0
	status.CoolantOutletTemp = ((float64(fc.StackCoolant.CoolantOutTemp)) / 10.0) - 40.0
	status.AirTemp = ((float64(fc.StackCoolant.AirTemp)) / 10.0) - 40.0
	status.AmbientTemp = ((float64(fc.StackCoolant.AmbientTemp)) / 10.0) - 40.0
	status.AirFlow = float64(fc.AirFlow.Flow) / 10.0
	status.StackVolts = float64(fc.StackOutput.Voltage) / 10.0
	status.StackCurrent = float64(fc.StackOutput.Current) / 10.0
	status.StackPower = float64(fc.StackOutput.Power) / 10.0
	status.DCInVolts = float64(fc.DCDCConverter.InputVoltage) / 100.0
	status.DCOutVolts = float64(fc.DCDCConverter.OutputVoltage) / 10.0
	status.DCInAmps = float64(fc.DCDCConverter.InputCurrent) / 10.0
	status.DCOutAmps = float64(fc.DCDCConverter.OutputCurrent) / 100.0
	status.BMSPower = float64(fc.BMSSettings.TargetPowerLevel)
	status.BMSHigh = float64(fc.BMSSettings.BMSHigh) / 10.0
	status.BMSLow = float64(fc.BMSSettings.BMSLow) / 10.0
	status.BMSCurrentPower = float64(fc.BMSSettings.CurrentPower)
	status.BMSTargetPower = fc.Control.TargetPower
	status.BMSTargetHigh = fc.Control.TargetBatteryHigh
	status.BMSTargetLow = fc.Control.TargetBatteryLow
	status.RunStatus = fc.PowerMode.PowerModeState.String()
	status.Alarms = fc.Alarms.Text()
	status.DCOutputStatus = fc.DCOutput.GetStatus()
	status.DCOutputFaultCode = fc.DCOutput.GetFaultCode()
	status.Start = fc.Control.FuelCellOn
	if !currentSettings.FuelCellSettings.IgnoreIsoLow {
		status.InsulationResistance = fc.Insulation.InsulationResistance
		status.InsulationStatus = fc.Insulation.getStatus()
		status.InsulationFault = fc.Insulation.getFault()
	} else {
		status.InsulationResistance = 0xFFFF
		status.InsulationStatus = ""
		status.InsulationFault = ""
	}
	status.WaterPumpSpeed = fc.WaterPump.Speed
	status.WaterPumpActive = fc.Control.PumpActive
	status.CoolingFanSpeed = fc.ATSCoolingFan.Speed
	return &status
}

func (fc *PANFuelCell) GetStatusAsJSON() ([]byte, error) {

	jsonBytes, err := json.MarshalIndent(fc.GetStatus(), "", "  ")
	if err != nil {
		return make([]byte, 0), err
	} else {
		return jsonBytes, nil
	}
}

func (fc *PANFuelCell) ClearFaults() {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.stop()
	limits.ClearFaults = true
	limits.repeats = NUMCOMMANDREPEATS
}

type PANDatabaseRecordType struct {
	StackCurrent          uint16
	StackVoltage          uint16
	CoolantInlTemp        uint16
	CoolantOutTemp        uint16
	OutputVoltage         uint16
	OutputCurrent         uint16
	CoolantFanSpeed       uint16
	CoolantPumpSpeed      uint16
	CoolantPumpVolts      uint8
	CoolantPumpAmps       uint8
	InsulationResistance  uint16
	HydrogenPressure      uint16
	AirPressure           uint16
	CoolantPressure       uint16
	AirinletTemp          uint16
	AmbientTemp           uint16
	AirFlow               uint16
	HydrogenConcentration uint8
	DCDCTemp              uint8
	DCDCInVolts           uint16
	DCDCOutVolts          uint16
	DCDCInAmps            uint16
	DCDCOutAmps           uint16
	MinCellVolts          uint16
	MaxCellVolts          uint16
	AvgCellVolts          uint16
	IdxMaxCell            uint8
	IdxMinCell            uint8
	RunStage              byte
	FaultLevel            byte
	PowerModeState        byte
	CellVoltages          [32]int16
	Alarms                uint32
	PowerRequested        uint8
	MaxBattVolts          int16
	MinBattVolts          int16
	mu                    sync.Mutex
	stmt                  *sql.Stmt
}

var dbRecord PANDatabaseRecordType

func (rec *PANDatabaseRecordType) saveToDatabase() error {
	rec.mu.Lock()
	defer rec.mu.Unlock()

	_, err := rec.stmt.Exec(
		rec.StackCurrent, rec.StackVoltage, rec.CoolantPressure, rec.CoolantOutTemp, rec.DCDCOutVolts,
		rec.DCDCOutAmps, rec.CoolantFanSpeed, rec.CoolantPumpSpeed, rec.CoolantPumpVolts, rec.CoolantPumpAmps,
		rec.InsulationResistance, rec.HydrogenPressure, rec.AirPressure, rec.CoolantPressure, rec.AirinletTemp,
		rec.AmbientTemp, rec.AirFlow, rec.HydrogenConcentration, rec.DCDCTemp, rec.DCDCInVolts,
		rec.DCDCOutVolts, rec.DCDCInAmps, rec.DCDCOutAmps, rec.MinCellVolts, rec.MaxCellVolts,
		rec.AvgCellVolts, rec.IdxMaxCell, rec.IdxMinCell, rec.RunStage, rec.FaultLevel,
		rec.CellVoltages[0], rec.CellVoltages[1], rec.CellVoltages[2], rec.CellVoltages[3], rec.CellVoltages[4],
		rec.CellVoltages[5], rec.CellVoltages[6], rec.CellVoltages[7], rec.CellVoltages[8], rec.CellVoltages[9],
		rec.CellVoltages[10], rec.CellVoltages[11], rec.CellVoltages[12], rec.CellVoltages[13], rec.CellVoltages[14],
		rec.CellVoltages[15], rec.CellVoltages[16], rec.CellVoltages[17], rec.CellVoltages[18], rec.CellVoltages[19],
		rec.CellVoltages[20], rec.CellVoltages[21], rec.CellVoltages[22], rec.CellVoltages[23], rec.CellVoltages[24],
		rec.CellVoltages[25], rec.CellVoltages[26], rec.CellVoltages[27], rec.CellVoltages[28], rec.CellVoltages[29],
		rec.CellVoltages[30], rec.CellVoltages[31], rec.Alarms, rec.PowerRequested, rec.MaxBattVolts, rec.MinBattVolts, rec.PowerModeState)
	if err != nil {
		log.Println(err)
	}
	return err
}

func getFuelCellDCDCData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "DC-DC Data"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendDataAsJSON(w, DeviceString, FuelCellDCDCVoltagesByMinute, start, end)
		} else {
			SendDataAsJSON(w, DeviceString, FuelCellDCDCVoltagesBySecond, start, end)
		}
		if err != nil {
			ReturnJSONError(w, DeviceString, err, http.StatusInternalServerError, true)
		}
	}
}

func getFuelCellStackData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "DC-DC Data"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendDataAsJSON(w, DeviceString, FuelCellStackVoltagesByMinute, start, end)
		} else {
			SendDataAsJSON(w, DeviceString, FuelCellStackVoltagesBySecond, start, end)
		}
	}
}

func getFuelCellPressureData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "FuelCell Pressures Data"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendDataAsJSON(w, DeviceString, FuelCellPressuresByMinute, start, end)
		} else {
			SendDataAsJSON(w, DeviceString, FuelCellPressuresBySecond, start, end)
		}
	}
}

func getFuelCellCoolantData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "FuelCell Coolant Data"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendDataAsJSON(w, DeviceString, FuelCellCoolantByMinute, start, end)
		} else {
			SendDataAsJSON(w, DeviceString, FuelCellCoolantBySecond, start, end)
		}
	}
}
