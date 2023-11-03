package main

import (
	"fmt"
	"sync"
)

/*
Query strings to return historical data
*/
const ACDataByMinute = `select UNIX_TIMESTAMP(logged) as logged
						,avg(A_volts) as A_volts
						,avg(A_amps) as A_amps
						,avg(A_hertz) as A_hertz
						,avg(A_watts) as A_watts
						,avg(A_powerFactor) as A_powerFactor
						,avg(B_volts) as B_volts
						,avg(B_amps) as B_amps
						,avg(B_hertz) as B_hertz
						,avg(B_watts) as B_watts
						,avg(B_powerFactor) as B_powerFactor
						,avg(C_volts) as C_volts
						,avg(C_amps) as C_amps
						,avg(C_hertz) as C_hertz
						,avg(C_watts) as C_watts
						,avg(C_powerFactor) as C_powerFactor
						,avg(D_volts) as D_volts
						,avg(D_amps) as D_amps
						,avg(D_hertz) as D_hertz
						,avg(D_watts) as D_watts
						,avg(D_powerFactor) as D_powerFactor
					from ACValues
				   where logged between ? and ?
			    group by UNIX_TIMESTAMP(logged) div 60`

const ACDataBySecond = `select UNIX_TIMESTAMP(logged) as logged
						,A_volts
						,A_amps
						,A_hertz
						,A_watts
						,A_powerFactor
						,B_volts
						,B_amps
						,B_hertz
						,B_watts
						,B_powerFactor
						,C_volts
						,C_amps
						,C_hertz
						,C_watts
						,C_powerFactor
						,D_volts
						,D_amps
						,D_hertz
						,D_watts
						,D_powerFactor
					from ACValues
				   where logged between ? and ?`

const ACValuesInsertStatement = "INSERT INTO firefly.ACValues (`A_volts`, `A_amps`, `A_watts`, `A_hertz`, `A_powerFactor`, `B_volts`, `B_amps`, `B_watts`, `B_hertz`, `B_powerFactor`, `C_volts`, `C_amps`, `C_watts`, `C_hertz`, `C_powerFactor`, `D_volts`, `D_amps`, `D_watts`, `D_hertz`, `D_powerFactor`) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);"

type ACMeasurementsType struct {
	Name        string
	Volts       float32
	Amps        float32
	Power       float32
	WattHours   uint32
	Frequency   float32
	PowerFactor float32
	Error       uint8
	mu          sync.Mutex
}

func (ac *ACMeasurementsType) InitACMeasurement() {
	ac.Name = ""
	ac.Volts = 0.0
	ac.Amps = 0.0
	ac.Power = 0.0
	ac.WattHours = 0
	ac.Frequency = 60.0
	ac.PowerFactor = 1.0
}

func (ac *ACMeasurementsType) setVolts(v uint16) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.Volts = float32(v) / 10.0
}

func (ac *ACMeasurementsType) setAmps(i uint32) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.Amps = float32(i) / 1000.0
}

func (ac *ACMeasurementsType) setPower(p uint32) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.Power = float32(p) / 10.0
}

func (ac *ACMeasurementsType) setEnergy(whr uint32) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.WattHours = whr
}

func (ac *ACMeasurementsType) setFrequency(f uint16) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.Frequency = float32(f) / 10.0
}

func (ac *ACMeasurementsType) setPowerFactor(pf uint16) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.PowerFactor = float32(pf) / 100
}

func (ac *ACMeasurementsType) getVolts() float32 {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return (ac.Volts)
}

func (ac *ACMeasurementsType) getAmps() float32 {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return (ac.Amps)
}

func (ac *ACMeasurementsType) getPower() float32 {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return (ac.Power)
}

func (ac *ACMeasurementsType) getEnergy() uint32 {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return ac.WattHours
}

func (ac *ACMeasurementsType) getFrequency() float32 {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return ac.Frequency
}

func (ac *ACMeasurementsType) getPowerFactor() float32 {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	return ac.PowerFactor
}

func (ac *ACMeasurementsType) getError() string {
	switch ac.Error {
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
		return fmt.Sprintf("Error = %0x", ac.Error)
	}
}

func (ac *ACMeasurementsType) setError(err uint8) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.Error = err
}
