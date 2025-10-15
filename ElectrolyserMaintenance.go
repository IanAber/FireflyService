package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type ELMaintenanceType struct {
	ID                    uint64    `json:"id"`
	Name                  string    `json:"name"`                  // Name of the electrolyser
	Logged                time.Time `json:"logged"`                // Time and date of the maintenance activity
	StackTimeOffset       int32     `json:"stackTimeOffset"`       // Used to adjust stack time when it is replaced
	SystemTimeOffset      int32     `json:"systemTimeOffset"`      // Used to adjust system run time
	RestartCyclesOffset   int32     `json:"restartCyclesOffset"`   // Used to adjust restart cycles
	StackProductionOffset float32   `json:"stackProductionOffset"` // Used to adjust stack production if it is changed
	StackSerialNumber     string    `json:"stackSerialNumber"`     // Used to override the stack serial number from the machine
	SystemSerialNumber    string    `json:"systemSerialNumber"`    // Used to override the serial number from the machine
	Activity              string    `json:"activity"`              // Records the maintenance activity performed
	Notes                 string    `json:"notes"`                 // Free text to record notes regarding the maintenance activity
}

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
	elm = new(ELMaintenanceType) // Initialise a new record
	elm.Name = name
	elm.SystemTimeOffset = 0
	elm.SystemSerialNumber = ""
	elm.StackTimeOffset = 0
	elm.StackProductionOffset = 0
	elm.StackSerialNumber = ""
	elm.RestartCyclesOffset = 0
	elm.Activity = ""
	elm.Notes = ""
	elm.Logged = time.Now()

	row := pdb.QueryRow(`select ID, Name, StackTimeOffset, SystemTimeOffset, RestartCyclesOffset, StackProductionOffset, StackSerial, SystemSerial, Activity, Notes from ElectrolyserMaintenanceLog eml where Name = ? order by id desc limit 1`, name)
	if err := row.Scan(&elm.ID, &elm.Name, &elm.StackTimeOffset, &elm.SystemTimeOffset, &elm.RestartCyclesOffset, &elm.StackProductionOffset, &elm.StackSerialNumber, &elm.SystemSerialNumber, &elm.Activity, &elm.Notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil // This is not an error, it just indicates that we do not yet have any records for this electrolyser
		} else {
			return err
		}
	}
	return nil
}

// WriteToDB writes a new record to the database
func (elm *ELMaintenanceType) WriteToDB(pdb *sql.DB) error {
	_, err := pdb.Exec("INSERT INTO `ElectrolyserMaintenanceLog` (Name, StackTimeOffset, StackSerial, RestartCyclesOffset, StackProductionOffset, SystemTimeOffset, SystemSerial, Activity, Notes) VALUES (?,?,?,?,?,?,?, ?,?)",
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
	elm.SystemTimeOffset = int32(Electrolysers.FindByName(elm.Name).status.SystemHours)
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
	if el := Electrolysers.FindByName(elm.Name); el == nil {
		log.Printf("Failed to find %s by name", elm.Name)
		return
	} else {
		elm.SystemTimeOffset = int32(el.status.SystemHours)
		elm.SystemSerialNumber = el.status.Serial
		elm.StackTimeOffset = int32(el.status.StackTotalRunTime)
		elm.StackProductionOffset = float32(el.status.StackTotalProduction)
		elm.StackSerialNumber = el.status.StackSerialNumber
		elm.RestartCyclesOffset = int32(el.status.StackStartStopCycles)
		elm.Activity = "Replace System"
		elm.Notes = notes
		if err := elm.WriteToDB(pdb); err != nil {
			log.Println(err)
		}
	}
}

func (elm *ELMaintenanceType) resetStackSerialNumber(pdb *sql.DB, el *ElectrolyserType, serial string, notes string) {
	elm.Activity = "Replace Stack"
	elm.SystemSerialNumber = el.status.Serial
	elm.StackSerialNumber = serial
	elm.RestartCyclesOffset = 0
	elm.Notes = notes
	elm.StackTimeOffset = int32(el.status.StackTotalRunTime)
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) replaceElectrolyte(pdb *sql.DB, notes string) {
	elm.Activity = "Replace Electrolyte"
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) resetSystemSerialNumber(el *ElectrolyserType, pdb *sql.DB, notes string) {
	elm.Activity = "Reset Serial Number"
	elm.SystemSerialNumber = el.status.Serial
	elm.StackSerialNumber = el.status.StackSerialNumber
	elm.Notes = notes
	elm.SystemTimeOffset = 0
	elm.StackTimeOffset = 0
	elm.RestartCyclesOffset = 0
	elm.Name = el.status.Name
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
			el.status.elm.resetStackSerialNumber(pDB, el, serial, notes)
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
