package main

import (
	"cmp"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"log"
	"net"
	"os"
	"slices"
	"strings"
	"time"
)

/*
*********************************************************
CAN bus must be enabled before this service can be started
https://www.pragmaticlinux.com/2021/07/automatically-bring-up-a-socketcan-interface-on-boot/
*/
var (
	canBus                *CANBus
	CANInterface          string
	WebPort               string
	databaseServer        string
	databasePort          string
	databaseName          string
	databaseLogin         string
	databasePassword      string
	Relays                RelaysType
	Outputs               DigitalOutputsType
	Inputs                DigitalInputsType
	AnalogInputs          AnalogInputsType
	ACMeasurements        [4]ACMeasurementsType
	DCMeasurements        [4]DCMeasurementsType
	Electrolysers         ElectrolysersType
	PowerControl          []*PowerControlType
	jsonSettings          string
	currentSettings       *SettingsType
	webFiles              string
	pDB                   *sql.DB
	ElectrolyserStatement *sql.Stmt
	DryerStatement        *sql.Stmt
	ACStatement           *sql.Stmt
	DCStatement           *sql.Stmt
	PowerStatement        *sql.Stmt
	FuelCell              PANFuelCell
	logFile               *os.File
	logFileName           string
	callLogging           = false
	debugOutput           = false
	store                 *sessions.CookieStore
)

func connectToDatabase() (*sql.Stmt, *sql.DB, error) {
	if pDB != nil {
		if closeErr := pDB.Close(); closeErr != nil {
			log.Println(closeErr)
		}
		pDB = nil
	}
	// Set the time zone to Local to correctly record times
	var sConnectionString = databaseLogin + ":" + databasePassword + "@tcp(" + databaseServer + ":" + databasePort + ")/" + databaseName + "?loc=Local"

	db, err := sql.Open("mysql", sConnectionString)
	if err != nil {
		log.Println(err)
		return nil, nil, err
	}
	err = db.Ping()
	if err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Println(closeErr)
		}
		pDB = nil
		return nil, nil, err
	}
	PowerStatement, err = db.Prepare(InsertPowerSQL)
	if err != nil {
		log.Println(err)
		if closeErr := db.Close(); closeErr != nil {
			log.Println(closeErr)
		}
		return nil, nil, err
	}
	logAnalog, err := db.Prepare("INSERT INTO firefly.IOValues(a0, a1, a2, a3, a4, a5, a6, a7, vref, cpuTemp, rawCpuTemp, temperature, inputs, outputs, relays) VALUES  (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		log.Println(err)
		if closeErr := db.Close(); closeErr != nil {
			log.Println(closeErr)
		}
		return nil, nil, err
	}
	const fuelCellLogStatement = `INSERT INTO firefly.PANFuelCell (
		StackCurrent, StackVoltage, OutputVoltage, OutputCurrent,
        CoolantInTemp, CoolantOutTemp, CoolantPressure, CoolantFanSpeed, CoolantPumpSpeed, CoolantPumpVolts, CoolantPumpAmps,
		InsulationResistance, HydrogenPressure, AirPressure, AirinletTemp,
		AmbientTemp,  AirFlow, HydrogenConcentration, DCDCTemp, DCDCInVolts,
		DCDCOutVolts, DCDCInAmps, DCDCOutAmps, MinCellVolts, MaxCellVolts,
		AvgCellVolts, IdxMaxCell, IdxMinCell, RunStage, FaultLevel, 
		Cell00Volts	, Cell01Volts	, Cell02Volts	, Cell03Volts	, Cell04Volts,
		Cell05Volts	, Cell06Volts	, Cell07Volts	, Cell08Volts	, Cell09Volts,
		Cell10Volts	, Cell11Volts	, Cell12Volts	, Cell13Volts	, Cell14Volts,
		Cell15Volts	, Cell16Volts	, Cell17Volts	, Cell18Volts	, Cell19Volts,
		Cell20Volts	, Cell21Volts	, Cell22Volts	, Cell23Volts	, Cell24Volts,
		Cell25Volts	, Cell26Volts	, Cell27Volts	, Cell28Volts	, Cell29Volts,
		Cell30Volts	, Cell31Volts	, Alarms		, PowerRequested, PowerDelivered,
        MaxBattVolts, MinBattVolts, PowerModeState)
        VALUES (
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
			?,?,?,?,?,
            ?,?,?)`

	if pStmt, err := db.Prepare(fuelCellLogStatement); err != nil {
		log.Println("Failed to prepare the fuel cell log statement -", err)
		return logAnalog, nil, err
	} else {
		dbRecord.stmt = pStmt
	}

	ElectrolyserStatement, err = db.Prepare(ElectrolyserInsertStatement)
	if err != nil {
		return logAnalog, nil, err
	}
	DryerStatement, err = db.Prepare(DryerInsertStatement)
	if err != nil {
		return logAnalog, nil, err
	}
	ACStatement, err = db.Prepare(ACValuesInsertStatement)
	if err != nil {
		return logAnalog, nil, err
	}
	DCStatement, err = db.Prepare(DCValuesInsertStatement)
	if err != nil {
		return logAnalog, nil, err
	}

	return logAnalog, db, err
}

func ConnectCANBus() *CANBus {
	if Bus, err := NewCANBus(CANInterface); err != nil {
		log.Println(err)
		return nil
	} else {
		return Bus
	}
}

func init() {
	flag.StringVar(&CANInterface, "can", "can0", "CAN Interface Name")
	flag.StringVar(&WebPort, "WebPort", "20080", "Web port")
	flag.StringVar(&jsonSettings, "jsonSettings", "/etc/FireflyService.json", "JSON file containing the system control parameters")
	flag.StringVar(&webFiles, "webFiles", "/FireflyService/web", "Path to the WEB files location")
	flag.StringVar(&databaseServer, "sqlServer", "localhost", "MySQL Server")
	flag.StringVar(&databaseName, "database", "firefly", "Database name")
	flag.StringVar(&databaseLogin, "dbUser", "FireflyService", "Database login user name")
	flag.StringVar(&databasePassword, "dbPassword", "logger", "Database user password")
	flag.StringVar(&databasePort, "dbPort", "3306", "Database port")
	flag.StringVar(&logFileName, "logfile", "/var/log/FireflyService", "Name of the log file")
	flag.Parse()

	// open log file
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panic(err)
	}
	// set log out put
	log.SetOutput(logFile)
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	Relays.InitRelays()
	Outputs.InitOutputs()
	Inputs.InitInputs()
	AnalogInputs.InitAnalogInputs()

	log.Println("Loading the settings")
	currentSettings = NewSettings()
	if err := currentSettings.LoadSettings(jsonSettings); err != nil {
		log.Print(err)
	}
	if len(currentSettings.SessionKey) == 0 {
		currentSettings.SessionKey = string(securecookie.GenerateRandomKey(32))
		if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
			log.Println(err)
		}
	}
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

	log.Println("Connecting to can bus")
	canBus = ConnectCANBus()
	FuelCell.init(canBus)
	if err := FuelCell.setTargetBattHigh(currentSettings.FuelCellSettings.HighBatterySetpoint); err != nil {
		log.Print(err)
	}
	if err := FuelCell.setTargetBattLow(currentSettings.FuelCellSettings.LowBatterySetpoint); err != nil {
		log.Print(err)
	}
	if err := FuelCell.setTargetPower(currentSettings.FuelCellSettings.PowerSetting); err != nil {
		log.Print(err)
	}

	log.Println("Starting the WEB site.")
	go setUpWebSite()
	go setUpLocalWebSocket()
}

// ClientLoop ticks every second and logs values to the database. It also broadcasts the values to any registered web socket clients.
func ClientLoop() {
	// Set up the sync to send data to waiting web socket clients every second
	broadcastTime := time.NewTicker(time.Second)

	for {
		select {
		case <-broadcastTime.C:
			{
				// Make sure the CAN bus is up and running
				if canBus == nil || canBus.bus == nil {
					if debugOutput {
						log.Println("Adding the CAN bus monitor")
					}
					if canBus != nil {
						//	if canBus.bus != nil {
						//		if err := canBus.bus.SetDown(); err != nil {
						//			log.Println(err)
						//		}
						//	}
						canBus = nil
						FuelCell.bus = nil
					}
					canBus = ConnectCANBus()
					FuelCell.bus = canBus
				}
				// Get the full status
				if bytes, err := getJsonStatus(false); err != nil {
					log.Print("Error marshalling the full data - ", err)
				} else {
					select {
					// Send the full status message
					case pool.Broadcast <- WSMessageType{
						data:    bytes,
						service: wsFull,
						device:  "",
					}:
					default:
						log.Println("Channel would block!")
					}
				}

				if bytes, err := FuelCell.GetStatusAsJSON(); err != nil {
					log.Print("Error marshalling the fuelcell data - ", err)
				} else {
					select {
					// Send the fuel cell only status
					case pool.Broadcast <- WSMessageType{
						data:    bytes,
						service: wsFuelCell,
						device:  "",
					}:
					default:
						log.Println("Channel would block!")
					}
				}

				for _, el := range Electrolysers.Arr {
					if bytes, err := el.getJsonStatus(); err != nil {
						log.Print("Error marshalling the electrolyser data - ", err)
					} else {
						select {
						case pool.Broadcast <- WSMessageType{
							data:    bytes,
							service: wsElectrolyser,
							device:  strings.ToLower(el.status.Name),
						}:
						default:
							if debugOutput {
								log.Println("Channel would block!")
							}
						}
					}
				}
			}
		}
	}
}

var WaterDumpHoldoff time.Time

func DatabaseLogger() {
	var (
		err       error
		logAnalog *sql.Stmt
	)
	logAnalog, pDB, err = connectToDatabase()
	if err != nil {
		log.Println(err)
	}
	loggingTime := time.NewTicker(time.Second)

	for {
		select {
		case <-loggingTime.C:
			if pDB == nil {
				log.Println("Reconnect to the database")
				logAnalog, pDB, err = connectToDatabase()
				if err != nil {
					log.Println(err)
				}
			}
			if pDB != nil {
				//if debugOutput {
				//	log.Println("Logging data")
				//}
				rawTemp, cpuTemp := AnalogInputs.GetCPUTemperature()
				if _, err := logAnalog.Exec(AnalogInputs.GetRawInput(0), AnalogInputs.GetRawInput(1), AnalogInputs.GetRawInput(2), AnalogInputs.GetRawInput(3),
					AnalogInputs.GetRawInput(4), AnalogInputs.GetRawInput(5), AnalogInputs.GetRawInput(6), AnalogInputs.GetRawInput(7),
					AnalogInputs.GetVREF(), cpuTemp, rawTemp, AnalogInputs.GetTemperature(),
					Inputs.GetAllInputs(), Outputs.GetAllOutputs(), Relays.GetAllRelays()); err != nil {
					log.Println(err)
					if closeErr := pDB.Close(); closeErr != nil {
						log.Println(closeErr)
					}
					pDB = nil
					logAnalog = nil
				}
				if err := dbRecord.saveToDatabase(); err != nil {
					log.Println(err)
					if closeErr := pDB.Close(); closeErr != nil {
						log.Println(closeErr)
					}
					pDB = nil
					dbRecord.stmt = nil
				}
				if _, err := ACStatement.Exec(ACMeasurements[0].getVolts(), ACMeasurements[0].getAmps(), ACMeasurements[0].getPower(), ACMeasurements[0].getFrequency(), ACMeasurements[0].getPowerFactor(),
					ACMeasurements[1].getVolts(), ACMeasurements[1].getAmps(), ACMeasurements[1].getPower(), ACMeasurements[1].getFrequency(), ACMeasurements[1].getPowerFactor(),
					ACMeasurements[2].getVolts(), ACMeasurements[2].getAmps(), ACMeasurements[2].getPower(), ACMeasurements[2].getFrequency(), ACMeasurements[2].getPowerFactor(),
					ACMeasurements[3].getVolts(), ACMeasurements[3].getAmps(), ACMeasurements[3].getPower(), ACMeasurements[3].getFrequency(), ACMeasurements[3].getPowerFactor()); err != nil {
					log.Println(err)
					if closeErr := pDB.Close(); closeErr != nil {
						log.Println(closeErr)
					}
					pDB = nil
					ACStatement = nil
				}
				if _, err := DCStatement.Exec(DCMeasurements[0].getVolts(), DCMeasurements[0].getAmps(),
					DCMeasurements[1].getVolts(), DCMeasurements[1].getAmps(),
					DCMeasurements[2].getVolts(), DCMeasurements[2].getAmps(),
					DCMeasurements[3].getVolts(), DCMeasurements[3].getAmps()); err != nil {
					log.Println(err)
					if closeErr := pDB.Close(); closeErr != nil {
						log.Println(closeErr)
					}
					pDB = nil
					DCStatement = nil
				}
				for _, pc := range PowerControl {
					pc.logData(PowerStatement)
				}
			} else {
				log.Println("Database is not connected")
			}
			// Check for water dump.
			if currentSettings.WaterDumpAction == ElConductivityHigh {
				on := false
				_, conductivity := AnalogInputs.GetInput(7)
				if conductivity > float32(currentSettings.MaximumConductivity) {
					for _, el := range Electrolysers.Arr {
						if el.IsSwitchedOn() {
							on = true
						}
					}
				}
				// If there is an electrolyser that is running and the holdoff time has expired
				if on && !WaterDumpHoldoff.After(time.Now()) {
					go TurnOnWaterDumpValve()
					WaterDumpHoldoff = time.Now().Add(time.Minute * 30)
				}
			}

			FuelCell.checkOffLine()
		}
	} // Log data to the database
}

/*
CANHeartbeat sends CAN packets to the fuel cell
*/
func CANHeartbeat() {
	heartbeatTime := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-heartbeatTime.C:
			{
				if canBus != nil {
					Relays.UpdateRelays() // Heartbeat to the FireflyService board. If we don't send this the board will turn all relays off after about a minute.
					if err := canBus.SetFlags(currentSettings.getModbusFlags(), 0, 0, 0, 0, 0, 0, 0); err != nil {
						log.Println(err)
					}
					if err := FuelCell.updateOutput(); err != nil {
						log.Print(err)
					}
					if err := FuelCell.updateSettings(); err != nil {
						log.Print(err)
					}
				} else {
					log.Println("No CAN bus available")
				}
				heartbeat++
			}
		}
	}
}

// ElectrolyserLoop reads the electrolysers every two seconds when they are powered on
//
//	and writes the data collected to the database.
func ElectrolyserLoop() {
	electrolyserHeartbeat := time.NewTicker(time.Second * 2)
	for {
		select {
		case <-electrolyserHeartbeat.C:
			{
				if !currentSettings.acquiringElectrolysers {
					gotDryer := false
					for _, el := range Electrolysers.Arr {
						// Is this electrolyser switched on?
						if el.IsSwitchedOn() {
							if !gotDryer {
								el.hasDryer = true                                       // First active electrolyser gets to control the dryer
								gotDryer = true                                          // We found an active electrolyser, so we have the dryer.
								Relays.SetRelay(uint8(currentSettings.DryerRelay), true) // Make sure the dryer is powered on
								// after 30 seconds, start the dryer error monitor
								time.AfterFunc(time.Second*30, el.MonitorDryerErrors)
							} else {
								el.hasDryer = false // We already have the dryer so this electrolyser can skip it.
							}
							// We must have a valid IP address
							if !el.status.IP.Equal(net.IPv4zero) {

								// Get the values for this electrolyser
								if debugOutput {
									log.Println("polling electrolyser ", el.GetIPString())
								}
								if err := el.ReadValues(); err != nil {
									log.Print(err)
								} else {
									// Write the data to the database
									if err := el.RecordData(ElectrolyserStatement); err != nil {

										log.Println(err)
									}
									if el.hasDryer {
										if _, err := DryerStatement.Exec(el.GetDryerTemp(0),
											el.GetDryerTemp(1),
											el.GetDryerTemp(2),
											el.GetDryerTemp(3),
											el.status.Dryer.InputPressure,
											el.status.Dryer.OutputPressure,
											el.GetDryerErrorText(),
											el.GetDryerWarningText()); err != nil {

											log.Println(err)
										}
									}
								}
							} else {
								log.Println("electrolyser has no ip address")
								if !el.CheckConnected() {
									log.Println("cannot find electrolyser....")
								} else {
									log.Println("Connected")
								}
							}
						} else {
							el.status.ClearErrors()
							el.status.ClearWarnings()
						}
					}
					// If we did not find an active electrolyser, turn off the dryer and resort the electrolysers
					if !gotDryer {
						Relays.SetRelay(uint8(currentSettings.DryerRelay), false)
						slices.SortStableFunc(Electrolysers.Arr[:], func(a *ElectrolyserType, b *ElectrolyserType) int {
							return cmp.Compare(a.status.StackTotalRunTime, b.status.StackTotalRunTime)
						})
					}
				}
			}
		}
	}
}

// ElectrolyserPumpManagement controls the pump relay.
func ElectrolyserPumpManagement() {
	pumpTicker := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-pumpTicker.C:
			{
				// Only if we have a relay defined
				if currentSettings.CoolingPumpRelay < 16 {
					tMax := 0.0
					for _, el := range Electrolysers.Arr {
						if el.IsSwitchedOn() {
							if float64(el.status.ElectrolyteTemp) > tMax {
								tMax = float64(el.status.ElectrolyteTemp)
							}
						}
					}
					if tMax > float64(currentSettings.CoolingPumpStartTemperature) {
						Relays.SetRelay(currentSettings.CoolingPumpRelay, true)
					} else if tMax < float64(currentSettings.CoolingPumpStopTemperature) {
						Relays.SetRelay(currentSettings.CoolingPumpRelay, false)
					}
				}
			}
		}
	}
}

// LeakDetection sets up a timer to check the hydrogen sensor and conductivity every second.
//
//	If five consecutive readings are higher than the threshold the electrolysers are powered down.
func LeakDetection() {
	leakTimer := time.NewTicker(time.Second)
	alarmCount := 0
	conductivityAlarmCount := 0

	for {
		select {
		case <-leakTimer.C:
			{
				if AnalogInputs.Inputs[currentSettings.GasDetectorInput].Raw > currentSettings.GasDetectorThreshold &&
					currentSettings.GasDetectorThreshold > 0 {
					alarmCount += 1
				} else {
					alarmCount -= 1
				}
				if alarmCount > 5 {
					for _, el := range Electrolysers.Arr {
						Relays.SetRelay(el.powerRelay, false)
						SystemAlarms.H2DetectedAlarm = true
					}
					alarmCount = 5
				} else if alarmCount <= 0 {
					alarmCount = 0
					SystemAlarms.H2DetectedAlarm = false
				}
				if AnalogInputs.Inputs[7].Value > currentSettings.WaterQualityAlarm && currentSettings.WaterQualityAlarm > 0 {
					conductivityAlarmCount += 1
				} else {
					conductivityAlarmCount -= 1
				}
				if conductivityAlarmCount > 5 {
					for _, el := range Electrolysers.Arr {
						Relays.SetRelay(el.powerRelay, false)
					}
					conductivityAlarmCount = 5
					SystemAlarms.ConductivityAlarm = true
				} else if conductivityAlarmCount <= 0 {
					conductivityAlarmCount = 0
					SystemAlarms.ConductivityAlarm = false
				}
			}

		}
	}
}

func DatabaseMaintenance() {
	const FuelCellDataArchiveStatement = `INSERT INTO firefly.PANFuelCell_Archive (
	StackCurrent, StackVoltage, CoolantInTemp,CoolantOutTemp, logged,
	OutputVoltage, OutputCurrent, CoolantFanSpeed, CoolantPumpSpeed, CoolantPumpVolts, CoolantPumpAmps,
	InsulationResistance, HydrogenPressure, AirPressure, CoolantPressure, AirinletTemp, AmbientTemp, AirFlow, HydrogenConcentration,
	DCDCTemp, DCDCInVolts, DCDCOutVolts, DCDCInAmps, DCDCOutAmps,
	MinCellVolts, MaxCellVolts,
	AvgCellVolts, IdxMaxCell, IdxMinCell, RunStage,
	FaultLevel, PowerModeState,
	Cell00Volts, Cell01Volts, Cell02Volts, Cell03Volts, Cell04Volts, Cell05Volts, Cell06Volts, Cell07Volts, Cell08Volts, Cell09Volts,
	Cell10Volts, Cell11Volts, Cell12Volts, Cell13Volts, Cell14Volts, Cell15Volts, Cell16Volts, Cell17Volts, Cell18Volts, Cell19Volts,
	Cell20Volts, Cell21Volts, Cell22Volts, Cell23Volts, Cell24Volts, Cell25Volts, Cell26Volts, Cell27Volts, Cell28Volts, Cell29Volts,
	Cell30Volts, Cell31Volts,
	Alarms, PowerRequested, PowerDelivered)
SELECT
	avg(StackCurrent), avg(StackVoltage), avg(CoolantInTemp), avg(CoolantOutTemp), round(min(logged)),
	avg(OutputVoltage), avg(OutputCurrent), avg(CoolantFanSpeed), avg(CoolantPumpSpeed), avg(CoolantPumpVolts), avg(CoolantPumpAmps),
	avg(InsulationResistance), avg(HydrogenPressure), avg(AirPressure), avg(CoolantPressure), avg(AirinletTemp), avg(AmbientTemp),
	avg(AirFlow), avg(HydrogenConcentration),
	avg(DCDCTemp), avg(DCDCInVolts), avg(DCDCOutVolts), avg(DCDCInAmps), avg(DCDCOutAmps),
	avg(MinCellVolts), avg(MaxCellVolts),
	avg(AvgCellVolts), avg(IdxMaxCell), avg(IdxMinCell), avg(RunStage),
	avg(FaultLevel),avg(PowerModeState),
	avg(Cell00Volts), avg(Cell01Volts), avg(Cell02Volts), avg(Cell03Volts), avg(Cell04Volts),
	avg(Cell05Volts), avg(Cell06Volts), avg(Cell07Volts), avg(Cell08Volts), avg(Cell09Volts),
	avg(Cell10Volts), avg(Cell11Volts), avg(Cell12Volts), avg(Cell13Volts), avg(Cell14Volts),
	avg(Cell15Volts), avg(Cell16Volts), avg(Cell17Volts), avg(Cell18Volts), avg(Cell19Volts),
	avg(Cell20Volts), avg(Cell21Volts), avg(Cell22Volts), avg(Cell23Volts), avg(Cell24Volts),
	avg(Cell25Volts), avg(Cell26Volts), avg(Cell27Volts), avg(Cell28Volts), avg(Cell29Volts),
	avg(Cell30Volts), avg(Cell31Volts),
	min(Alarms), avg(PowerRequested), avg(PowerDelivered)
   FROM firefly.PANFuelCell where logged < DATE(DATE_ADD( now(), interval -1 month))
   group by UNIX_TIMESTAMP(logged) DIV 60`
	const FuelCellDataCleanupStatement = `delete FROM firefly.PANFuelCell where logged < DATE(DATE_ADD( now(), interval -1 month))`

	const IOValuesArchiveStatement = `INSERT INTO firefly.IOValues_Archive (logged, a0, a1, a2, a3, a4, a5, a6, a7, inputs, outputs, relays, vref, cpuTemp, rawCpuTemp, temperature)
SELECT min(logged), avg(a0), avg(a1), avg(a2), avg(a3), avg(a4), avg(a5), avg(a6), avg(a7), 
       min(inputs), min(outputs), min(relays), avg(vref), avg(cpuTemp), avg(rawCpuTemp), avg(temperature)
      FROM firefly.IOValues
      WHERE logged < DATE(DATE_ADD( now(), interval -1 month))
      GROUP BY UNIX_TIMESTAMP(logged) DIV 60`

	const IOValuesCleanupStatement = `DELETE FROM IOValues WHERE logged < DATE(DATE_ADD( now(), interval -1 month))`

	const ACValuesArchiveStatement = `INSERT INTO firefly.ACValues_Archive (logged, A_volts, A_amps, A_watts, A_hertz, A_powerFactor
			, B_volts, B_amps, B_watts, B_hertz, B_powerFactor
			, C_volts, C_amps, C_watts, C_hertz, C_powerFactor
			, D_volts, D_amps, D_watts, D_hertz, D_powerFactor)
       SELECT min(logged), avg(A_volts), avg(A_amps), avg(A_watts), avg(A_hertz), avg(A_powerFactor), 
        avg(B_volts), avg(B_amps), avg(B_watts), avg(B_hertz), avg(B_powerFactor), 
        avg(C_volts), avg(C_amps), avg(C_watts), avg(C_hertz), avg(C_powerFactor), 
        avg(D_volts), avg(D_amps), avg(D_watts), avg(D_hertz), avg(D_powerFactor)
  FROM firefly.ACValues
 WHERE logged < DATE(DATE_ADD( now(), INTERVAL -1 MONTH))
 GROUP BY UNIX_TIMESTAMP(logged) DIV 60;`

	const ACValuesCleanupStatement = `DELETE FROM ACValues WHERE logged < DATE(DATE_ADD( now(), INTERVAL -1 MONTH))`

	const DCValuesArchiveStatement = `INSERT INTO firefly.DCValues_Archive (logged, A_volts, A_amps, B_volts, B_amps, C_volts, C_amps, D_volts, D_amps)
SELECT min(logged), avg(A_volts), avg(A_amps), avg(B_volts), avg(B_amps), avg(C_volts), avg(C_amps), avg(D_volts), avg(D_amps)
  FROM firefly.DCValues
WHERE logged < DATE(DATE_ADD( now(), INTERVAL -1 MONTH))
 GROUP BY UNIX_TIMESTAMP(logged) DIV 60;`

	const DCValuesCleanupStatement = `DELETE FROM firefly.DCValues WHERE logged < DATE(DATE_ADD( now(), INTERVAL -1 MONTH));`

	const PowerArchiveStatement = `INSERT INTO firefly.Power_Archive (logged, volts, amps, soc, frequency, solar)
  SELECT min(logged), avg(volts), avg(amps), avg(soc), avg(frequency), avg(solar)
    FROM firefly.Power
WHERE logged < DATE(DATE_ADD( now(), INTERVAL -1 MONTH))
 GROUP BY UNIX_TIMESTAMP(logged) DIV 60;`

	const PowerCleanupStatement = `DELETE FROM firefly.Power WHERE logged < DATE(DATE_ADD( now(), INTERVAL -1 MONTH));`

	maintenanceTimer := time.NewTicker(time.Hour)

	for {
		select {
		case <-maintenanceTimer.C:
			{
				// Fuel Cell
				if trans, err := pDB.Begin(); err != nil {
					log.Print("Fuel Cell Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(FuelCellDataArchiveStatement); err != nil {
						log.Println("Fuel Cell Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back Fuel Cell Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(FuelCellDataCleanupStatement); err != nil {
							log.Println("Fuel Cell Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back Fuel Cell Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("Fuel Cell Archive transaction failed - ", err)
							}
						}
					}

				}
				// IOValues
				if trans, err := pDB.Begin(); err != nil {
					log.Print("Fuel Cell Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(IOValuesArchiveStatement); err != nil {
						log.Println("IO Values Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back IOValues Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(IOValuesCleanupStatement); err != nil {
							log.Println("IOValues Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back IOValues Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("IOValues Archive transaction failed - ", err)
							}
						}
					}
				}
				// Electrolyser
				if trans, err := pDB.Begin(); err != nil {
					log.Print("Electrolyser Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(ElectrolyserArchiveStatement); err != nil {
						log.Println("Electrolyser Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back Electrolyser Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(ElectrolyserCleanupStatement); err != nil {
							log.Println("Electrolyser Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back Electrolyser Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("Electrolyser Archive transaction failed - ", err)
							}
						}
					}
				}
				// Dryer
				if trans, err := pDB.Begin(); err != nil {
					log.Print("Dryer Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(DryerArchiveStatement); err != nil {
						log.Println("Dryer Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back Dryer Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(DryerCleanupStatement); err != nil {
							log.Println("Dryer Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back Dryer Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("Dryer Archive transaction failed - ", err)
							}
						}
					}
				}
				// DCValues
				if trans, err := pDB.Begin(); err != nil {
					log.Print("DCValues Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(DCValuesArchiveStatement); err != nil {
						log.Println("DCValues Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back DCValues Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(DCValuesCleanupStatement); err != nil {
							log.Println("DCValues Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back DCValues Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("DCValues Archive transaction failed - ", err)
							}
						}
					}
				}
				// ACValues
				if trans, err := pDB.Begin(); err != nil {
					log.Print("ACValues Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(ACValuesArchiveStatement); err != nil {
						log.Println("ACValues Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back ACValues Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(ACValuesCleanupStatement); err != nil {
							log.Println("ACValues Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back ACValues Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("ACValues Archive transaction failed - ", err)
							}
						}
					}
				}
				// Power
				if trans, err := pDB.Begin(); err != nil {
					log.Print("Power Archive transaction failed - ", err)
				} else {
					if _, err := trans.Exec(PowerArchiveStatement); err != nil {
						log.Println("Power Archive failed - ", err)
						if err := trans.Rollback(); err != nil {
							log.Print("Failed to roll back Power Archive transaction - ", err)
						}
					} else {
						if _, err := trans.Exec(PowerCleanupStatement); err != nil {
							log.Println("Power Cleanup failed - ", err)
							if err := trans.Rollback(); err != nil {
								log.Print("Failed to roll back Power Archive transaction - ", err)
							}
						} else {
							if err := trans.Commit(); err != nil {
								log.Print("Power Archive transaction failed - ", err)
							}
						}
					}
				}
			}
		}
	}
}

func main() {
	defer func() {
		if err := logFile.Close(); err != nil {
			_, _ = fmt.Fprint(os.Stderr, err)
		}
	}()

	go ElectrolyserLoop()
	go CANHeartbeat()
	go DatabaseLogger()
	go LeakDetection()
	go ElectrolyserPumpManagement()
	go DatabaseMaintenance()
	ClientLoop()
}
