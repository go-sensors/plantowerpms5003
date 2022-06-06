package plantowerpms5003

import (
	"time"

	"github.com/go-sensors/core/serial"
)

const (
	DefaultReconnectTimeout = 5 * time.Second
)

// GetDefaultSerialPortConfig gets the manufacturer-specified defaults for connecting to the sensor
func GetDefaultSerialPortConfig() *serial.SerialPortConfig {
	return &serial.SerialPortConfig{
		Baud:        9600,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second,
	}
}
