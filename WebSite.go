package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const firefly = "firefly"

type neuteredFileSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	} else {
		if s, err := f.Stat(); err != nil {
			return f, err
		} else {
			if s.IsDir() {
				index := filepath.Join(path, "index.html")
				if _, err := nfs.fs.Open(index); err != nil {
					closeErr := f.Close()
					if closeErr != nil {
						return nil, closeErr
					}
					return nil, err
				}
			}
		}
	}
	return f, nil
}

func RequestLoggerMiddleware(_ *mux.Router) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Check Authentication
			// log.Println("Checking - ", req.RequestURI)
			if err, code := Authenticate(w, req); err != nil || code != 0 {
				if code == 1 {
					serveDefault(w, req)
					return
				}
				http.ServeFile(w, req, webFiles+"/Login.html")
				return
			}
			//}
			if callLogging {
				defer func() {
					log.Printf(
						"[%s] %s %s %s",
						req.Method,
						req.Host,
						req.URL.Path,
						req.URL.RequestURI(),
					)
				}()
			}
			next.ServeHTTP(w, req)
		})
	}
}

func setUpLocalWebSocket() {
	wsRouter := mux.NewRouter()
	wsRouter.HandleFunc("/ws/fuelcell", startFuelCellWebSocket).Methods("GET")
	wsRouter.HandleFunc("/ws/electrolyser/{electrolyser}", startElectrolyserWebSocket).Methods("GET")
	wsRouter.HandleFunc("/ws", startDataWebSocket).Methods("GET")

	if webPort, err := strconv.ParseInt(WebPort, 10, 16); err != nil {
		log.Fatal(err)
	} else {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", webPort+1), wsRouter))
	}
}

var cert tls.Certificate

func setUpWebSite() {
	//	pool = NewPool()
	pool.Init()
	go pool.StartRegister()
	go pool.StartUnregister()
	go pool.StartBroadcast()

	router := mux.NewRouter().StrictSlash(true)
	RegisterWebSiteAPICalls(router)
	router.HandleFunc("/action-login", serveDefault).Methods("POST")

	fileServer := http.FileServer(neuteredFileSystem{http.Dir(webFiles)})
	router.PathPrefix("/").Handler(http.StripPrefix("/", fileServer))

	router.Use(RequestLoggerMiddleware(router))

	port := fmt.Sprintf(":%s", WebPort)
	//	certFile := "/certs/localhost.crt"
	//	keyFile := "/certs/localhost.key"
	//log.Fatal(http.ListenAndServe(port, router))

	server := &http.Server{
		Addr:    port,
		Handler: router,
		TLSConfig: &tls.Config{
			GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
				// Load the latest localhost.crt and localhost.key if the certificate is nil
				if cert.Certificate == nil {
					if newCert, err := tls.LoadX509KeyPair(certFile, keyFile); err != nil {
						log.Print(err)
					} else {
						cert = newCert
					}
				}
				return &cert, nil
			},
		},
	}
	for {
		log.Fatal(server.ListenAndServeTLS("", ""))
	}
}

type PowerBody struct {
	Source    string  `json:"source"`
	Amps      float64 `json:"iBatt"`
	Volts     float64 `json:"vBatt"`
	SOC       float64 `json:"soc"`
	Frequency float64 `json:"frequency"`
	Solar     float64 `json:"solar"`
}

func recordPowerData(w http.ResponseWriter, r *http.Request) {
	var data PowerBody
	data.Source = firefly
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnJSONError(w, "recordBatteryData", err, http.StatusBadRequest, true)
		return
	}
	pc := FindPowerControl(data.Source)
	pc.setAll(data.Volts, data.Amps, data.SOC, data.Frequency, data.Solar)
}

func recordBatteryVolts(w http.ResponseWriter, r *http.Request) {
	var data PowerBody
	data.Source = firefly
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnJSONError(w, "recordBatteryVolts", err, http.StatusBadRequest, true)
		return
	}
	pc := FindPowerControl(data.Source)
	pc.setVoltage(data.Volts)
}

func recordBatteryAmps(w http.ResponseWriter, r *http.Request) {
	var data PowerBody
	data.Source = firefly
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnJSONError(w, "recordBatteryAmps", err, http.StatusBadRequest, true)
		return
	}
	pc := FindPowerControl(data.Source)
	pc.setCurrent(data.Amps)
}
func recordBatterySOC(w http.ResponseWriter, r *http.Request) {
	var data PowerBody
	data.Source = firefly
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnJSONError(w, "recordBatterySOC", err, http.StatusBadRequest, true)
		return
	}
	pc := FindPowerControl(data.Source)
	pc.setStateOfCharge(data.SOC)
}
func recordMainsFrequency(w http.ResponseWriter, r *http.Request) {
	var data PowerBody
	data.Source = firefly
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnJSONError(w, "recordMainsFrequency", err, http.StatusBadRequest, true)
		return
	}
	pc := FindPowerControl(data.Source)
	pc.setFrequency(data.Frequency)
}

func recordSolar(w http.ResponseWriter, r *http.Request) {
	var data PowerBody
	data.Source = firefly
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		ReturnJSONError(w, "recordBatteryData", err, http.StatusBadRequest, true)
		return
	}
	log.Println(data)
	pc := FindPowerControl(data.Source)
	pc.setSolar(data.Solar)
	if data.Frequency > 0 {
		pc.setFrequency(data.Frequency)
	}
}

func getTitle(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := fmt.Fprintf(w, `{"title":"%s"}`, currentSettings.Name); err != nil {
		log.Println(err)
	}
}

func RefreshCertificates(w http.ResponseWriter, _ *http.Request) {
	// Set the certificate to nil to force a reload of the certificate files
	cert.Certificate = nil
	ReturnJSONSuccess(w)
}

func ping(w http.ResponseWriter, _ *http.Request) {
	if _, err := fmt.Fprint(w, "OK"); err != nil {
		log.Println(err)
	}
}

func openDCCalibration(w http.ResponseWriter, r *http.Request) {
	const function = "openDCCalibration"

	vars := mux.Vars(r)
	channel, err := strconv.ParseInt(vars["channel"], 10, 8)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	name := DCMeasurements[channel].Name

	if fileContent, err := os.ReadFile(webFiles + "/DCCalibration.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		channel := fmt.Sprintf("  <script>\n    var channel= %s;\n  </script>\n", vars["channel"])
		if _, err := fmt.Fprint(w, strings.Replace(strings.Replace(string(fileContent), `<!--variables-->`, channel, -1), "<!--Name-->", name, -1)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

func calibrateDC(w http.ResponseWriter, r *http.Request) {
	const function = "calibrateDC"
	vars := mux.Vars(r)
	if channel, err := strconv.ParseInt(vars["channel"], 10, 8); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, false)
		return
	} else {
		if value, err := strconv.ParseFloat(vars["value"], 64); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, false)
			return
		} else {
			log.Printf("Channel = %d : type = %s : value = %f\n", channel, vars["type"], value)
			switch vars["type"] {
			case "lowVolts":
				if err := canBus.SetDCCalibration(uint8(channel), CALIBRATE_DC_VOLTAGE_LOW, value); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
					return
				}
				break
			case "highVolts":
				if err := canBus.SetDCCalibration(uint8(channel), CALIBRATE_DC_VOLTAGE_HIGH, value); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
					return
				}
				break
			case "lowCurrent":
				if err := canBus.SetDCCalibration(uint8(channel), CALIBRATE_DC_CURRENT_LOW, value); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
					return
				}
				break
			case "highCurrent":
				if err := canBus.SetDCCalibration(uint8(channel), CALIBRATE_DC_CURRENT_HIGH, value); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
					return
				}
				break
			default:
				ReturnJSONErrorString(w, function, "Invalid parameter - "+vars["type"], http.StatusBadRequest, true)
				return
			}
		}
	}
	ReturnJSONSuccess(w)
}

func setSettings(w http.ResponseWriter, r *http.Request) {
	currentSettings.setSettings(w, r)
}

func serveUserControl(w http.ResponseWriter, r *http.Request) {
	const function = "Load User Control Page"
	role := "user"
	if session, err := store.Get(r, "user-session"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		roleInterface := session.Values["role"]
		if roleInterface != nil {
			role = session.Values["role"].(string)
		} else {
			log.Println("No role found in the session")
			http.ServeFile(w, r, webFiles+"/Login.html")
			return
		}
	}
	adminLink := ""
	if role == "admin" {
		adminLink = `<li><a id="adminLink" href="/admin.html">Administration</a></li>`
	}
	if page, err := os.ReadFile(webFiles + "/userControl.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		if _, err := fmt.Fprint(w, strings.Replace(string(page), "<!--adminLink-->", adminLink, -1)); err != nil {
			log.Println(err)
		}
	}
}

func serveDefault(w http.ResponseWriter, r *http.Request) {
	role := "user"
	if session, err := store.Get(r, "user-session"); err != nil {
		ReturnJSONError(w, "Load Default Page", err, http.StatusInternalServerError, true)
		return
	} else {
		roleInterface := session.Values["role"]
		if roleInterface != nil {
			role = session.Values["role"].(string)
		} else {
			log.Println("No role found in the session")
			http.ServeFile(w, r, webFiles+"/Login.html")
			return
		}
	}
	adminLink := ""
	if role == "admin" {
		adminLink = `<li><a id="adminLink" href="/admin.html">Administration</a></li>`
	}
	if page, err := os.ReadFile(webFiles + "/default.html"); err != nil {
		ReturnJSONError(w, "ServerDefault", err, http.StatusInternalServerError, true)
		return
	} else {
		if _, err := fmt.Fprint(w, strings.Replace(string(page), "<!--adminLink-->", adminLink, -1)); err != nil {
			log.Println(err)
		}
	}
}

func generateKey(w http.ResponseWriter, _ *http.Request) {
	guid := uuid.NewString()
	if _, err := fmt.Fprintf(w, `{ "guid": "%s" }`, guid); err != nil {
		log.Println(err)
	}
}

func resetFCFault(w http.ResponseWriter, _ *http.Request) {
	FuelCell.ClearFaults()
	ReturnJSONSuccess(w)
}

func setDebug(w http.ResponseWriter, r *http.Request) {
	const function = "setDebug"
	vars := mux.Vars(r)
	on := vars["on"]

	on = strings.ToLower(on)
	if (on == "on") || (on == "true") || (on == "1") {
		debugOutput = true
	} else if (on == "off") || (on == "false") || (on == "0") {
		debugOutput = false
	} else {
		ReturnJSONErrorString(w, function, "Invalid value given for debug setting. Valid values are on, true, 1, off, false or 0", http.StatusBadRequest, true)
		return
	}
	w.Header().Add("Cache-Control", "no-store")
	http.ServeFile(w, r, webFiles+"/debug.html")
}

func setCallLogging(w http.ResponseWriter, r *http.Request) {
	const function = "setCallLogging"
	vars := mux.Vars(r)
	on := vars["on"]

	on = strings.ToLower(on)
	if (on == "on") || (on == "true") || (on == "1") {
		callLogging = true
	} else if (on == "off") || (on == "false") || (on == "0") {
		callLogging = false
	} else {
		ReturnJSONErrorString(w, function, "Invalid value given for debug setting. Valid values are on, true, 1, off, false or 0", http.StatusBadRequest, true)
		return
	}
	w.Header().Add("Cache-Control", "no-store")
	http.ServeFile(w, r, webFiles+"/debug.html")
}

func redirectToNodeRED(w http.ResponseWriter, r *http.Request) {
	url := currentSettings.NodeRED
	if url == "" {
		http.ServeFile(w, r, webFiles+"/config.html")
	} else {
		if strings.Contains(r.RequestURI, "20080") {
			url = strings.Replace(strings.Replace(r.RequestURI, "20080", "1880", 1), "NodeRED", "", 1)
		} else {
			if strings.HasSuffix(r.RequestURI, "0/NodeRED") {
				url = strings.Replace(r.RequestURI, "0/NodeRED", "1", 1)
			}
		}
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func redirectToNodeREDUI(w http.ResponseWriter, r *http.Request) {
	url := currentSettings.NodeRED + "/ui"
	if strings.Contains(r.RequestURI, "20080") {
		// We are local so go to the local NodeRedUI
		url = strings.Replace(strings.Replace(r.RequestURI, "20080", "1880", 1), "NodeREDUI", "ui", 1)
	} else {
		if strings.HasSuffix(r.RequestURI, "0/NodeREDUI") {
			url = strings.Replace(r.RequestURI, "0/NodeREDUI", "1/ui", 1)
		}
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

type ReplacementsType map[string]string

func replaceText(txt string, replacements ReplacementsType) string {
	for key, val := range replacements {
		//		log.Println("Replace ", key, " with ", val)
		txt = strings.Replace(txt, "{{"+key+"}}", val, -1)
	}
	return txt
}

func serveElectrolyser(w http.ResponseWriter, r *http.Request) {
	const function = "serveElectrolyser"

	if fileContent, err := os.ReadFile(webFiles + "/Electrolyser.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		replacements := make(ReplacementsType)
		replacements["title"] = currentSettings.Name + " - " + r.FormValue("name")
		replacements["name"] = r.FormValue("name")
		replacements["version"] = version

		if _, err := fmt.Fprint(w, replaceText(string(fileContent), replacements)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

func serveElectrolyserData(w http.ResponseWriter, r *http.Request) {
	const function = "serveElectrolyserData"

	if fileContent, err := os.ReadFile(webFiles + "/ElectrolyserData.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		replacements := make(ReplacementsType)
		replacements["name"] = r.FormValue("name")

		if _, err := fmt.Fprint(w, replaceText(string(fileContent), replacements)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

func serveACData(w http.ResponseWriter, r *http.Request) {
	const function = "serveACData"

	if fileContent, err := os.ReadFile(webFiles + "/ACData.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		replacements := make(ReplacementsType)
		replacements["name"] = r.FormValue("name")
		replacements["channel"] = r.FormValue("channel")

		if _, err := fmt.Fprint(w, replaceText(string(fileContent), replacements)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

func serveDCData(w http.ResponseWriter, r *http.Request) {
	const function = "serveDCData"

	if fileContent, err := os.ReadFile(webFiles + "/DCData.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		replacements := make(ReplacementsType)
		replacements["name"] = r.FormValue("name")
		replacements["channel"] = r.FormValue("channel")

		if _, err := fmt.Fprint(w, replaceText(string(fileContent), replacements)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

func replacementText(setting AnalogSettingType) string {
	return fmt.Sprintf(`{"name":"%s", "min": %f, "max": %f, "interval": %d}`,
		setting.Name, setting.MinVal,
		setting.MaxVal, int64((setting.MaxVal-setting.MinVal)/5))
}

func serveAnalogData(w http.ResponseWriter, r *http.Request) {
	const function = "serveAnalogData"

	if fileContent, err := os.ReadFile(webFiles + "/AnalogData.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		replacements := make(ReplacementsType)
		replacements["name"] = r.FormValue("name")

		for idx := 0; idx < 8; idx++ {

		}
		replacements["struct"] = fmt.Sprintf("[%s,%s,%s,%s,%s,%s,%s,%s]",
			replacementText(currentSettings.AnalogChannels[0]),
			replacementText(currentSettings.AnalogChannels[1]),
			replacementText(currentSettings.AnalogChannels[2]),
			replacementText(currentSettings.AnalogChannels[3]),
			replacementText(currentSettings.AnalogChannels[4]),
			replacementText(currentSettings.AnalogChannels[5]),
			replacementText(currentSettings.AnalogChannels[6]),
			replacementText(currentSettings.AnalogChannels[7]))
		if _, err := fmt.Fprint(w, replaceText(string(fileContent), replacements)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}

func rescanElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "rescanElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if debugOutput {
			log.Println("Rescanning for ", request)
		}
		if IP, sIP, err := el.rescan(1, currentSettings.findElByName(request).Serial); err != nil {
			ReturnJSONError(w, function, err, http.StatusNotFound, true)
		} else {
			currentSettings.findElByName(request).IP = IP.String()
			if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				log.Println("Electrolyser found at", sIP)
				ReturnJSONSuccess(w)
			}
		}
	}
}

func getElectrolyserState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "getElectrolyserState"
	var State struct {
		State string `json:"state"`
	}

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
		return
	} else {
		if !el.IsSwitchedOn() {
			State.State = "OFF"
		} else {
			State.State = el.getState()
		}
		if sData, err := json.Marshal(State); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		} else {
			if _, err := fmt.Fprintf(w, string(sData)); err != nil {
				log.Println(err)
			}
		}
	}
}

func setElectrolyserProductionRate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "setElectrolyserProductionRate"

	if debugOutput {
		log.Println("Set ", request, " production = ", vars["rate"])
	}
	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
		return
	} else {
		if rate, err := strconv.ParseFloat(vars["rate"], 64); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, false)
			return
		} else {
			if rate < 0 || rate > 100 {
				str := fmt.Sprintf("Invalid rate %f (0..100 allowed)", rate)
				ReturnJSONErrorString(w, function, str, http.StatusBadRequest, false)
			} else {
				el.SetProduction(uint8(rate))
				if debugOutput {
					log.Printf("Production rate set to %d on %s", uint8(rate), request)
				}
				if err := el.ReadValues(); err != nil {
					log.Print(err)
				}
			}
		}
	}
	ReturnJSONSuccess(w)
}

func getElectrolyserProductionRate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "getElectrolyserProductionRate"
	var Rate struct {
		Rate int `json:"rate"`
	}

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if el.IsSwitchedOn() {
			Rate.Rate = el.GetRate()
		} else {
			Rate.Rate = 0
		}
		if sData, err := json.Marshal(Rate); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		} else {
			if _, err := fmt.Fprintf(w, string(sData)); err != nil {
				log.Println(err)
			}
		}
	}
}

// getElectrolyserData returns the data collected from the named electrolyser between two date/times
func getElectrolyserData(w http.ResponseWriter, r *http.Request) {
	const function = "getElectrolyserData"

	if pDB == nil {
		ReturnJSONErrorString(w, function, "No Database", http.StatusInternalServerError, true)
		return
	}

	vars := mux.Vars(r)
	name := strings.ToLower(vars["electrolyser"])

	if el := Electrolysers.FindByName(name); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+name+" was not found", http.StatusBadRequest, false)
	} else {
		if start, end, err := GetTimeRange(r); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, false)
		} else {
			if end.Sub(start) > time.Hour {
				SendDataAsJSON(w, function, ElectrolyserDataByMinute, name, start, end)
			} else {
				SendDataAsJSON(w, function, ElectrolyserDataBySecond, name, start, end)

			}
		}
	}
}

func getACData(w http.ResponseWriter, r *http.Request) {
	const function = "getACData"

	if pDB == nil {
		ReturnJSONErrorString(w, function, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendDataAsJSON(w, function, ACDataByMinute, start, end)
		} else {
			SendDataAsJSON(w, function, ACDataBySecond, start, end)
		}
	}
}

func getDCData(w http.ResponseWriter, r *http.Request) {
	const function = "getDCData"

	if pDB == nil {
		ReturnJSONErrorString(w, function, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendDataAsJSON(w, function, DCDataByMinute, start, end)
		} else {
			SendDataAsJSON(w, function, DCDataBySecond, start, end)
		}
	}
}

func getAnalogData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "getAnalogData"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendAnalogByMinute(w, start, end)
		} else {
			SendAnalogBySecond(w, start, end)
		}
	}
}

func getPowerData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "getPowerData"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	params := r.URL.Query()
	source := params.Get("source")
	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		if end.Sub(start) > time.Hour {
			SendPowerByMinute(w, start, end, source)
		} else {
			SendPowerBySecond(w, start, end, source)
		}
	}
}

type h2DataType struct {
	TankCapacity  float64 `json:"tank_volume"`
	TankRemaining float64 `json:"tank_remaining"`
}

func getH2Volume(w http.ResponseWriter, _ *http.Request) {
	var H2 h2DataType

	// Calculate hydrogen using the ideal gas law PV=nRT
	// M = (V * P * C1) / (T1 + T) Where V is litres, P is bar and T is Celsius
	const C1 = 0.02424826
	const T1 = 273.15
	var (
		volume      = float64(currentSettings.GasCapacity)
		gasPressure float64
	)

	if currentSettings.GasUnits == "bar" {
		// SI units
		gasPressure = float64(AnalogInputs.Inputs[currentSettings.GasPressureInput].Value)
	} else {
		// stupid units - convert to SI
		gasPressure = float64(AnalogInputs.Inputs[currentSettings.GasPressureInput].Value) / 14.50377
	}
	H2.TankRemaining = (volume * gasPressure * C1) / (T1 + AnalogInputs.GasTemperature)
	H2.TankCapacity = (volume * 35.0 * C1) / (T1 + AnalogInputs.GasTemperature)

	if sJSON, err := json.Marshal(H2); err != nil {
		ReturnJSONError(w, "getH2Volume", err, http.StatusInternalServerError, true)
	} else {
		if _, err := fmt.Fprint(w, sJSON); err != nil {
			log.Println(err)
		}
	}
}

func getHydrogenData(w http.ResponseWriter, r *http.Request) {
	const DeviceString = "Hydrogen Data"

	if pDB == nil {
		ReturnJSONErrorString(w, DeviceString, "No Database", http.StatusInternalServerError, true)
		return
	}

	if start, end, err := GetTimeRange(r); err != nil {
		ReturnJSONError(w, DeviceString, err, http.StatusBadRequest, false)
	} else {
		start := start.Truncate(time.Hour)
		end := end.Truncate(time.Hour)
		if end.Sub(start) > (time.Hour * 24) {
			SendHydrogenByDay(w, start, end)
		} else {
			SendHydrogenByHour(w, start, end)
		}
	}
}

func powerOnElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "powerOnElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if err := turnOnOffRelay(el.powerRelay, true); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		} else {
			ReturnJSONSuccess(w)
		}
	}
}

func powerOffElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "powerOffElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if err := turnOnOffRelay(el.powerRelay, false); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		} else {
			ReturnJSONSuccess(w)
		}
	}
}

func startElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "startElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		// Is this the first to be started?
		started := false
		for _, el := range Electrolysers.Arr {
			if el.IsSwitchedOn() {
				if el.status.State == 3 {
					started = true
				}
			}
		}
		if !started {
			// We are the first to be started
			switch currentSettings.WaterDumpAction {
			case ElStart:
				go TurnOnWaterDumpValve()
			case ElStartAndConductivity:
				_, conductivity := AnalogInputs.GetInput(7)
				if float32(currentSettings.MaximumConductivity) < conductivity {
					go TurnOnWaterDumpValve()
				}
			default:
			}
		}
		if err := el.Start(); err != http.StatusOK {
			if err == http.StatusConflict {
				ReturnJSONErrorString(w, function, "Too soon after last Stop", err, false)
			} else if err == http.StatusBadRequest {
				ReturnJSONErrorString(w, function, "Electrolyser is not powered on", err, false)
			} else {
				ReturnJSONErrorString(w, function, "Unknown Error", err, true)
			}
		} else {
			ReturnJSONSuccess(w)
		}
	}
}

func stopElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "stopElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if err := el.Stop(); err != http.StatusOK {
			if err == http.StatusConflict {
				ReturnJSONErrorString(w, function, "Too soon after last Start", err, false)
			} else {
				if err == http.StatusBadRequest {
					ReturnJSONErrorString(w, function, "Electrolyser is not powered on", err, false)
				} else {
					ReturnJSONErrorString(w, function, "Unknown Error", err, true)
				}
			}
		} else {
			ReturnJSONSuccess(w)
		}
	}
}

func preheatElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "preheatElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
		return
	} else {
		if err := el.Preheat(); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
	ReturnJSONSuccess(w)
}

func rebootElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "rebootElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if !el.IsSwitchedOn() {
			ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
		} else {
			if err := el.Reboot(); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				ReturnJSONSuccess(w)
				log.Println("Reboot sent.")
			}
		}
	}
}

func locateElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "locateElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if !el.IsSwitchedOn() {
			ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
		} else {
			if err := el.Locate(); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				ReturnJSONSuccess(w)
			}
		}
	}
}

func blowDownElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "blowDownElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
		return
	} else {
		if !el.IsSwitchedOn() {
			ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
		} else {
			if err := el.BlowDown(); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				ReturnJSONSuccess(w)
			}
		}
	}
}

func refillElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "refillElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if !el.IsSwitchedOn() {
			ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
		} else {
			if err := el.Refill(); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				ReturnJSONSuccess(w)
			}
		}
	}
}

func startMaintenanceElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "startMaintenanceElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if !el.IsSwitchedOn() {
			ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
		} else {
			if err := el.EnableMaintenance(); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				ReturnJSONSuccess(w)
			}
		}
	}
}

func stopMaintenanceElectrolyser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "stopMaintenanceElectrolyser"

	if el := Electrolysers.FindByName(request); el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser "+request+" was not found", http.StatusBadRequest, false)
	} else {
		if !el.IsSwitchedOn() {
			ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
		} else {
			if err := el.DisableMaintenance(); err != nil {
				ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			} else {
				ReturnJSONSuccess(w)
			}
		}
	}
}

func startDryer(w http.ResponseWriter, _ *http.Request) {
	const function = "startDryer"
	for _, el := range Electrolysers.Arr {
		if el.hasDryer {
			if !el.IsSwitchedOn() {
				ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
			} else {
				if err := el.StartDryer(); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
				} else {
					ReturnJSONSuccess(w)
				}
			}
			return
		}
	}
	ReturnJSONErrorString(w, function, "Dryer not found", http.StatusBadRequest, false)
}

func stopDryer(w http.ResponseWriter, _ *http.Request) {
	const function = "stopDryer"
	for _, el := range Electrolysers.Arr {
		if el.hasDryer {
			if !el.IsSwitchedOn() {
				ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
			} else {
				if err := el.StopDryer(); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
				} else {
					ReturnJSONSuccess(w)
				}
			}
			return
		}
	}
	ReturnJSONErrorString(w, function, "Dryer not found", http.StatusBadRequest, false)
}

func rebootDryer(w http.ResponseWriter, _ *http.Request) {
	const function = "rebootDryer"
	for _, el := range Electrolysers.Arr {
		if el.hasDryer {
			if !el.IsSwitchedOn() {
				ReturnJSONErrorString(w, function, "Electrolyse must be powered up first.", http.StatusBadRequest, false)
			} else {
				if err := el.RebootDryer(); err != nil {
					ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
				} else {
					ReturnJSONSuccess(w)
				}
			}
			return
		}
	}
	ReturnJSONErrorString(w, function, "Dryer not found", http.StatusBadRequest, false)
}

func getElectrolyserStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "getElectrolyserStatus"

	status := make([]*ElectrolyserStatusType, 0)

	if request == "all" {
		for _, el := range Electrolysers.Arr {
			status = append(status, el.getStatus())
		}
	} else {
		el := Electrolysers.FindByName(request)
		if el == nil {
			ReturnJSONErrorString(w, function, "Electrolyser not found - "+request, http.StatusBadRequest, false)
			return
		}
		status = append(status, el.getStatus())
	}
	if str, err := json.Marshal(status); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
	} else {
		if _, err = fmt.Fprint(w, string(str)); err != nil {
			log.Println(err)
		}
	}
}

func startDataWebSocket(w http.ResponseWriter, r *http.Request) {
	if debugOutput {
		log.Print("WebSocket Endpoint Hit")
	}
	conn, err := Upgrade(w, r)
	if err != nil {
		_, err = fmt.Fprintf(w, "%+v\n", err)
		if err != nil {
			log.Println(err)
		}
	}

	client := &Client{
		ID:      r.RemoteAddr,
		Conn:    conn,
		Service: wsFull,
	}

	pool.Register <- client
}

func startElectrolyserWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "startElectrolyserStatus"

	if debugOutput {
		log.Print("Electrolyser WebSocket Endpoint Hit for ", request)
	}
	conn, err := Upgrade(w, r)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	}

	client := &Client{
		ID:      r.RemoteAddr,
		Conn:    conn,
		Service: wsElectrolyser,
		Device:  request,
	}

	pool.Register <- client
}

func startFuelCellWebSocket(w http.ResponseWriter, r *http.Request) {
	if debugOutput {
		log.Print("FuelCell WebSocket Endpoint Hit")
	}
	conn, err := Upgrade(w, r)
	if err != nil {
		_, err = fmt.Fprintf(w, "%+v\n", err)
		if err != nil {
			log.Println(err)
		}
	}

	client := &Client{
		ID:      r.RemoteAddr,
		Conn:    conn,
		Service: wsFuelCell,
	}

	pool.Register <- client
}

func turnOnFCHeater(w http.ResponseWriter, _ *http.Request) {
	if debugOutput {
		log.Println("Turning on the coolant heater")
	}
	FuelCell.startHeater()
	time.AfterFunc(time.Minute, func() { FuelCell.stopHeater() })
	ReturnJSONSuccess(w)
}

func turnOffFCHeater(w http.ResponseWriter, _ *http.Request) {
	FuelCell.stopHeater()
	ReturnJSONSuccess(w)
}

func enableFc(w http.ResponseWriter, r *http.Request) {
	currentSettings.FuelCellSettings.Enabled = true
	if debugOutput {
		log.Println("Enabled")
	}
	if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
		ReturnJSONError(w, "Enable Fuel Cell", err, http.StatusInternalServerError, true)
		return
	} else {
		getFuelCell(w, r)
	}
}

func disableFc(w http.ResponseWriter, r *http.Request) {
	currentSettings.FuelCellSettings.Enabled = false
	if debugOutput {
		log.Println("Disabled")
	}
	if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
		ReturnJSONError(w, "Enable Fuel Cell", err, http.StatusInternalServerError, true)
		return
	} else {
		getFuelCell(w, r)
	}
}

func setFcPower(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := vars["power"]
	const function = "Set Fuel Cell Power"

	fPower, err := strconv.ParseFloat(request, 64)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if debugOutput {
		log.Println("set fuel cell power to ", fPower)
	}
	err = FuelCell.setTargetPower(fPower)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if err := FuelCell.updateOutput(); err != nil {
		ReturnJSONError(w, "Set Fuel Cell Power", err, http.StatusInternalServerError, true)
		return
	}
	getFuelCell(w, r)
}

func setFcBattHigh(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := vars["volts"]
	const function = "Set Fuel Cell Batt High"

	fVolts, err := strconv.ParseFloat(request, 64)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if debugOutput {
		log.Println("set fuel cell high battery limit to ", fVolts)
	}
	err = FuelCell.setTargetBattHigh(fVolts)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if err = FuelCell.updateSettings(); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	}
	getFuelCell(w, r)
}

func setFcBatLow(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := vars["volts"]
	const function = "Set Fuel Cell Batt Low"

	fVolts, err := strconv.ParseFloat(request, 64)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if debugOutput {
		log.Println("set fuel cell low battery limit to ", fVolts)
	}
	err = FuelCell.setTargetBattLow(fVolts)
	if err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if err = FuelCell.updateSettings(); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	}
	getFuelCell(w, r)
}

func startFc(w http.ResponseWriter, r *http.Request) {
	FuelCell.start()
	getFuelCell(w, r)
}

func stopFc(w http.ResponseWriter, r *http.Request) {
	FuelCell.stop()
	getFuelCell(w, r)
}

func exhaustOpen(w http.ResponseWriter, r *http.Request) {
	FuelCell.exhaustOpen()
	getFuelCell(w, r)
}

func exhaustClose(w http.ResponseWriter, r *http.Request) {
	FuelCell.exhaustClose()
	getFuelCell(w, r)
}

func setFuelCellSettings(w http.ResponseWriter, r *http.Request) {
	const function = "Set Fuel Cell Settings"
	if err := r.ParseForm(); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	}
	if floatVal, err := strconv.ParseFloat(r.FormValue("PowerDemand"), 64); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
		return
	} else {
		currentSettings.FuelCellSettings.PowerSetting = floatVal
		FuelCell.Control.TargetPower = floatVal
	}
	if floatVal, err := strconv.ParseFloat(r.FormValue("LowBattDemand"), 64); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
	} else {
		currentSettings.FuelCellSettings.LowBatterySetpoint = floatVal
	}
	if floatVal, err := strconv.ParseFloat(r.FormValue("HighBattDemand"), 64); err != nil {
		ReturnJSONError(w, function, err, http.StatusBadRequest, true)
	} else {
		currentSettings.FuelCellSettings.HighBatterySetpoint = floatVal
	}
	if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
		log.Print(err)
	}
	if FuelCell.SystemInfo.Run {
		if err := FuelCell.updateSettings(); err != nil { // Update the battery limit settings
			log.Print(err)
		}
	}
	if err := FuelCell.updateOutput(); err != nil { // Update the power setting
		log.Print(err)
	}
	http.Redirect(w, r, "/FuelCellSettings.html", http.StatusTemporaryRedirect)
}

func setRelay(w http.ResponseWriter, r *http.Request) {
	const function = "setRelay"
	var bOn bool
	vars := mux.Vars(r)
	relay := vars["relay"]
	on := vars["on"]

	on = strings.ToLower(on)
	if (on == "on") || (on == "true") || (on == "1") {
		bOn = true
	} else if (on == "off") || (on == "false") || (on == "0") {
		bOn = false
	} else {
		ReturnJSONErrorString(w, "setRelay", "Invalid value given for relay setting. Valid values are on, true, 1, off, false or 0", http.StatusBadRequest, true)
		return
	}
	relayNum, err := strconv.ParseInt(relay, 10, 8)
	if err != nil {
		if err := Relays.SetRelayByName(relay, bOn); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, true)
			return
		}
	} else {
		if err := turnOnOffRelay(uint8(relayNum), bOn); err != nil {
			ReturnJSONError(w, function, err, http.StatusBadRequest, true)
			return
		}
	}
	getFuelCell(w, r)
}

func turnOnOffRelay(relayNum uint8, bOn bool) error {
	if relayNum < uint8(len(Relays.Relays)) {
		if !bOn {
			// Turning the relay off so check if we are controlling an Electrolyser
			if el := Electrolysers.FindByRelay(relayNum); el != nil {
				// Check the stack voltage if it is running.
				if el.clientConnected && int(el.status.StackVoltage) >= currentSettings.ElectrolyserMaxStackVoltsTurnOff {
					return fmt.Errorf("electrolyser stack voltage is too high (%f). It must be below %dV", el.status.StackVoltage, currentSettings.ElectrolyserMaxStackVoltsTurnOff)
				}
			}
		} else {
			// If this is an electrolyser
			if el := Electrolysers.FindByRelay(relayNum); el != nil {
				// Check status of all electrolysers. Are any on already?
				on := false
				for _, el := range Electrolysers.Arr {
					if el.IsSwitchedOn() {
						on = true
					}
				}
				if !on {
					// This the first one to be powered up?
					switch currentSettings.WaterDumpAction {
					case ElPowered:
						go TurnOnWaterDumpValve()
					case ElPowerAndConductivity:
						_, conductivity := AnalogInputs.GetInput(7)
						if float32(currentSettings.MaximumConductivity) < conductivity {
							go TurnOnWaterDumpValve()
						}
					default:
					}
				}
			}
		}
		Relays.SetRelay(relayNum, bOn)
	} else {
		return fmt.Errorf("invalid relay number - %d", relayNum)
	}
	return nil
}

// TurnOnWaterDumpValve will dump water if we have a water dump relay configured and the current conductivity is above the minimum set.
// It will dump for the configured number of seconds
func TurnOnWaterDumpValve() {
	if currentSettings.WaterDumpRelay != 255 {
		if !Relays.GetRelay(currentSettings.WaterDumpRelay) {
			Relays.SetRelay(currentSettings.WaterDumpRelay, true)
			time.Sleep(time.Second * time.Duration(currentSettings.WaterDumpSeconds))
			Relays.SetRelay(currentSettings.WaterDumpRelay, false)
		}
	}
}

func setButton(w http.ResponseWriter, r *http.Request) {
	var bOn bool
	vars := mux.Vars(r)
	button := vars["button"]
	on := vars["on"]

	on = strings.ToLower(on)
	if (on == "on") || (on == "true") || (on == "1") {
		bOn = true
	} else if (on == "off") || (on == "false") || (on == "0") {
		bOn = false
	} else {
		ReturnJSONErrorString(w, "setButton", "Invalid value given for button setting. Valid values are on, true, 1, off, false or 0", http.StatusBadRequest, true)
		return
	}
	buttonNum, err := strconv.ParseInt(button, 10, 8)
	if err != nil {
		if err := currentSettings.setButtonByName(button, bOn); err != nil {
			ReturnJSONError(w, "setOutput", err, http.StatusBadRequest, true)
			return
		}
	} else {
		if (buttonNum >= 0) && (buttonNum < int64(len(currentSettings.Buttons))) {
			currentSettings.Buttons[buttonNum].Pressed = bOn
			if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
				ReturnJSONError(w, "setButton", err, http.StatusInternalServerError, true)
				return
			}
		} else {
			ReturnJSONErrorString(w, "setButton", fmt.Sprintf("Invalid button number - %d", buttonNum), http.StatusBadRequest, true)
			return
		}
	}
	log.Printf("Button (%s) pressed", button)
	ReturnJSONSuccess(w)
}

func setOutput(w http.ResponseWriter, r *http.Request) {
	var bOn bool
	vars := mux.Vars(r)
	output := vars["output"]
	on := vars["on"]

	on = strings.ToLower(on)
	if (on == "on") || (on == "true") || (on == "1") {
		bOn = true
	} else if (on == "off") || (on == "false") || (on == "0") {
		bOn = false
	} else {
		ReturnJSONErrorString(w, "setOutput", "Invalid value given for output setting. Valid values are on, true, 1, off, false or 0", http.StatusBadRequest, true)
		return
	}
	outputNum, err := strconv.ParseInt(output, 10, 8)
	if err != nil {
		if err := Outputs.SetOutputByName(output, bOn); err != nil {
			ReturnJSONError(w, "setOutput", err, http.StatusBadRequest, true)
			return
		}
	} else {
		if (outputNum >= 0) && (outputNum < int64(len(Outputs.Outputs))) {
			Outputs.SetOutput(uint8(outputNum), bOn)
		} else {
			ReturnJSONErrorString(w, "setOutput", fmt.Sprintf("Invalid output number - %d", outputNum), http.StatusBadRequest, true)
			return
		}
	}
	ReturnJSONSuccess(w)
}

type ACValuesType struct {
	Name          string
	ACVolts       float32
	ACAmps        float32
	ACWatts       float32
	ACWattHours   uint32
	ACHertz       float32
	ACPowerFactor float32
	Error         string
}

type DCValuesType struct {
	Name    string
	DCVolts float32
	DCAmps  float32
	Error   string
}

type SystemSettings struct {
	MaxGasPressure        uint16  `json:"maxGas"`
	GasUnits              string  `json:"gasUnits"`
	GasDisplayUnits       string  `json:"gasDisplayUnits"`
	GasCapacity           uint32  `json:"gasCapacity"`
	GasPressureInput      uint8   `json:"gasInput"`
	GasDetectorInput      uint8   `json:"gasDetector"`
	GasDetectorThreshold  uint16  `json:"gasDetected"`
	ConductivityGreenMax  float32 `json:"maxConductivityGreen"`
	ConductivityYellowMax float32 `json:"maxConductivityYellow"`
}

type SystemAlarmsType struct {
	ConductivityAlarm bool `json:"conductivityAlarm"`
	H2DetectedAlarm   bool `json:"h2Alarm"`
}

var SystemAlarms SystemAlarmsType

type JsonDataType struct {
	System            string                       `json:"System"`
	Version           string                       `json:"Version"`
	Relays            *RelaysType                  `json:"Relays"`
	Analog            *AnalogInputsType            `json:"Analog"`
	Buttons           []ButtonType                 `json:"Buttons"`
	DigitalOut        *DigitalOutputsType          `json:"DigitalOut"`
	DigitalIn         *DigitalInputsType           `json:"DigitalIn"`
	ACMeasurements    []ACValuesType               `json:"ACMeasurements"`
	DCMeasurements    []DCValuesType               `json:"DCMeasurements"`
	PanFuelCellStatus *PanStatus                   `json:"PanFuelCellStatus"`
	Electrolysers     []ElectrolyserJSONStatusType `json:"Electrolysers"`
	SystemSettings    SystemSettings               `json:"SystemSettings"`
	SystemAlarms      *SystemAlarmsType            `json:"SystemAlarms"`
	Power             []*PowerControlType          `json:"Power"`
	Hydrogen          float64                      `json:"kgH2"`
}

func getJsonStatus(indent bool) ([]byte, error) {
	var data JsonDataType

	data.Power = PowerControl
	data.System = currentSettings.Name
	data.Version = version
	data.Relays = &Relays
	data.DigitalIn = &Inputs
	data.DigitalOut = &Outputs
	data.Analog = &AnalogInputs
	data.Buttons = currentSettings.Buttons[:]
	data.Electrolysers = make([]ElectrolyserJSONStatusType, len(Electrolysers.Arr))
	count := 0
	for idx := range ACMeasurements {
		if ACMeasurements[idx].Name != "" {
			count++
		}
	}
	data.ACMeasurements = make([]ACValuesType, count)

	count = 0
	for idx := range DCMeasurements {
		if DCMeasurements[idx].Name != "" {
			count++
		}
	}
	data.DCMeasurements = make([]DCValuesType, count)

	Electrolysers.mu.Lock()
	for idx := range Electrolysers.Arr {

		data.Electrolysers[idx].Device = Electrolysers.Arr[idx].status.Device
		data.Electrolysers[idx].Name = Electrolysers.Arr[idx].status.Name
		data.Electrolysers[idx].Powered = Electrolysers.Arr[idx].status.Powered
		data.Electrolysers[idx].Model = Electrolysers.Arr[idx].status.Model
		data.Electrolysers[idx].Serial = Electrolysers.Arr[idx].status.Serial
		data.Electrolysers[idx].SystemState = Electrolysers.Arr[idx].status.SystemState
		data.Electrolysers[idx].H2Flow = Electrolysers.Arr[idx].status.H2Flow
		data.Electrolysers[idx].State = Electrolysers.Arr[idx].status.State
		data.Electrolysers[idx].ElectrolyteLevel = Electrolysers.Arr[idx].status.ElectrolyteLevel
		data.Electrolysers[idx].StackCurrent = Electrolysers.Arr[idx].status.StackCurrent
		data.Electrolysers[idx].StackVoltage = Electrolysers.Arr[idx].status.StackVoltage
		data.Electrolysers[idx].InnerH2Pressure = Electrolysers.Arr[idx].status.InnerH2Pressure
		data.Electrolysers[idx].OuterH2Pressure = Electrolysers.Arr[idx].status.OuterH2Pressure
		data.Electrolysers[idx].WaterPressure = Electrolysers.Arr[idx].status.WaterPressure
		data.Electrolysers[idx].ElectrolyteTemp = Electrolysers.Arr[idx].status.ElectrolyteTemp
		data.Electrolysers[idx].CurrentProductionRate = Electrolysers.Arr[idx].status.CurrentProductionRate
		data.Electrolysers[idx].MaxTankPressure = Electrolysers.Arr[idx].status.MaxTankPressure
		data.Electrolysers[idx].RestartPressure = Electrolysers.Arr[idx].status.RestartPressure
		data.Electrolysers[idx].DryerFailure = Electrolysers.Arr[idx].status.DryerFailure
		data.Electrolysers[idx].Dryer = Electrolysers.Arr[idx].status.Dryer
		data.Electrolysers[idx].IP = Electrolysers.Arr[idx].status.IP
		data.Electrolysers[idx].PowerRelay = Electrolysers.Arr[idx].status.PowerRelay
		data.Electrolysers[idx].Enabled = Electrolysers.Arr[idx].status.Enabled
		data.Electrolysers[idx].StackTotalProduction = Electrolysers.Arr[idx].status.StackTotalProduction
		data.Electrolysers[idx].StackStartStopCycles = Electrolysers.Arr[idx].status.StackStartStopCycles
		data.Electrolysers[idx].StackTotalRunTime = Electrolysers.Arr[idx].status.StackTotalRunTime
		data.Electrolysers[idx].StackSerialNumber = Electrolysers.Arr[idx].status.GetStackSerial()
		data.Electrolysers[idx].Warnings = Electrolysers.Arr[idx].GetWarnings()
		data.Electrolysers[idx].Errors = Electrolysers.Arr[idx].GetErrors()
	}
	Electrolysers.mu.Unlock()
	i := 0
	for idx := range ACMeasurements {
		if ACMeasurements[idx].Name != "" {
			data.ACMeasurements[i].Name = ACMeasurements[idx].Name
			data.ACMeasurements[i].ACVolts = ACMeasurements[idx].getVolts()
			data.ACMeasurements[i].ACAmps = ACMeasurements[idx].getAmps()
			data.ACMeasurements[i].ACWatts = ACMeasurements[idx].getPower()
			data.ACMeasurements[i].ACWattHours = ACMeasurements[idx].getEnergy()
			data.ACMeasurements[i].ACHertz = ACMeasurements[idx].getFrequency()
			data.ACMeasurements[i].ACPowerFactor = ACMeasurements[idx].getPowerFactor()
			data.ACMeasurements[i].Error = ACMeasurements[idx].getError()
			i++
		}
	}

	count = 0
	for idx := range DCMeasurements {
		if DCMeasurements[idx].Name != "" {
			count++
		}
	}
	data.DCMeasurements = make([]DCValuesType, count)
	i = 0
	for idx := range DCMeasurements {
		if DCMeasurements[idx].Name != "" {
			data.DCMeasurements[i].Name = DCMeasurements[idx].Name
			data.DCMeasurements[i].DCVolts = DCMeasurements[idx].getVolts()
			data.DCMeasurements[i].DCAmps = DCMeasurements[idx].getAmps()
			data.DCMeasurements[i].Error = DCMeasurements[idx].getError()
			i++
		}
	}
	data.PanFuelCellStatus = FuelCell.GetStatus()
	data.SystemSettings.GasPressureInput = currentSettings.GasPressureInput
	data.SystemSettings.MaxGasPressure = currentSettings.MaxGasPressure
	data.SystemSettings.GasUnits = currentSettings.GasUnits
	data.SystemSettings.GasDisplayUnits = currentSettings.GasDisplayUnits
	data.SystemSettings.GasCapacity = currentSettings.GasCapacity
	data.SystemSettings.GasDetectorInput = currentSettings.GasDetectorInput
	data.SystemSettings.GasDetectorThreshold = currentSettings.GasDetectorThreshold
	data.SystemSettings.ConductivityGreenMax = currentSettings.ConductivityGreenMax
	data.SystemSettings.ConductivityYellowMax = currentSettings.ConductivityYellowMax

	data.SystemAlarms = &SystemAlarms

	data.Hydrogen = CalculateHydrogenKg(data.Analog.Inputs[currentSettings.GasPressureInput].Value, data.Analog.GasTemperature)

	var JSONBytes []byte
	var err error
	if indent {
		JSONBytes, err = json.MarshalIndent(data, "", "  ")
	} else {
		JSONBytes, err = json.Marshal(data)
	}
	if err != nil {
		return nil, err
	} else {
		return JSONBytes, nil
	}
}

func getSettings(w http.ResponseWriter, _ *http.Request) {
	currentSettings.SendSettingsJSON(w)
}

type elSettingType struct {
	Name     string `json:"Name"`
	Relay    uint8  `json:"Relay"`
	HasDryer bool   `json:"Dryer"`
	Enabled  bool   `json:"enabled"`
}

/**
findNewElByName returns the index of the matching electrolyser from the given array or -1 if not found
*/
//func findNewElByName(settings []elSettingType, name string) int {
//	for el, setting := range settings {
//		if setting.Name == name {
//			return el
//		}
//	}
//	return -1
//}

/*
*
findNewElByName returns the index of the matching electrolyser from the given array or -1 if not found
*/
func findNewElByRelay(settings []elSettingType, relay uint8) int {
	for el, setting := range settings {
		if setting.Relay == relay {
			return el
		}
	}
	return -1
}

func getStatus(w http.ResponseWriter, _ *http.Request) {
	sJSON, err := getJsonStatus(true)
	setContentTypeHeader(w)
	_, err = fmt.Fprint(w, string(sJSON))
	if err != nil {
		log.Println("failed to send the status - ", err)
		return
	}
}

func getFuelCell(w http.ResponseWriter, _ *http.Request) {
	strStatus, err := FuelCell.GetStatusAsJSON()
	setContentTypeHeader(w)
	if err != nil {
		ReturnJSONError(w, "FuelCell Status", err, http.StatusInternalServerError, true)
	}
	if _, err := fmt.Fprint(w, string(strStatus)); err != nil {
		log.Println(err)
	}
}

func setContentTypeHeader(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
}

func acquireElectrolysers(w http.ResponseWriter, _ *http.Request) {
	if acquireAllElectrolysers(w) {
		ReturnJSONSuccess(w)
	}
}

func acquireAllElectrolysers(w http.ResponseWriter) bool {
	const function = "Acquire Electrolyser"
	currentSettings.acquiringElectrolysers = true
	defer func() { currentSettings.acquiringElectrolysers = false }()

	for _, el := range currentSettings.Electrolysers {
		if Relays.Relays[el.PowerRelay].On {
			ReturnJSONErrorString(w, function, "All electrolysers must be turned off before performing a search", http.StatusBadRequest, true)
			return false
		}
	}

	for _, el := range Electrolysers.Arr {
		el.setClient(net.IPv4zero)
	}

	for _, el := range Electrolysers.Arr {
		//		el := Electrolysers.Arr[idx]
		log.Printf("Searching for electrolyser on relay %d", el.powerRelay)
		if err := el.Acquire(); err != nil {
			ReturnErrorPage(w, err, http.StatusInternalServerError, true)
			return false
		}
		if txt, err := json.Marshal(el.status); err != nil {
			ReturnErrorPage(w, err, http.StatusInternalServerError, true)
			return false
		} else {
			if _, err := fmt.Fprint(w, string(txt)); err != nil {
				log.Print(err)
			}
			if elSetting := currentSettings.findElByName(el.status.Name); elSetting != nil {
				elSetting.IP = el.status.IP.String()
				elSetting.Serial = el.GetSerial()
			} else {
				log.Println("Settings for electrolyser " + el.status.Name + " not found.")
			}
		}
		if debugOutput {
			log.Println("Save settings")
		}
		if err := currentSettings.SaveSettings(currentSettings.filepath); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
			return false
		}
	}
	return true
}
