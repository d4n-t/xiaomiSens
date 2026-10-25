package main

import (
	"encoding/binary"
	"net"
)

// measurement preserves the firmware's raw units. Temperature and humidity
// are hundredths; downstream queries perform their display scaling.
type measurement struct {
	MAC               string
	Temperature       int16
	Humidity          uint16
	BatteryMillivolts uint16
	BatteryPercentage uint8
	Sequence          uint8
	Flags             uint8
}

// decodePayload expects at least 15 bytes from the current sensor firmware.
// Payload validation and support for future formats remain deferred.
// Layout: reversed MAC (6), little-endian temperature (2), humidity (2),
// battery mV (2), battery percentage (1), sequence (1), flags (1).
func decodePayload(data []byte) measurement {
	mac := make(net.HardwareAddr, 6)
	for i := range mac {
		mac[i] = data[5-i]
	}
	return measurement{
		MAC:               mac.String(),
		Temperature:       int16(binary.LittleEndian.Uint16(data[6:8])),
		Humidity:          binary.LittleEndian.Uint16(data[8:10]),
		BatteryMillivolts: binary.LittleEndian.Uint16(data[10:12]),
		BatteryPercentage: data[12],
		Sequence:          data[13],
		Flags:             data[14],
	}
}
