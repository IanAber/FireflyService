package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type APIType struct {
	url         string
	params      string
	method      string
	description string
}

var APICalls []APIType

func RegisterWebSiteAPI(router *mux.Router, url string, params string, method string, description string, f func(http.ResponseWriter, *http.Request)) {
	router.HandleFunc(url, f).Methods(method)
	APICalls = append(APICalls, APIType{url: url, params: params, method: method, description: description})
}

func buildAPIDocumentationPage(w http.ResponseWriter, _ *http.Request) {
	if _, err := fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
    <head>
        <meta charset="UTF-8">
        <title>ElektrikGreen FireflyIO API</title>
        <link rel="stylesheet" type="text/css" href="css/fireflyio.css" />
        <link rel="stylesheet" type="text/css" href="css/api.css" />
        <script type="text/ecmascript" src="scripts/utils.js"></script>
    </head>
    <body onload="PopulateTitle()">
        <header class="header">
            <h1>
                <span class="system" id="system">Loading...</span>
                <img id="logo" class="logo" src="images/logo.png" alt="ElektrikGreen Logo"/>
            </h1>
        </header>
        <div>
            <table>
                <tr><th colspan="3">Web Service Calls</th></tr>`); err != nil {
		log.Print(err)
		return
	}
	for _, call := range APICalls {
		if _, err := fmt.Fprintf(w, `<tr><td class="uri">%s`, call.url); err != nil {
			log.Print(err)
			return
		}
		if call.params != "" {
			if _, err := fmt.Fprintf(w, "?%s", call.params); err != nil {
				log.Print(err)
				return
			}
		}
		if _, err := fmt.Fprintf(w, `</td><td class="verb">%s</td><td class="description">%s</td></tr>`, call.method, call.description); err != nil {
			log.Print(err)
			return
		}
	}
	if _, err := fmt.Fprint(w, `            </table>
        </div>
    </body>
</html>`); err != nil {
		log.Print(err)
	}
}

func RegisterWebSiteAPICalls(router *mux.Router) {

	// Register with the WebSocket which will then push a JSON payload with data to keep the displayed data up to date. No polling necessary.
	RegisterWebSiteAPI(router, "/ws/fuelcell", "", "GET", "Register with the WebSocket which will then push a JSON payload with the fuel cell data to keep the displayed data up to date. No polling necessary.", startFuelCellWebSocket)
	RegisterWebSiteAPI(router, "/ws/electrolyser/{electrolyser}", "", "GET", `Register with the WebSocket which will then push a JSON payload with the electrolyser data to keep the displayed data up to date. No polling necessary.`, startElectrolyserWebSocket)
	RegisterWebSiteAPI(router, "/ws", "", "GET", `Register with the WebSocket which will then push a <a href="/getStatus">JSON payload</a> with data to keep the displayed data up to date. No polling necessary.`, startDataWebSocket)

	RegisterWebSiteAPI(router, "/registration", "", "GET", "Basic user management screen", userManagement)
	RegisterWebSiteAPI(router, "/Logout.html", "", "GET", "Log out the current session", Logout)
	RegisterWebSiteAPI(router, "/addUser", "", "POST", "Add a user", addUser)
	RegisterWebSiteAPI(router, "/deleteUser", "", "POST", "Delete a registered user", deleteUser)
	RegisterWebSiteAPI(router, "/generateKey", "", "GET", "Generate a new API key", generateKey)
	RegisterWebSiteAPI(router, "/setRelay/{relay}/{on}", "", "PUT", "Turn on or off (1,0,on,off,true,false) a relay by name or number", setRelay)
	RegisterWebSiteAPI(router, "/setOutput/{output}/{on}", "", "PUT", "Turn on or off (1,0,on,off,true,false) a digital output by name or number", setOutput)
	RegisterWebSiteAPI(router, "/setButton/{button}/{on}", "", "PUT", "Turn on or off (1,0,on,off,true,false) a button by name or number", setButton)
	RegisterWebSiteAPI(router, "/getSettings", "", "GET", "Get the current system settings", getSettings)
	RegisterWebSiteAPI(router, "/setSettings", "", "POST", "Set the system settings", setSettings)
	RegisterWebSiteAPI(router, "/getStatus", "", "GET", `Get the full system <a href="/getStatus">status</a>`, getStatus)
	RegisterWebSiteAPI(router, "/hydrogen", "", "GET", "Returns the current hydrogen data", getHydrogen)
	RegisterWebSiteAPI(router, "/getFuelCell", "", "GET", "Returns the current status of the fuel cell only", getFuelCell)
	RegisterWebSiteAPI(router, "/setFuelCell/TargetPower/{power}", "", "PUT", "Set the target power output", setFcPower)
	RegisterWebSiteAPI(router, "/setFuelCell/TargetBattHigh/{volts}", "", "PUT", "Set the battery high voltage setpoint", setFcBattHigh)
	RegisterWebSiteAPI(router, "/setFuelCell/TargetBattLow/{volts}", "", "PUT", "Set the battery low voltage set point", setFcBatLow)
	RegisterWebSiteAPI(router, "/setFuelCell/Start", "", "PUT", "Start the fuel cell", startFc)
	RegisterWebSiteAPI(router, "/setFuelCell/Stop", "", "PUT", "Stop the fuel cell", stopFc)
	RegisterWebSiteAPI(router, "/setFuelCellSettings", "", "POST", "Set the fuel cell settings", setFuelCellSettings)
	RegisterWebSiteAPI(router, "/setFuelCell/ExhaustOpen", "", "PUT", "Start the water pump on high and begin air removal", exhaustOpen)
	RegisterWebSiteAPI(router, "/setFuelCell/ExhaustClose", "", "PUT", "Stop the exhaust function", exhaustClose)
	RegisterWebSiteAPI(router, "/setFuelCell/Enable", "", "PUT", "Enable CAN communications to the fuel cell (we are always listening but may not be sending)", enableFc)
	RegisterWebSiteAPI(router, "/setFuelCell/Disable", "", "PUT", "Disable CAN communications to the fuel cell so it can be controlled locally by its own user interface", disableFc)
	RegisterWebSiteAPI(router, "/setFuelCell/ResetFault", "", "PUT", "Send the reset fault code to the fuel cell", resetFCFault)
	RegisterWebSiteAPI(router, "/setFuelCell/TurnOnHeater", "", "PUT", "Turn on the fuel cell coolant heater", turnOnFCHeater)
	RegisterWebSiteAPI(router, "/setFuelCell/TurnOffHeater", "", "PUT", "Turn off the fuel cell coolant heater", turnOffFCHeater)
	RegisterWebSiteAPI(router, "/Electrolyser.html/{electrolyser}", "", "GET", "Open the Electrolyser screen", serveElectrolyser)
	RegisterWebSiteAPI(router, "/Electrolyser.html/{electrolyser}", "", "POST", "Allows various maintenance operations to be recorded for an electrolyser", recordElMaintenance)
	RegisterWebSiteAPI(router, "/electrolyser/acquire}", "", "GET", "Go and find the electrolyser IP address based on its name or index.", acquireElectrolysers)
	RegisterWebSiteAPI(router, "/getElectrolyser/{electrolyser}", "", "GET", "Get the status for an electrolyser by electrolyser name or index", getElectrolyserStatus)
	RegisterWebSiteAPI(router, "/setElectrolyser/Production/{electrolyser}/{rate}", "", "PUT", "Set the electrolyser production rate (60-100) by electrolyser name or index.", setElectrolyserProductionRate)
	RegisterWebSiteAPI(router, "/setElectrolyser/Start/{electrolyser}", "", "PUT", "Start the electrolyser by electrolyser name or index", startElectrolyser)                                                 //
	RegisterWebSiteAPI(router, "/setElectrolyser/Stop/{electrolyser}", "", "PUT", "Stop the electrolyser by electrolyser name or index", stopElectrolyser)                                                    //
	RegisterWebSiteAPI(router, "/setElectrolyser/Preheat/{electrolyser}", "", "PUT", "Start the preheating cycle by electrolyser name or index", preheatElectrolyser)                                         //
	RegisterWebSiteAPI(router, "/setElectrolyser/Reboot/{electrolyser}", "", "PUT", "Reboot the electrolyser by electrolyser name or index", rebootElectrolyser)                                              //
	RegisterWebSiteAPI(router, "/setElectrolyser/Locate/{electrolyser}", "", "PUT", "Start the electrolyser location signal by electrolyser name or index", locateElectrolyser)                               //
	RegisterWebSiteAPI(router, "/setElectrolyser/BlowDown/{electrolyser}", "", "PUT", "Blow down the electrolyser by electrolyser name or index", blowDownElectrolyser)                                       //
	RegisterWebSiteAPI(router, "/setElectrolyser/Refill/{electrolyser}", "", "PUT", "Refill the electrolyser by electrolyser name or index", refillElectrolyser)                                              //
	RegisterWebSiteAPI(router, "/setElectrolyser/StartMaintenance/{electrolyser}", "", "PUT", "Start the maintenance cycle for the electrolyser by electrolyser name or index", startMaintenanceElectrolyser) //
	RegisterWebSiteAPI(router, "/setElectrolyser/StopMaintenance/{electrolyser}", "", "PUT", "Stop the maintenance cycle for the electrolyser by electrolyser name or index", stopMaintenanceElectrolyser)    //
	RegisterWebSiteAPI(router, "/setElectrolyser/PowerOn/{electrolyser}", "", "PUT", "Turn on the relay associated with the electrolyser by electrolyser name or index", powerOnElectrolyser)
	RegisterWebSiteAPI(router, "/setElectrolyser/PowerOff/{electrolyser}", "", "PUT", "Turn off the relay associated with the electrolyser by electrolyser name or index", powerOffElectrolyser)
	RegisterWebSiteAPI(router, "/ElectrolyserMaintenance/{electrolyser}", "", "GET", "Show the form to log maintenance activity on an electrolyser", showElectrolyserMaintenance)
	RegisterWebSiteAPI(router, "/setDryer/Start", "", "PUT", "Start the dryer", startDryer)                                                                                                                        // Start the dryer
	RegisterWebSiteAPI(router, "/setDryer/Stop", "", "PUT", "Stop the dryer", stopDryer)                                                                                                                           //
	RegisterWebSiteAPI(router, "/setDryer/Reboot", "", "PUT", "Reboot the dryer", rebootDryer)                                                                                                                     //
	RegisterWebSiteAPI(router, "/getElectrolyser/production/{electrolyser}", "", "GET", "Returns the current production rate setting by electrolyser name or index", getElectrolyserProductionRate)                //
	RegisterWebSiteAPI(router, "/getElectrolyser/state/{electrolyser}", "", "GET", "Returns the electrolyser run state by electrolyser name or index", getElectrolyserState)                                       //
	RegisterWebSiteAPI(router, "/setElectrolyser/Rescan/{electrolyser}", "", "PUT", "Scans for an electrolyser that has the same IP. Used to find an electrolyser that has moved to a new IP", rescanElectrolyser) //
	RegisterWebSiteAPI(router, "/calibrateDC/{channel}/{type}/{value}", "", "PUT", "Sends the given calibration measured value to the DC monitor", calibrateDC)                                                    //
	RegisterWebSiteAPI(router, "/calibrateDC/{channel}", "", "GET", "Opens the calibration form for the given DC device", openDCCalibration)                                                                       //
	RegisterWebSiteAPI(router, "/title", "", "GET", "Returns the system title e.g. Florida", getTitle)

	RegisterWebSiteAPI(router, "/debug/{on}", "", "GET", "Enable debug output", setDebug)
	RegisterWebSiteAPI(router, "/logCalls/{on}", "", "GET", "Enable logging of all API calls", setCallLogging)
	RegisterWebSiteAPI(router, "/logCanBus/{on}", "", "GET", "Enable logging of all CAN bus errors", setCANLogging)

	RegisterWebSiteAPI(router, "/recordPower", "", "POST", `Post a JSON block {"source":"firefly", current":i.i, "voltage": v.v, "soc":s.s, "hz":f.f, "solar":w } hz is optional, positive current is charging.`, recordPowerData)
	RegisterWebSiteAPI(router, "/recordBatteryVolts", "", "POST", `Post a JSON block {"source":"firefly", "voltage": v.v }`, recordBatteryVolts)
	RegisterWebSiteAPI(router, "/recordBatteryAmps", "", "POST", `Post a JSON block {"source":"firefly", "current":i.i } positive current is charging.`, recordBatteryAmps)
	RegisterWebSiteAPI(router, "/recordBatterySOC", "", "POST", `Post a JSON block {"source":"firefly", "soc":s.s } State of charge as %`, recordBatterySOC)
	RegisterWebSiteAPI(router, "/recordBatteryHertz", "", "POST", `Post a JSON block {"source":"firefly", "hz":f.f } hz is mains frequency in Hz.`, recordMainsFrequency)
	RegisterWebSiteAPI(router, "/recordSolar", "", "POST", `Post a JSON Block {"source":"firefly", "solar":w, "hz":f.f } hz is mains frequency and is optional. Set to 0 to ignore`, recordSolar)

	RegisterWebSiteAPI(router, "/h2", "", "GET", `Return the current hydrogen volume`, getH2Volume)

	// Historical data access
	RegisterWebSiteAPI(router, "/FuelCellData/DCDC", "start=datetime&end=datetime", "GET", "Get the historical data for the DC-DC converter. Take start and end parameters to determining the time span.", getFuelCellDCDCData)
	RegisterWebSiteAPI(router, "/FuelCellData/Stack", "start=datetime&end=datetime", "GET", "Get the historical data for the fuel cells stack. Take start and end parameters to determining the time span.", getFuelCellStackData)
	RegisterWebSiteAPI(router, "/FuelCellData/Pressures", "start=datetime&end=datetime", "GET", "Get the historical data for the fuel cell pressures. Take start and end parameters to determining the time span.", getFuelCellPressureData)
	RegisterWebSiteAPI(router, "/FuelCellData/Coolant", "start=datetime&end=datetime", "GET", "Get the historical data for the fuel cell coolant. Take start and end parameters to determining the time span.", getFuelCellCoolantData)
	RegisterWebSiteAPI(router, "/Electrolyser/Data/{electrolyser}", "start=datetime&end=datetime", "GET", "Get the historical data for the electrolyser. Take start and end parameters to determining the time span.", getElectrolyserData)
	RegisterWebSiteAPI(router, "/Analog/Data", "start=datetime&end=datetime", "GET", "Get the historical data for the analog inputs. Take start and end parameters to determining the time span.", getAnalogData)
	RegisterWebSiteAPI(router, "/AC/Data", "start=datetime&end=datetime", "GET", "Get the historical data for the AC devices. Take start and end parameters to determining the time span.", getACData)
	RegisterWebSiteAPI(router, "/DC/Data", "start=datetime&end=datetime", "GET", "Get the historical data for the DC devices. Take start and end parameters to determining the time span.", getDCData)
	RegisterWebSiteAPI(router, "/Hydrogen/Data", "start=datetime&end=datetime", "GET", "Get the historical data for the Hydrogen storage. Take start and end parameters to determining the time span.", getHydrogenData)
	RegisterWebSiteAPI(router, "/Power/Data", "source=firefly&start=datetime&end=datetime", "GET", "Get the historical data for the battery and solar inputs. Take start and end parameters to determining the time span.", getPowerData)

	// Charts
	RegisterWebSiteAPI(router, "/ElectrolyserData.html", "", "GET", "Display the graph for the electrolyser", serveElectrolyserData)
	RegisterWebSiteAPI(router, "/AnalogData.html", "", "GET", "Display the graph for the analog inputs", serveAnalogData)
	RegisterWebSiteAPI(router, "/ACData.html", "", "GET", "Display the graph for the AC Device", serveACData)
	RegisterWebSiteAPI(router, "/DCData.html", "", "GET", "Display the graph for the DC Device", serveDCData)

	// Default page
	RegisterWebSiteAPI(router, "/", "", "GET", "Serve the default home page (same as /default.html)", serveDefault)
	RegisterWebSiteAPI(router, "/userControl.html", "", "GET", "User control page", serveUserControl)
	RegisterWebSiteAPI(router, "/default.html", "", "GET", "Serve the default home page", serveDefault)
	RegisterWebSiteAPI(router, "/admin.html", "", "GET", "Serve the admin home page", serveAdmin)
	RegisterWebSiteAPI(router, "/ping", "", "GET", "Respond to ping. Can be used to verify connectivity", ping)

	// Refresh Certificates
	RegisterWebSiteAPI(router, "/refreshCertificates", "", "GET", "Execute the script to get updated certificates from the cloud site.", RefreshCertificates)

	router.HandleFunc("/api", buildAPIDocumentationPage).Methods("GET")
	router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, webFiles+"/images/favicon.ico") }).Methods("GET")
}
