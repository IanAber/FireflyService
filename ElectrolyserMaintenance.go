package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"
)

type ELMaintenanceType struct {
	ID                    uint64    `json:"id"`
	Name                  string    `json:"name"`
	Logged                time.Time `json:"logged"`
	StackHoursOffset      uint32    `json:"stackHoursOffset"`
	SystemHoursOffset     uint32    `json:"systemHoursOffset"`
	RestartCyclesOffset   uint32    `json:"restartCyclesOffset"`
	StackProductionOffset float32   `json:"stackProductionOffset"`
	StackSerialNumber     string    `json:"stackSerialNumber"`
	SystemSerialNumber    string    `json:"systemSerialNumber"`
	Activity              string    `json:"activity"`
	Notes                 string    `json:"notes"`
}

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

func (elm *ELMaintenanceType) loadLatest(pdb *sql.DB, name string) error {
	row := pdb.QueryRow(`select ID, Name, StackHoursOffset, SystemHoursOffset, RestartCyclesOffset, StackProductionOffset, StackSerial, SystemSerial, Activity, Notes from ElectrolyserMantenanceLog eml where ? = 'EL1' order by id desc`, name)
	if err := row.Scan(&elm.ID, &elm.Name, &elm.StackHoursOffset, &elm.SystemHoursOffset, &elm.RestartCyclesOffset, &elm.StackProductionOffset, &elm.StackSerialNumber, &elm.SystemSerialNumber, &elm.Activity, &elm.Notes); err != nil {
		if err == sql.ErrNoRows {
			return nil
		} else {
			return err
		}
	}
	return nil
}

func (elm *ELMaintenanceType) WriteToDB(pdb *sql.DB) error {
	_, err := pdb.Exec("INSERT INTO `ElectrolyserMantenanceLog` (Name, StackHoursOffset, StackSerial, RestartCyclesOffset, SystemHoursOffset, SystemSerial, Activity, Notes) VALUES (?,?,?,?,?,?,?,?)",
		elm.Name, elm.StackHoursOffset, elm.StackSerialNumber, elm.RestartCyclesOffset, elm.SystemHoursOffset, elm.SystemSerialNumber, elm.Activity, elm.Notes)
	if err != nil {
		return err
	}
	return elm.loadLatest(pdb, elm.Name)
}

func (elm *ELMaintenanceType) resetStackHours(pdb *sql.DB, notes string) {
	elm.StackHoursOffset = Electrolysers.FindByName(elm.Name).status.StackTotalRunTime
	elm.Activity = "Reset Stack Hours"
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) resetSystemHours(pdb *sql.DB, notes string) {
	elm.StackHoursOffset = Electrolysers.FindByName(elm.Name).status.StackTotalRunTime
	elm.Activity = "Reset System Hours"
	elm.Notes = notes
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) resetRestartCycles(pdb *sql.DB, notes string) {
	elm.RestartCyclesOffset = Electrolysers.FindByName(elm.Name).status.StackStartStopCycles
	elm.Activity = "Reset Restart Cycles"
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
	elm.RestartCyclesOffset = st.status.StackStartStopCycles
	elm.StackHoursOffset = st.status.StackTotalRunTime
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) replaceElectrolyte(pdb *sql.DB, notes string) {
	elm.Notes = notes
	elm.Activity = "Replace Electrolyte"
	st := Electrolysers.FindByName(elm.Name)
	if _, err := pdb.Exec("INSERT INTO ElectrolyserMantenanceLog (Name, StackHoursOffset, StackSerial, RestartCyclesOffset, SystemHoursOffset, SystemSerial, Activity) VALUES (?,?,?,?,?,?,?)",
		elm.Name, elm.SystemHoursOffset, st.GetSerial(), elm.RestartCyclesOffset, elm.SystemHoursOffset, st.GetSerial(), "replceElectrolyte"); err != nil {
		log.Println(err)
	}
}

func (elm *ELMaintenanceType) resetSystemSerialNumber(pdb *sql.DB, notes string) {
	st := Electrolysers.FindByName(elm.Name)
	elm.Activity = "Replace System"
	elm.SystemSerialNumber = st.status.Serial
	elm.Notes = notes
	elm.SystemHoursOffset = 0
	elm.StackHoursOffset = 0
	if err := elm.WriteToDB(pdb); err != nil {
		log.Println(err)
	}
}

func recordElMaintenance(_ http.ResponseWriter, r *http.Request) {
	notes := r.PostFormValue("notes")
	name := r.PostFormValue("name")
	log.Printf("Name = %s : Notes = %s", name, notes)
}
