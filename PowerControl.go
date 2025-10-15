package main

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"sync"
	"time"
)

const InsertPowerSQL = "INSERT INTO firefly.Power(source, amps, volts, soc, maxchargeamps, frequency, solar) VALUES (?,?,?,?,?,?,?)"

type PowerControlType struct {
	Source        string    `json:"source"`
	Voltage       float64   `json:"volts"`
	Current       float64   `json:"amps"`
	StateOfCharge float64   `json:"soc"`
	MaxChargeAmps float64   `json:"bmsChargeCurrentMax"`
	Frequency     float64   `json:"hz"`
	Solar         float64   `json:"solar"`
	LastUpdated   time.Time `json:"lastUpdated"`
	updated       bool
	mu            sync.Mutex
}

func NewPowerControl(source string) *PowerControlType {
	pc := new(PowerControlType)
	pc.Source = source
	return pc
}

func FindPowerControl(source string) *PowerControlType {
	for _, pc := range PowerControl {
		if pc.Source == source {
			return pc
		}
	}
	// If we get to here we don't have a matching PowerControl entry so we should add a new one
	pc := NewPowerControl(source)
	PowerControl = append(PowerControl, pc)
	return pc
}

// setVoltage sets the battery voltage in volts
func (pc *PowerControlType) setVoltage(v float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Voltage = v
	pc.updated = true
}

// setCurrent sets the battery current in amps. Positive should be charging
func (pc *PowerControlType) setCurrent(i float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Current = i
	pc.updated = true
}

// setStateFoCharge set a percentage for the current battery SOC
func (pc *PowerControlType) setStateOfCharge(s float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.StateOfCharge = s
	pc.updated = true
}

// setMaxChargeCurrent sets the value provided by the BMS
func (pc *PowerControlType) setMaxChargeCurrent(f float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.MaxChargeAmps = f
	pc.updated = true
}

// setFrequency sets the mains frequency used to help determine available solar power
func (pc *PowerControlType) setFrequency(f float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Frequency = f
	pc.updated = true
}

// setSolar sets the solar production in watts
func (pc *PowerControlType) setSolar(s float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Solar = s
	pc.updated = true
}

// setAll sets volts, amps, soc and optionally f. If f < 1 ignore it. If solar < 0 ignore it.
func (pc *PowerControlType) setAll(v float64, i float64, soc float64, f float64, maxCharge float64, solar float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.Voltage = v
	pc.Current = i
	pc.StateOfCharge = soc
	pc.MaxChargeAmps = maxCharge
	if solar >= 0.0 {
		pc.Solar = solar
	}
	if f > 1 {
		pc.Frequency = f
	}
	pc.updated = true
}

func (pc *PowerControlType) getValues() (source string, current int32, voltage uint16, soc uint16, maxchargecurrent int64, frequency uint16, solar uint32, updated bool) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.Source, int32(math.Round(pc.Current * 100)), uint16(math.Round(pc.Voltage * 100)), uint16(math.Round(pc.StateOfCharge * 100)), int64(math.Round(pc.MaxChargeAmps * 100)), uint16(math.Round(pc.Frequency * 100)), uint32(pc.Solar), pc.updated
}

func (pc *PowerControlType) logData(stmt *sql.Stmt) {
	// source, amps, volts, soc, maxchargeamps, frequency, solar
	source, current, voltage, soc, maxchargeamps, frequency, solar, updated := pc.getValues()
	// Update the database if values have changed since last time
	if updated {
		if _, err := stmt.Exec(source, current, voltage, soc, maxchargeamps, frequency, solar); err != nil {
			log.Print(err)
		} else {
			log.Printf("Wrote %s %d %d %d %d %d %d", source, current, voltage, soc, maxchargeamps, frequency, solar)
		}
	}
	pc.updated = false
}

const PowerBySecond = `select UNIX_TIMESTAMP(logged) as logged,
	amps / 100 as iBatt,
	volts / 100 as vBatt,
	soc / 100 as soc,
	frequency / 100 as hz,
	maxchargeamps / 100 as maxCharge,
	solar / 1000 as solar
	from Power
   where logged between ? and ? AND source = ?`

func SendPowerBySecond(w http.ResponseWriter, start time.Time, end time.Time, source string) {
	SendDataAsJSON(w, "GetAnalogBySecond", PowerBySecond, start, end, source)
}

const PowerByMinute = `select min(UNIX_TIMESTAMP(logged)) as logged,
	AVG(amps) / 100 as iBatt,
	AVG(volts) / 100 as vBatt,
	AVG(soc) / 100 as soc,
	AVG(frequency) / 100 as hz,
	Avg(maxchargeamps) / 100 as maxCharge,
	AVG(solar) / 1000 as solar
	from Power
  where logged between ? and ? AND source = ?
  group by UNIX_TIMESTAMP(logged) div 60`

func SendPowerByMinute(w http.ResponseWriter, start time.Time, end time.Time, source string) {
	SendDataAsJSON(w, "GetPowerByMinute", PowerByMinute, start, end, source)
}
