package main

import (
	"database/sql"
	"log"
	"net"
)

type ElectrolyserStatusType struct {
	Device                               uint8                  `json:"device"`
	Name                                 string                 `json:"name"`
	Powered                              bool                   `json:"on"`                                   // Relay is turned on
	Model                                string                 `json:"model"`                                // 0
	FirmwareMajor                        uint16                 `json:"firmwareMajor"`                        // 2
	FirmwarePatch                        uint16                 `json:"firmwarePatch"`                        // 3
	FirmwareBuild                        uint32                 `json:"firmwareBuild"`                        // 4
	DeviceControlBoardSerial             string                 `json:"deviceControlBoardSerial"`             // 6
	Serial                               string                 `json:"serial"`                               // 14
	SystemState                          uint16                 `json:"systemState"`                          // 18
	SystemHours                          jsonFloat32            `json:"liveTime"`                             // 20
	StackSerialNumber                    string                 `json:"stackSerialNumber"`                    // 1000
	StackStartStopCycles                 uint32                 `json:"stackStartStopCycles"`                 // 1002
	StackTotalRunTime                    uint32                 `json:"stackTotalRunTime"`                    // 1004
	StackTotalProduction                 jsonFloat32            `json:"stackTotalProduction"`                 // 1006
	H2Flow                               jsonFloat32            `json:"h2Flow"`                               // 1008
	ProductCode                          uint32                 `json:"productCode"`                          // 1010
	State                                uint16                 `json:"state"`                                // 1200
	ElectrolyteLevel                     ElectrolyteLevelType   `json:"electrolyteLevel"`                     // (7000 - 7003 four booleans)
	ElectrolyteTankPressureTooHigh       bool                   `json:"electrolyteTankPressureTooHigh"`       // 7004
	HydrogenPressureTooHigh              bool                   `json:"hydrogenPressureTooHigh"`              // 7005
	DownstreamHighTemperature            bool                   `json:"downstreamHighTemperature"`            // 7006
	ElectronicCompartmentHighTemp        bool                   `json:"electronicCompartmentHighTemp"`        // 7007
	VeryLowElectrolyteTemp               bool                   `json:"veryLowElectrolyteTemp"`               // 7008
	ChassisWaterPresent                  bool                   `json:"chassisWaterPresent"`                  // 7009
	DryContact                           bool                   `json:"dryContact"`                           // 7010
	ElectrolyteCoolerFanSpeed            jsonFloat32            `json:"electrolyteCoolerFanSpeed"`            // 7500
	AirCirculationFanSpeed               jsonFloat32            `json:"airCirculationFanSpeed"`               // 7502
	ElectronicCompartmentCoolingFanSpeed jsonFloat32            `json:"electronicCompartmentCoolingFanSpeed"` // 7504
	ElectrolyteFlowMeter                 jsonFloat32            `json:"electrolyteFlowMeter"`                 // 7506
	StackCurrent                         jsonFloat32            `json:"stackCurrent"`                         // 7508
	StackVoltage                         jsonFloat32            `json:"stackVoltage"`                         // 7510
	InnerH2Pressure                      jsonFloat32            `json:"innerH2"`                              // 7512
	OuterH2Pressure                      jsonFloat32            `json:"outerH2"`                              // 7514
	WaterPressure                        jsonFloat32            `json:"waterPressure"`                        // 7516
	ElectrolyteTemp                      jsonFloat32            `json:"temp"`                                 // 7518
	DownstreamTemp                       jsonFloat32            `json:"downstreamTemp"`                       // 7520
	CurrentProductionRate                jsonFloat32            `json:"rate"`                                 // H1002
	MaxTankPressure                      jsonFloat32            `json:"maxPressure"`                          // H4308
	RestartPressure                      jsonFloat32            `json:"restartPressure"`                      // H4310
	DryerNetworkEnabled                  bool                   `json:"dryerNetworkEnabled"`
	DryerFailure                         string                 `json:"dryerFailure"`
	Warnings                             ElectrolyserEventsType `json:"warnings"` // 768
	Errors                               ElectrolyserEventsType `json:"errors"`   // 832
	Dryer                                *DryerStatusType       `json:"dryer"`
	IP                                   net.IP                 `json:"ip"`
	PowerRelay                           uint8                  `json:"powerRelay"`
	Enabled                              bool                   `json:"enabled"`
	monitored                            bool
	elm                                  ELMaintenanceType
}

type ElectrolyserJSONStatusType struct {
	Device                               uint8                `json:"device"`
	Name                                 string               `json:"name"`
	Powered                              bool                 `json:"on"` // Relay is turned on
	Model                                string               `json:"model"`
	Serial                               string               `json:"serial"`      // 14
	SystemState                          uint16               `json:"systemState"` // 18
	StackSerialNumber                    string               `json:"stackSerialNumber"`
	StackStartStopCycles                 int32                `json:"stackStartStopCycles"`
	StackTotalRunTime                    int32                `json:"stackTotalRunTime"`
	SystemRunTime                        int32                `json:"systemRunTime"`
	StackTotalProduction                 jsonFloat32          `json:"stackTotalProduction"`
	H2Flow                               jsonFloat32          `json:"h2Flow"` // 1008
	ProductCode                          string               `json:"productCode"`
	State                                uint16               `json:"state"`                                // 1200
	ElectrolyteLevel                     ElectrolyteLevelType `json:"electrolyteLevel"`                     // (7000 - 7003 four booleans)
	ElectrolyteTankPressureTooHigh       bool                 `json:"electrolyteTankPressureTooHigh"`       // 7004
	HydrogenPressureTooHigh              bool                 `json:"hydrogenPressureTooHigh"`              // 7005
	DownstreamHighTemperature            bool                 `json:"downstreamHighTemperature"`            // 7006
	ElectronicCompartmentHighTemp        bool                 `json:"electronicCompartmentHighTemp"`        // 7007
	VeryLowElectrolyteTemp               bool                 `json:"veryLowElectrolyteTemp"`               // 7008
	ChassisWaterPresent                  bool                 `json:"chassisWaterPresent"`                  // 7009
	DryContact                           bool                 `json:"dryContact"`                           // 7010
	ElectrolyteCoolerFanSpeed            jsonFloat32          `json:"electrolyteCoolerFanSpeed"`            // 7500
	AirCirculationFanSpeed               jsonFloat32          `json:"airCirculationFanSpeed"`               // 7502
	ElectronicCompartmentCoolingFanSpeed jsonFloat32          `json:"electronicCompartmentCoolingFanSpeed"` // 7504
	ElectrolyteFlowMeter                 jsonFloat32          `json:"electrolyteFlowMeter"`                 // 7506
	StackCurrent                         jsonFloat32          `json:"stackCurrent"`                         // 7508
	StackVoltage                         jsonFloat32          `json:"stackVoltage"`                         // 7510
	InnerH2Pressure                      jsonFloat32          `json:"innerH2"`                              // 7512
	OuterH2Pressure                      jsonFloat32          `json:"outerH2"`                              // 7514
	WaterPressure                        jsonFloat32          `json:"waterPressure"`                        // 7516
	ElectrolyteTemp                      jsonFloat32          `json:"temp"`                                 // 7518
	DownstreamTemp                       jsonFloat32          `json:"downstreamTemp"`                       // 7520
	CurrentProductionRate                jsonFloat32          `json:"rate"`                                 // H1002
	MaxTankPressure                      jsonFloat32          `json:"maxPressure"`                          // H4308
	RestartPressure                      jsonFloat32          `json:"restartPressure"`                      // H4310
	DryerNetworkEnabled                  bool                 `json:"dryerNetworkEnabled"`
	DryerFailure                         string               `json:"dryerFailure"`
	Warnings                             []string             `json:"warnings"` // 768
	Errors                               []string             `json:"errors"`   // 832
	Dryer                                *DryerStatusType     `json:"dryer"`
	IP                                   net.IP               `json:"ip"`
	PowerRelay                           uint8                `json:"powerRelay"`
	Enabled                              bool                 `json:"enabled"`
	PowerRelayEnergised                  bool                 `json:"powerRelayEnergised"`
}

func (elt *ElectrolyserStatusType) loadMaintenance(pdb *sql.DB, elm string) {
	if err := elt.elm.loadLatest(pdb, elt.Name); err != nil {
		log.Println(elm, err)
	}
}

func (elt *ElectrolyserStatusType) GetStackStartStopCycles() {

}

func (elt *ElectrolyserStatusType) IsRunning() bool {
	switch elt.State {
	case 0:
		return false // Halted
	case 1:
		return false // Maintenance
	case 2:
		return false // idle
	case 3:
		return true // Steady
	case 4:
		return true // Standby
	case 5:
		return false // Curve
	case 6:
		return false // Blow down
	default:
		return false // Unknown
	}
}

func (elt *ElectrolyserStatusType) GetProductCode() string {
	switch elt.ProductCode {
	case 0x00:
		return "ELE210535A2AXV01_03"
	case 0x01:
		return "ELE210508A2AXV01_03"
	case 0x02:
		return "ELE210535A2AXV04"
	case 0x03:
		return "ELE210508A2AXV04"
	case 0x04:
		return "ELE210535A2LSV01"
	case 0x05:
		return "ELE210508A2LXV01"
	case 0x06:
		return "ELE210535D4AXV01"
	case 0x07:
		return "ELE210508D4AXV01"
	case 0x08:
		return "ELE210535A2ASV05"
	case 0x09:
		return "ELE210508A2ASV05"
	case 0x0A:
		return "ELE210535A2ASV06"
	case 0x0B:
		return "ELE210508A2ASV06"
	case 0x0C:
		return "ELE210535A2ASV07"
	case 0x0D:
		return "ELE210508A2ASV07"
	case 0x0E:
		return "ELE210535A2LSV02"
	case 0x0F:
		return "ELE210508A2LSV02"
	case 0x10:
		return "ELE210535A2LSV03"
	case 0x11:
		return "ELE210508A2LSV03"
	case 0x12:
		return "ELE210535A2ASV08"
	case 0x13:
		return "ELE210508A2ASV08"
	case 0x14:
		return "ELE210535A2LSV04"
	case 0x15:
		return "ELE210508A2LSV04"
	case 0x16:
		return "ELE210535A2ASV09"
	case 0x17:
		return "ELE210508A2ASV09"
	case 0x18:
		return "ELE210535A2LSV05"
	case 0x19:
		return "ELE210508A2LSV05"
	case 0x1A:
		return "ELE210535D4ANV02"
	case 0x1C:
		return "ELE210535A2ASV10"
	case 0x1D:
		return "ELE210508A2ASV10"
	case 0x1E:
		return "ELE210535A2LSV06"
	case 0x1F:
		return "ELE210508A2LSV06"
	case 0x20:
		return "ELE210535A2ASV11"
	case 0x21:
		return "ELE210508A2ASV11"
	case 0x22:
		return "ELE210535A2LSV07"
	case 0x23:
		return "ELE210508A2LSV07"
	case 0x24:
		return "ELE210535A2ASV12"
	case 0x25:
		return "ELE210508A2ASV12"
	case 0x26:
		return "ELE210535A2LSV08"
	case 0x27:
		return "ELE210508A2LSV08"
	case 0x28:
		return "ELE400535A2ASV01"
	case 0x29:
		return "ELE400535D4ASV01"
	case 0x2A:
		return "ELE400535A2LSV01"
	case 0x2B:
		return "ELE400535D4LSV01"
	case 0x2C:
		return "ELE400535A2ASV02"
	case 0x2D:
		return "ELE400535D4ASV02"
	case 0x2E:
		return "ELE400535A2LSV02"
	case 0x2F:
		return "ELE400535D4LSV02"
	case 0x30:
		return "ELE210535A2ASV13"
	case 0x31:
		return "ELE210508A2ASV13"
	case 0x32:
		return "ELE210535A2LSV09"
	case 0x33:
		return "ELE210508A2LSV09"
	case 0x34:
		return "ELE400535A2ASV03"
	case 0x35:
		return "ELE400535A2LSV03"
	case 0x36:
		return "ELE210535D4ANV03"
	case 0x37:
		return "ELE210535D4ANV04"
	case 0x38:
		return "ELE400508A2ASV03"
	case 0x39:
		return "ELE400508A2LSV03"
	case 0x3A:
		return "ELE400535A2ASV04"
	case 0x3B:
		return "ELE400508A2ASV04"
	case 0x3C:
		return "ELE400535A2LSV04"
	case 0x3D:
		return "ELE400508A2LSV04"
	case 0x3E:
		return "ELE400535A2ASV05"
	case 0x3F:
		return "ELE400508A2ASV05"
	case 0x40:
		return "ELE400535A2LSV05"
	case 0x41:
		return "ELE400508A2LSV05"
	case 0x42:
		return "ELE400535A2AEV03"
	case 0x43:
		return "ELE400535A2LEV03"
	case 0x44:
		return "ELE410535A2ASV01"
	case 0x45:
		return "ELE410508A2ASV01"
	case 0x46:
		return "ELE410535A2LSV01"
	case 0x47:
		return "ELE410508A2LSV01"
	default:
		return "Unknown Product Code"
	}
}

func (elt *ElectrolyserStatusType) GetWarnings() []string {
	var s []string
	for w := uint16(0); w < elt.Warnings.count; w++ {
		s = append(s, decodeMessage(elt.Warnings.codes[w]))
	}
	return s
}

func (elt *ElectrolyserStatusType) GetErrors() []string {
	var s []string
	for w := uint16(0); w < elt.Errors.count; w++ {
		s = append(s, decodeMessage(elt.Errors.codes[w]))
	}
	return s
}

func (elt *ElectrolyserStatusType) ClearWarnings() {
	elt.Warnings.count = 0
	for idx := range elt.Warnings.codes {
		elt.Warnings.codes[idx] = 0
	}
}

func (elt *ElectrolyserStatusType) ClearErrors() {
	elt.Errors.count = 0
	for idx := range elt.Errors.codes {
		elt.Errors.codes[idx] = 0
	}
}

func (elt *ElectrolyserStatusType) Clear() {
	elt.ClearWarnings()
	elt.ClearErrors()
	elt.H2Flow = 0
	elt.HydrogenPressureTooHigh = false
	elt.InnerH2Pressure = 0
	elt.OuterH2Pressure = 0
	elt.StackCurrent = 0
	elt.StackVoltage = 0
	elt.State = 0
	elt.SystemState = 0
}

func (elt *ElectrolyserStatusType) PowerRelayEnergised() bool {
	return Relays.GetRelay(elt.PowerRelay)
}

func (eljst *ElectrolyserJSONStatusType) load(elt ElectrolyserStatusType) {
	eljst.Device = elt.Device
	eljst.Name = elt.Name
	eljst.Powered = elt.Powered
	eljst.PowerRelayEnergised = elt.PowerRelayEnergised()
	eljst.Model = elt.Model
	eljst.Serial = elt.Serial
	eljst.SystemState = elt.SystemState
	if elt.elm.StackSerialNumber != "" {
		eljst.StackSerialNumber = elt.StackSerialNumber + "(" + elt.elm.StackSerialNumber + ")"
	} else {
		eljst.StackSerialNumber = elt.StackSerialNumber
	}
	eljst.StackStartStopCycles = int32(elt.StackStartStopCycles) - elt.elm.RestartCyclesOffset
	//	log.Printf("Stack run time = %d offset = %d total = %d", elt.StackTotalRunTime, elt.elm.StackTimeOffset, int32(elt.StackTotalRunTime)-elt.elm.StackTimeOffset)
	eljst.StackTotalRunTime = int32(elt.StackTotalRunTime) - elt.elm.StackTimeOffset
	eljst.StackTotalProduction = elt.StackTotalProduction - jsonFloat32(elt.elm.StackProductionOffset)
	eljst.H2Flow = elt.H2Flow
	eljst.ProductCode = elt.GetProductCode()
	eljst.State = elt.State
	eljst.ElectrolyteLevel = elt.ElectrolyteLevel
	eljst.ElectrolyteTankPressureTooHigh = elt.ElectrolyteTankPressureTooHigh
	eljst.HydrogenPressureTooHigh = elt.HydrogenPressureTooHigh
	eljst.DownstreamHighTemperature = elt.DownstreamHighTemperature
	eljst.ElectronicCompartmentHighTemp = elt.ElectronicCompartmentHighTemp
	eljst.VeryLowElectrolyteTemp = elt.VeryLowElectrolyteTemp
	eljst.ChassisWaterPresent = elt.ChassisWaterPresent
	eljst.DryContact = elt.DryContact
	eljst.ElectrolyteCoolerFanSpeed = elt.ElectrolyteCoolerFanSpeed
	eljst.AirCirculationFanSpeed = elt.AirCirculationFanSpeed
	eljst.ElectronicCompartmentCoolingFanSpeed = elt.ElectronicCompartmentCoolingFanSpeed
	eljst.ElectrolyteFlowMeter = elt.ElectrolyteFlowMeter
	eljst.StackCurrent = elt.StackCurrent
	eljst.StackVoltage = elt.StackVoltage
	eljst.InnerH2Pressure = elt.InnerH2Pressure
	eljst.OuterH2Pressure = elt.OuterH2Pressure
	eljst.WaterPressure = elt.WaterPressure
	eljst.ElectrolyteTemp = elt.ElectrolyteTemp
	eljst.DownstreamTemp = elt.DownstreamTemp
	eljst.CurrentProductionRate = elt.CurrentProductionRate
	eljst.MaxTankPressure = elt.MaxTankPressure
	eljst.RestartPressure = elt.RestartPressure
	eljst.DryerNetworkEnabled = elt.DryerNetworkEnabled
	eljst.DryerFailure = elt.DryerFailure
	eljst.Warnings = elt.GetWarnings()
	eljst.Errors = elt.GetErrors()
	eljst.Dryer = elt.Dryer
	eljst.IP = elt.IP
	eljst.PowerRelay = elt.PowerRelay
	eljst.Enabled = elt.Enabled
}
