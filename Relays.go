package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
)

type RelayType struct {
	Name string `json:"Name"`
	On   bool   `json:"On"`
}

type RelaysType struct {
	Relays            [16]RelayType `json:"Relays"`
	lastReportedValue uint16
	mu                sync.Mutex
}

func (rl *RelaysType) InitRelays() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for idx := range rl.Relays {
		rl.Relays[idx].Name = fmt.Sprintf("relay-%d", idx)
		rl.Relays[idx].On = false
	}
}

func (rl *RelaysType) SetAllRelays(settings uint16) {
	newSettings := settings
	for relay := range rl.Relays {
		rl.Relays[relay].On = (newSettings & 1) != 0
		newSettings >>= 1
	}
	rl.lastReportedValue = settings
}

func (rl *RelaysType) GetAllRelays() uint16 {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.getAllRelays()
}

func (rl *RelaysType) getAllRelays() uint16 {
	var val uint16

	for _, relay := range rl.Relays {
		val >>= 1
		if relay.On {
			val += 0x8000
		}
	}
	return val
}

func (rl *RelaysType) GetRelay(relay uint8) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.Relays[relay].On
}

func (rl *RelaysType) GetRelayName(relay uint8) string {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	return rl.Relays[relay].Name
}

func (rl *RelaysType) SetRelayName(relay uint8, name string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.Relays[relay].Name = name
}

func (rl *RelaysType) SetRelay(relay uint8, on bool) {
	// If this is the dryer relay and we are trying to turn it off
	if !on && relay == uint8(currentSettings.DryerRelay) {
		for _, el := range Electrolysers.Arr {
			// If an electrolyser is powered on then do not turn off the dryer
			if el.IsSwitchedOn() && el.powerRelay != relay {
				// We are trying to turn off the dryer relay but there is an electrolyser
				// that is still powered on with a different relay, so ignore the request.

				return
			}
		}
	}
	// If this is the water management relay, and we are trying to turn it off and we want it on when any electrolyser is on
	if !on && relay == uint8(currentSettings.WaterDumpRelay) && (currentSettings.WaterDumpAction == ELRun) {
		// If the water dump is set to the recirculating pump and the conductivity is high do not turn off the relay
		if currentSettings.WaterDumpAction == ELRun {
			_, conductivity := AnalogInputs.GetInput(7)
			if conductivity > float32(currentSettings.MaximumConductivity) {
				return
			}
		}
		for _, el := range Electrolysers.Arr {
			// If an electrolyser is powered on then do not turn off the water dump relay
			if el.IsSwitchedOn() {
				// We are trying to turn off the dryer relay but there is an electrolyser
				// that is still powered on with a different relay, so ignore the request.

				return
			}
		}
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Get the current relay settings
	relays := rl.getAllRelays()
	// Set or reset the supplied relay
	if on {
		relays |= uint16(1) << relay
	} else {
		relays &= ^(uint16(1) << relay)
	}
	// Set the hardware
	if relays != rl.lastReportedValue {
		if err := canBus.SetRelays(relays); err != nil {
			log.Print(err)
		} else {
			mask := uint16(1)
			for RL := 0; RL < 16; RL++ {
				if (relays & mask) != (rl.lastReportedValue & mask) {
					var newVal string
					if (relays & mask) != 0 {
						newVal = "ON"
					} else {
						newVal = "OFF"
					}
					log.Printf("Relay %s turned %s", rl.Relays[RL].Name, newVal)
				}
				mask = mask << 1
			}
		}
	}
	// Update the local copy
	rl.SetAllRelays(relays)
}

func (rl *RelaysType) SetRelayByName(relay string, on bool) error {
	relay = strings.ToLower(relay)
	for idx, r := range rl.Relays {
		if strings.ToLower(r.Name) == relay {
			rl.SetRelay(uint8(idx), on)
			return nil
		}
	}
	return fmt.Errorf("invalid relay name - %s", relay)
}

/*
UpdateRelays retransmits the current relay settings as a heartbeat signal to the Firefly IO board
*/
func (rl *RelaysType) UpdateRelays() {
	relays := rl.GetAllRelays()
	if err := canBus.SetRelays(relays); err != nil {
		log.Print(err)
	}
}
