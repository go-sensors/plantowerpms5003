package plantowerpms5003_test

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/go-sensors/core/io/mocks"
	corepm "github.com/go-sensors/core/pm"
	"github.com/go-sensors/core/units"
	"github.com/go-sensors/plantowerpms5003"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func Test_NewSensor_returns_a_configured_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	// Act
	sensor := plantowerpms5003.NewSensor(portFactory)

	// Assert
	assert.NotNil(t, sensor)
	assert.Equal(t, plantowerpms5003.DefaultReconnectTimeout, sensor.ReconnectTimeout())
	assert.Nil(t, sensor.RecoverableErrorHandler())
}

func Test_NewSensor_with_options_returns_a_configured_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	expectedReconnectTimeout := plantowerpms5003.DefaultReconnectTimeout * 10

	// Act
	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithReconnectTimeout(expectedReconnectTimeout),
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	// Assert
	assert.NotNil(t, sensor)
	assert.Equal(t, expectedReconnectTimeout, sensor.ReconnectTimeout())
	assert.NotNil(t, sensor.RecoverableErrorHandler())
	assert.True(t, sensor.RecoverableErrorHandler()(nil))
}

func Test_ConcentrationSpecs_returns_supported_concentrations(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	sensor := plantowerpms5003.NewSensor(portFactory)
	expected := []*corepm.ConcentrationSpec{
		{
			UpperBoundSize:   plantowerpms5003.PM01_0UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 1000 * units.MicrogramPerCubicMeter,
		},
		{
			UpperBoundSize:   plantowerpms5003.PM02_5UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 1000 * units.MicrogramPerCubicMeter,
		},
		{
			UpperBoundSize:   plantowerpms5003.PM10_0UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 1000 * units.MicrogramPerCubicMeter,
		},
	}

	// Act
	actual := sensor.ConcentrationSpecs()

	// Assert
	assert.NotNil(t, actual)
	assert.EqualValues(t, expected, actual)
}

func Test_Run_fails_when_opening_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(nil, errors.New("boom"))
	sensor := plantowerpms5003.NewSensor(portFactory)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to open port")
}

func Test_Run_fails_reading_header_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Read(gomock.Any()).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to read while seeking measurement header")
}

func Test_Run_fails_reading_EOF_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Read(gomock.Any()).
		Return(0, io.EOF)
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorIs(t, err, io.EOF)
}

func Test_Run_fails_reading_measurement_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	returnsHeader := port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			noise := make([]byte, len(buf)-len(plantowerpms5003.ReadingHeader))
			rand.Read(noise)

			copy(buf, append(noise, plantowerpms5003.ReadingHeader...))
			return len(buf), nil
		})
	port.EXPECT().
		Read(gomock.Any()).
		Return(0, errors.New("boom")).
		After(returnsHeader)
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to read measurement")
}

func Test_Run_ignores_measurement_with_invalid_checksum(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			valid := append(
				plantowerpms5003.ReadingHeader,
				[]byte{
					0x00, 28, // Length
					0x01, 0xAA, // Pm10Std
					0x02, 0xBB, // Pm25Std
					0x03, 0xCC, // Pm100Std
					0x04, 0xDD, // Pm10Env
					0x05, 0xEE, // Pm25Env
					0x06, 0xFF, // Pm100Env
					0x00, 0x00, // Particles3um
					0x00, 0x00, // Particles5um
					0x00, 0x00, // Particles10um
					0x00, 0x00, // Particles25um
					0x00, 0x00, // Particles50um
					0x00, 0x00, // Particles100um
					0x12, 0x34, // Unused
					0x00, 0x00, // Checksum
				}...,
			)
			copy(buf, valid)
			return len(valid), nil
		}).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	recoverableErrorCount := 0
	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool {
			if recoverableErrorCount > 0 {
				return true
			}
			recoverableErrorCount++
			return false
		}))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	group.Go(func() error {
		select {
		case <-ctx.Done():
			return nil
		case <-sensor.Concentrations():
			cancel()
			return errors.New("received unexpected measurement")
		}
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "checksum")
}

func Test_Run_successfully_reads_measurement_from_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	expected := []*corepm.Concentration{
		{
			UpperBoundSize: plantowerpms5003.PM01_0UpperBoundSize,
			Amount:         0x04DD * units.MicrogramPerCubicMeter,
		},
		{
			UpperBoundSize: plantowerpms5003.PM02_5UpperBoundSize,
			Amount:         0x05EE * units.MicrogramPerCubicMeter,
		},
		{
			UpperBoundSize: plantowerpms5003.PM10_0UpperBoundSize,
			Amount:         0x06FF * units.MicrogramPerCubicMeter,
		},
	}
	port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			valid := append(
				plantowerpms5003.ReadingHeader,
				[]byte{
					0x00, 28, // Length
					0x01, 0xAA, // Pm10Std
					0x02, 0xBB, // Pm25Std
					0x03, 0xCC, // Pm100Std
					0x04, 0xDD, // Pm10Env
					0x05, 0xEE, // Pm25Env
					0x06, 0xFF, // Pm100Env
					0x00, 0x00, // Particles3um
					0x00, 0x00, // Particles5um
					0x00, 0x00, // Particles10um
					0x00, 0x00, // Particles25um
					0x00, 0x00, // Particles50um
					0x00, 0x00, // Particles100um
					0x12, 0x34, // Unused
					0x06, 0x01, // Checksum
				}...,
			)
			copy(buf, valid)
			return len(valid), nil
		}).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	actual := []*corepm.Concentration{}
	group.Go(func() error {
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				return errors.New("failed to receive expected measurement")
			case concentration := <-sensor.Concentrations():
				actual = append(actual, concentration)
			}
		}
		cancel()
		return nil
	})
	err := group.Wait()

	// Assert
	assert.Nil(t, err)
	assert.EqualValues(t, actual, expected)
}

func Test_Run_attempts_to_recover_from_failure(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Read(gomock.Any()).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := plantowerpms5003.NewSensor(portFactory,
		plantowerpms5003.WithRecoverableErrorHandler(func(err error) bool { return false }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.Nil(t, err)
}
