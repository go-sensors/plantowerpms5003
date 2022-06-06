package plantowerpms5003_test

import (
	"testing"
	"time"

	"github.com/go-sensors/core/serial"
	"github.com/go-sensors/plantowerpms5003"
	"github.com/stretchr/testify/assert"
)

func Test_GetDefaultSerialPortConfig_returns_expected_configuration(t *testing.T) {
	// Arrange
	expected := &serial.SerialPortConfig{
		Baud:        9600,
		Size:        8,
		Parity:      serial.ParityNone,
		StopBits:    serial.Stop1,
		ReadTimeout: 1 * time.Second,
	}

	// Act
	actual := plantowerpms5003.GetDefaultSerialPortConfig()

	// Assert
	assert.NotNil(t, actual)
	assert.EqualValues(t, expected, actual)
}
