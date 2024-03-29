package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type HydrogenParamsType struct {
	gas   uint8
	m     float64
	c     float64
	pv    float64
	field string
}

func NewHydrogenParams() *HydrogenParamsType {
	hp := new(HydrogenParamsType)
	hp.gas = currentSettings.GasPressureInput
	hp.m = float64(currentSettings.AnalogChannels[hp.gas].calibrationMultiplier)
	hp.c = float64(currentSettings.AnalogChannels[hp.gas].calibrationConstant)
	hp.pv = float64(currentSettings.GasCapacity)
	hp.field = fmt.Sprintf("a%d", currentSettings.GasPressureInput)
	if currentSettings.GasUnits == "psi" {
		hp.pv = hp.pv / 14.5038
	}
	return hp
}

const AnalogBySecond = `select UNIX_TIMESTAMP(logged) as logged,
	((a0 * ?) + ?) as a0,
	((a1 * ?) + ?) as a1,
	((a2 * ?) + ?) as a2,
	((a3 * ?) + ?) as a3,
	((a4 * ?) + ?) as a4,
	((a5 * ?) + ?) as a5,
	((a6 * ?) + ?) as a6,
	((a7 * ?) + ?) as a7,
	temperature as temperature,
	(((('h2' * ?) + ?) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) as h2kg
	from IOValues
   where logged between ? and ?`

func SendAnalogBySecond(w http.ResponseWriter, start time.Time, end time.Time) {
	hp := NewHydrogenParams()
	strSQL := strings.Replace(AnalogBySecond, "'h2'", hp.field, -1)
	SendDataAsJSON(w, "GetAnalogBySecond", strSQL,
		currentSettings.AnalogChannels[0].calibrationMultiplier, currentSettings.AnalogChannels[0].calibrationConstant,
		currentSettings.AnalogChannels[1].calibrationMultiplier, currentSettings.AnalogChannels[1].calibrationConstant,
		currentSettings.AnalogChannels[2].calibrationMultiplier, currentSettings.AnalogChannels[2].calibrationConstant,
		currentSettings.AnalogChannels[3].calibrationMultiplier, currentSettings.AnalogChannels[3].calibrationConstant,
		currentSettings.AnalogChannels[4].calibrationMultiplier, currentSettings.AnalogChannels[4].calibrationConstant,
		currentSettings.AnalogChannels[5].calibrationMultiplier, currentSettings.AnalogChannels[5].calibrationConstant,
		currentSettings.AnalogChannels[6].calibrationMultiplier, currentSettings.AnalogChannels[6].calibrationConstant,
		currentSettings.AnalogChannels[7].calibrationMultiplier, currentSettings.AnalogChannels[7].calibrationConstant,
		hp.m, hp.c, hp.pv,
		start, end)
}

const AnalogByMinute = `select min(UNIX_TIMESTAMP(logged)) as logged,
	((AVG(a0) * ?) + ?) as a0,
	((AVG(a1) * ?) + ?) as a1,
	((AVG(a2) * ?) + ?) as a2,
	((AVG(a3) * ?) + ?) as a3,
	((AVG(a4) * ?) + ?) as a4,
	((AVG(a5) * ?) + ?) as a5,
	((AVG(a6) * ?) + ?) as a6,
	((AVG(a7) * ?) + ?) as a7,
	AVG(temperature) as temperature,
	(((('h2' * ?) + ?) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) as h2kg
	from IOValues
  where logged between ? and ?
  group by UNIX_TIMESTAMP(logged) div 60`

func SendAnalogByMinute(w http.ResponseWriter, start time.Time, end time.Time) {
	hp := NewHydrogenParams()
	strSQL := strings.Replace(AnalogByMinute, "'h2'", hp.field, -1)
	SendDataAsJSON(w, "GetAnalogByMinute", strSQL,
		currentSettings.AnalogChannels[0].calibrationMultiplier, currentSettings.AnalogChannels[0].calibrationConstant,
		currentSettings.AnalogChannels[1].calibrationMultiplier, currentSettings.AnalogChannels[1].calibrationConstant,
		currentSettings.AnalogChannels[2].calibrationMultiplier, currentSettings.AnalogChannels[2].calibrationConstant,
		currentSettings.AnalogChannels[3].calibrationMultiplier, currentSettings.AnalogChannels[3].calibrationConstant,
		currentSettings.AnalogChannels[4].calibrationMultiplier, currentSettings.AnalogChannels[4].calibrationConstant,
		currentSettings.AnalogChannels[5].calibrationMultiplier, currentSettings.AnalogChannels[5].calibrationConstant,
		currentSettings.AnalogChannels[6].calibrationMultiplier, currentSettings.AnalogChannels[6].calibrationConstant,
		currentSettings.AnalogChannels[7].calibrationMultiplier, currentSettings.AnalogChannels[7].calibrationConstant,
		hp.m, hp.c, hp.pv,
		start, end)
}

type AnalogInputType struct {
	Name  string  `json:"Name"`
	Raw   uint16  `json:"Raw"`
	Value float32 `json:"Value"`
}

type AnalogInputsType struct {
	Inputs         [8]AnalogInputType `json:"Inputs"`
	Temperature    int16              `json:"Temperature"`
	RawTemperature uint16             `json:"RawTemperature"`
	VrefValue      uint16             `json:"VrefValue"`
	GasTemperature float64            `json:"GasTemperature"`
	mu             sync.Mutex
}

func (ai *AnalogInputsType) InitAnalogInputs() {
	for idx := range ai.Inputs {
		ai.Inputs[idx].Name = fmt.Sprintf("input-%d", idx)
	}
}

const H2ByDay = `SELECT MIN(ROUND(UNIX_TIMESTAMP(logged_start))) as logged
     , SUM(ROUND(GREATEST(end_h2_calculated - start_h2_calculated, 0), 2)) AS increase
     , SUM(ROUND(GREATEST(start_h2_calculated - end_h2_calculated, 0), 2)) as decrease
  FROM (SELECT DISTINCT FIRST_VALUE(logged) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as logged_start
             , FIRST_VALUE((((hydrogen * ?) + (?)) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as start_h2_calculated
             , LAST_VALUE((((hydrogen * ?) + (?)) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as end_h2_calculated
		  FROM (select min(logged) as logged, avg('h2') as hydrogen , AVG(temperature) as temperature
		          FROM IOValues
		         WHERE logged BETWEEN ? AND ? group by UNIX_TIMESTAMP(logged) DIV 60) as avgvals) as vals GROUP BY UNIX_TIMESTAMP(logged_start) DIV 86400`

// Get the SQL query to fetch hydrogen generation or consumption by day
func SendHydrogenByDay(w http.ResponseWriter, start time.Time, end time.Time) {
	hp := NewHydrogenParams()
	log.Printf("hp.m = %f | hp.c = %f | hp.pv = %f", hp.m, hp.c, hp.pv)
	strSQL := strings.Replace(H2ByDay, "'h2'", hp.field, -1)
	SendDataAsJSON(w, "SendHydrogenByDay", strSQL, hp.m, hp.c, hp.pv, hp.m, hp.c, hp.pv, start, end)
}

//const H2ByHour = `SELECT ROUND(UNIX_TIMESTAMP(logged_start)) as logged
//     , ROUND(GREATEST(end_h2_calculated - start_h2_calculated, 0), 2) AS increase
//     , ROUND(GREATEST(start_h2_calculated - end_h2_calculated, 0), 2) as decrease
//  FROM (SELECT DISTINCT FIRST_VALUE(logged) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as logged_start
//             , FIRST_VALUE(((('h2' * ?) + (?)) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as start_h2_calculated
//             , LAST_VALUE(((('h2' * ?) + (?)) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as end_h2_calculated
//		  FROM IOValues
//		 WHERE logged BETWEEN ? AND ?) as vals ORDER BY logged`

const H2ByHour = `SELECT ROUND(UNIX_TIMESTAMP(logged_start)) as logged
     , ROUND(GREATEST(end_h2_calculated - start_h2_calculated, 0), 2) AS increase
     , ROUND(GREATEST(start_h2_calculated - end_h2_calculated, 0), 2) as decrease
  FROM (SELECT DISTINCT FIRST_VALUE(logged) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as logged_start
             , FIRST_VALUE((((hydrogen * ?) + (?)) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as start_h2_calculated
             , LAST_VALUE((((hydrogen * ?) + (?)) * ? * 0.2016) / ((273.15 + temperature) * 8.314)) OVER (PARTITION BY (UNIX_TIMESTAMP(logged) DIV 3600) ORDER BY logged ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as end_h2_calculated
		  FROM (select min(logged) as logged, avg('h2') as hydrogen , AVG(temperature) as temperature
		          FROM IOValues
		         WHERE logged BETWEEN ? AND ? group by UNIX_TIMESTAMP(logged) DIV 60) as avgvals) as vals ORDER BY logged`

// Get the SQL query to fetch hydrogen generation or consumption by hour
func SendHydrogenByHour(w http.ResponseWriter, start time.Time, end time.Time) {
	hp := NewHydrogenParams()
	strSQL := strings.Replace(H2ByHour, "'h2'", hp.field, -1)
	SendDataAsJSON(w, "SendHydrogenByHour", strSQL, hp.m, hp.c, hp.pv, hp.m, hp.c, hp.pv, start, end)
}

func (ai *AnalogInputsType) SetAnanlog0To3(data [8]byte) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.Inputs[0].Raw = binary.LittleEndian.Uint16(data[0:2])
	ai.Inputs[0].Value = (float32(ai.Inputs[0].Raw) * currentSettings.AnalogChannels[0].calibrationMultiplier) + currentSettings.AnalogChannels[0].calibrationConstant
	ai.Inputs[1].Raw = binary.LittleEndian.Uint16(data[2:4])
	ai.Inputs[1].Value = (float32(ai.Inputs[1].Raw) * currentSettings.AnalogChannels[1].calibrationMultiplier) + currentSettings.AnalogChannels[1].calibrationConstant
	ai.Inputs[2].Raw = binary.LittleEndian.Uint16(data[4:6])
	ai.Inputs[2].Value = (float32(ai.Inputs[2].Raw) * currentSettings.AnalogChannels[2].calibrationMultiplier) + currentSettings.AnalogChannels[2].calibrationConstant
	ai.Inputs[3].Raw = binary.LittleEndian.Uint16(data[6:8])
	ai.Inputs[3].Value = (float32(ai.Inputs[3].Raw) * currentSettings.AnalogChannels[3].calibrationMultiplier) + currentSettings.AnalogChannels[3].calibrationConstant
}

func (ai *AnalogInputsType) SetAnanlog4To7(data [8]byte) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.Inputs[4].Raw = binary.LittleEndian.Uint16(data[0:2])
	ai.Inputs[4].Value = (float32(ai.Inputs[4].Raw) * currentSettings.AnalogChannels[4].calibrationMultiplier) + currentSettings.AnalogChannels[4].calibrationConstant
	ai.Inputs[5].Raw = binary.LittleEndian.Uint16(data[2:4])
	ai.Inputs[5].Value = (float32(ai.Inputs[5].Raw) * currentSettings.AnalogChannels[5].calibrationMultiplier) + currentSettings.AnalogChannels[5].calibrationConstant
	ai.Inputs[6].Raw = binary.LittleEndian.Uint16(data[4:6])
	ai.Inputs[6].Value = (float32(ai.Inputs[6].Raw) * currentSettings.AnalogChannels[6].calibrationMultiplier) + currentSettings.AnalogChannels[6].calibrationConstant
	ai.Inputs[7].Raw = binary.LittleEndian.Uint16(data[6:8])
	ai.Inputs[7].Value = (float32(ai.Inputs[7].Raw) * currentSettings.AnalogChannels[7].calibrationMultiplier) + currentSettings.AnalogChannels[7].calibrationConstant
}

func (ai *AnalogInputsType) SetAnanlogInternal(data [8]byte) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.Temperature = int16(binary.LittleEndian.Uint16(data[0:2]))
	ai.RawTemperature = binary.LittleEndian.Uint16(data[2:4])
	ai.VrefValue = binary.LittleEndian.Uint16(data[4:6])
}

func (ai *AnalogInputsType) SetTemperature(temperature float64) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.GasTemperature = temperature
}

func (ai *AnalogInputsType) SetInputName(port uint8, name string) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.Inputs[port].Name = strings.ToLower(name)
}

func (ai *AnalogInputsType) GetInputByName(port string) (int32, error) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	port = strings.ToLower(port)
	switch port {
	case "temperature":
		return int32(ai.Temperature), nil
	case "raw temperature":
		return int32(ai.RawTemperature), nil
	case "vref":
		return int32(ai.VrefValue), nil
	default:
		for _, ip := range ai.Inputs {
			if ip.Name == port {
				return int32(ip.Value), nil
			}
		}
	}
	return 0, fmt.Errorf("invalid input name %s", port)
}

func (ai *AnalogInputsType) GetTemperature() float64 {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	return ai.GasTemperature
}

func (ai *AnalogInputsType) GetCPUTemperature() (rawValue uint16, celsiusValue float32) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	return ai.RawTemperature, float32(ai.Temperature) / 100
}

func (ai *AnalogInputsType) GetVREF() uint16 {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	return ai.VrefValue
}

func (ai *AnalogInputsType) GetInput(port uint8) (uint16, float32) {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	return ai.Inputs[port].Raw, ai.Inputs[port].Value
}

func (ai *AnalogInputsType) GetRawInput(port uint8) uint16 {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	return ai.Inputs[port].Raw
}

// Calculate the hydrogen in kg
func CalculateHydrogenKg(pressure float32, gasTemperature float64) float64 {
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
		gasPressure = float64(pressure)
	} else {
		// stupid units - convert to SI
		gasPressure = float64(pressure) / 14.50377
	}
	return (volume * gasPressure * C1) / (T1 + gasTemperature)
}
