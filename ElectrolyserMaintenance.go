package main

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type ELMaintenanceType struct {
	ID                    uint64    `json:"id"`
	Name                  string    `json:"name"`
	Logged                time.Time `json:"logged"`
	StackTimeOffset       int32     `json:"stackTimeOffset"`
	SystemTimeOffset      int32     `json:"systemTimeOffset"`
	RestartCyclesOffset   int32     `json:"restartCyclesOffset"`
	StackProductionOffset float32   `json:"stackProductionOffset"`
	StackSerialNumber     string    `json:"stackSerialNumber"`
	SystemSerialNumber    string    `json:"systemSerialNumber"`
	Activity              string    `json:"activity"`
	Notes                 string    `json:"notes"`
}

// NewELMaintenance creates a new maintenance record from the current status
//func NewELMaintenance(pdb *sql.DB, elst ElectrolyserStatusType) *ELMaintenanceType {
//	elm := new(ELMaintenanceType)
//	elm.SystemSerialNumber = elst.Serial
//	elm.Name = elst.Name
//	elm.StackSerialNumber = elst.StackSerialNumber
//	if err := elm.loadLatest(pdb, elst.Name); err != nil {
//		log.Println(err)
//	}
//	return elm
//}

// LoadMaintenanceRecords loads a new set of maintenance records with the latest data
func LoadMaintenanceRecords(pdb *sql.DB) {
	for _, el := range Electrolysers.Arr {
		if err := el.status.elm.loadLatest(pdb, el.status.Name); err != nil {
			log.Print(err)
		}
		//		log.Printf("%s : Restart Cycles Offset = %d : Stack Time Offset = %d : Production Offset = %f : System Time Offset = %d : Serial = %s : Stack = %s",
		//			el.status.elm.Name, el.status.elm.RestartCyclesOffset, el.status.elm.StackTimeOffset, el.status.elm.StackProductionOffset, el.status.elm.SystemTimeOffset,
		//			el.status.elm.SystemSerialNumber, el.status.elm.StackSerialNumber)
	}
}

// loadLatest reads the database to find the latest record
func (elm *ELMaintenanceType) loadLatest(pdb *sql.DB, name string) error {
	row := pdb.QueryRow(`select ID, Name, StackTimeOffset, SystemTimeOffset, RestartCyclesOffset, StackProductionOffset, StackSerial, SystemSerial, Activity, Notes from ElectrolyserMantenanceLog eml where Name = ? order by id desc`, name)
	if err := row.Scan(&elm.ID, &elm.Name, &elm.StackTimeOffset, &elm.SystemTimeOffset, &elm.RestartCyclesOffset, &elm.StackProductionOffset, &elm.StackSerialNumber, &elm.SystemSerialNumber, &elm.Activity, &elm.Notes); err != nil {
		if err == sql.ErrNoRows {
			return nil
		} else {
			return err
		}
	}
	return nil
}

// WriteToDB writes a new record to the database
func (elm *ELMaintenanceType) WriteToDB(pdb *sql.DB) error {
	_, err := pdb.Exec("INSERT INTO `ElectrolyserMantenanceLog` (Name, StackTimeOffset, StackSerial, RestartCyclesOffset, StackProductionOffset, SystemTimeOffset, SystemSerial, Activity, Notes) VALUES (?,?,?,?,?,?,?, ?,?)",
		elm.Name, elm.StackTimeOffset, elm.StackSerialNumber, elm.RestartCyclesOffset, elm.StackProductionOffset, elm.SystemTimeOffset, elm.SystemSerialNumber, elm.Activity, elm.Notes)
	if err != nil {
		return err
	}
	return elm.loadLatest(pdb, elm.Name)
}

// resetStackHours resets the stack hours by updating the offset to the current setting and writes a new record to the database
func (elm *ELMaintenanceType) resetStackHours(pdb *sql.DB, notes string) {
	elm.StackTimeOffset = int32(Electrolysers.FindByName(elm.Name).status.StackTotalRunTime)
	elm.Activity = "Reset Stack Hours"
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

// resetSystemHours resets the system hours by updating the offset to the current setting and writes a new record to the database
func (elm *ELMaintenanceType) resetSystemHours(pdb *sql.DB, notes string) {
	elm.StackTimeOffset = int32(Electrolysers.FindByName(elm.Name).status.StackTotalRunTime)
	elm.Activity = "Reset System Hours"
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

// resetRestartCycles resets the restart cycles by updating the offset to the current setting and writes a new record to the database
func (elm *ELMaintenanceType) resetRestartCycles(pdb *sql.DB, notes string) {
	elm.RestartCyclesOffset = int32(Electrolysers.FindByName(elm.Name).status.StackStartStopCycles)
	elm.Activity = "Reset Restart Cycles"
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) replaceSystem(pdb *sql.DB, notes string) {
	elm.StackTimeOffset = 0
	elm.StackProductionOffset = 0
	elm.SystemTimeOffset = 0
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) resetStackSerialNumber(pdb *sql.DB, serial string, notes string) {
	elm.StackSerialNumber = serial
	elm.Activity = "Replace Stack"
	elm.Notes = notes
	st := Electrolysers.FindByName(elm.Name)
	elm.RestartCyclesOffset = int32(st.status.StackStartStopCycles)
	elm.StackTimeOffset = int32(st.status.StackTotalRunTime)
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	} else {
		log.Println("Updated stack serial number for ", elm.Name)
	}
}

func (elm *ELMaintenanceType) replaceElectrolyte(pdb *sql.DB, notes string) {
	elm.Notes = notes
	elm.Activity = "Replace Electrolyte"
	st := Electrolysers.FindByName(elm.Name)
	if _, err := pdb.Exec("INSERT INTO ElectrolyserMantenanceLog (Name, StackTimeOffset, StackSerial, RestartCyclesOffset, SystemTimeOffset, SystemSerial, Activity) VALUES (?,?,?,?,?,?,?)",
		elm.Name, elm.SystemTimeOffset, st.GetSerial(), elm.RestartCyclesOffset, elm.SystemTimeOffset, st.GetSerial(), "replceElectrolyte"); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) resetSystemSerialNumber(pdb *sql.DB, notes string) {
	st := Electrolysers.FindByName(elm.Name)
	elm.Activity = "Replace System"
	elm.SystemSerialNumber = st.status.Serial
	elm.Notes = notes
	elm.SystemTimeOffset = 0
	elm.StackTimeOffset = 0
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func recordElMaintenance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := strings.ToLower(vars["electrolyser"])
	notes := r.PostFormValue("notes")
	serial := r.PostFormValue("StackSerial")
	action := r.PostFormValue("action")
	log.Printf("%s : Name = %s : Serial = %s : Notes = %s", action, name, serial, notes)
	if el := Electrolysers.FindByName(name); el != nil {
		switch action {
		case "ReplaceStack":
			el.status.elm.resetStackSerialNumber(pDB, serial, notes)
		case "ReplaceSystem":
			log.Println("Electrolyser Replaced")
			el.status.elm.replaceSystem(pDB, notes)
		case "ReplaceElectrolyte":
			el.status.elm.replaceElectrolyte(pDB, notes)
		default:
			log.Print("Unknown action - ", action)
		}
	}
	serveElectrolyser(w, r)
}

func showElectrolyserMaintenance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	request := strings.ToLower(vars["electrolyser"])
	const function = "showElectrolyserMaintenance"

	el := Electrolysers.FindByName(request)
	if el == nil {
		ReturnJSONErrorString(w, function, "Electrolyser not found - "+request, http.StatusBadRequest, false)
		return
	}

	if fileContent, err := os.ReadFile(webFiles + "/ElectrolyserMaintenance.html"); err != nil {
		ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		return
	} else {
		replacements := make(ReplacementsType)
		replacements["title"] = currentSettings.Name + " - " + request
		replacements["name"] = request
		replacements["version"] = version

		if _, err := fmt.Fprint(w, replaceText(string(fileContent), replacements)); err != nil {
			ReturnJSONError(w, function, err, http.StatusInternalServerError, true)
		}
	}
}
