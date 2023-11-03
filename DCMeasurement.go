package main

import (
	"fmt"
	"sync"
)

/*
Query strings to return historical data
*/
const DCDataByMinute = `select UNIX_TIMESTAMP(logged) as logged
						,avg(A_volts) as A_volts
						,avg(A_amps) as A_amps
     					,avg(A_amps * A_volts) as A_watts
						,avg(B_volts) as B_volts
						,avg(B_amps) as B_amps
     					,avg(B_amps * B_volts) as B_watts
						,avg(C_volts) as C_volts
						,avg(C_amps) as C_amps
     					,avg(C_amps * C_volts) as C_watts
						,avg(D_volts) as D_volts
						,avg(D_amps) as D_amps
     					,avg(D_amps * D_volts) as D_watts
					from DCValues
				   where logged between ? and ?
			    group by UNIX_TIMESTAMP(logged) div 60`

const DCDataBySecond = `select UNIX_TIMESTAMP(logged) as logged
						,A_volts
						,A_amps
     					,A_amps * A_volts as A_watts
						,B_volts
						,B_amps
     					,B_amps * B_volts as B_watts
						,C_volts
						,C_amps
     					,C_amps * C_volts as C_watts
						,D_volts
						,D_amps
     					,D_amps * D_volts as D_watts
					from DCValues
				   where logged between ? and ?`

const DCValuesInsertStatement = "INSERT INTO firefly.DCValues (`A_volts`, `A_amps`, `B_volts`, `B_amps`, `C_volts`, `C_amps`, `D_volts`, `D_amps`) VALUES(?, ?, ?, ?, ?, ?, ?, ?);"

type DCMeasurementsType struct {
	Name  string
	Volts float32
	Amps  float32
	Error uint8
	mu    sync.Mutex
}

func (dc *DCMeasurementsType) InitDCMeasurement() {
	dc.Name = ""
	dc.Volts = 0.0
	dc.Amps = 0.0
}

func (dc *DCMeasurementsType) setVolts(v uint16) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.Volts = float32(v) / 100.0
}

func (dc *DCMeasurementsType) setAmps(i uint32) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.Amps = float32(int32(i)) / 1000.0
}

func (dc *DCMeasurementsType) getVolts() float32 {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	return (dc.Volts)
}

func (dc *DCMeasurementsType) getAmps() float32 {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	return (dc.Amps)
}

func (dc *DCMeasurementsType) getPower() float32 {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	return (dc.Volts * dc.Amps)
}

func (dc *DCMeasurementsType) getError() string {
	switch dc.Error {
	case 0:
		return ""
	case 0xFF:
		return "Not a Master"
	case 0xFE:
		return "Polling Error"
	case 0xFD:
		return "Buffer Overflow"
	case 0xFC:
		return "Bad CRC"
	case 0xFB:
		return "Exception"
	case 0xFA:
		return "Bad Size"
	case 0xF9:
		return "Bad Address"
	case 0xF8:
		return "Timeout"
	case 0xF7:
		return "Bad Slave ID"
	case 0xF6:
		return "Bad TCP ID"
	default:
		return fmt.Sprintf("Error = %0x", dc.Error)
	}
}

func (dc *DCMeasurementsType) setError(err uint8) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.Error = err
}
