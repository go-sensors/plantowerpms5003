package plantowerpms5003

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	coreio "github.com/go-sensors/core/io"
	"github.com/go-sensors/core/pm"
	"github.com/go-sensors/core/units"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=../sensorio/serial.go -destination=./serial_mock_test.go -package=plantowerpms5003_test

// Sensor represents a configured Plantower PMS5003 particulate matter sensor
type Sensor struct {
	concentrations   chan *pm.Concentration
	portFactory      coreio.PortFactory
	reconnectTimeout time.Duration
	errorHandlerFunc ShouldTerminate
}

// Option is a configured option that may be applied to a Sensor
type Option struct {
	apply func(*Sensor)
}

type ChecksumError struct {
	Bytes            []byte
	ReportedChecksum uint16
	ActualChecksum   uint16
}

func (ce *ChecksumError) Error() string {
	return fmt.Sprintf("checksum of measurement %v was reported as %v, but calculated as %v",
		ce.Bytes,
		ce.ReportedChecksum,
		ce.ActualChecksum)
}

// NewSensor creates a Sensor with optional configuration
func NewSensor(portFactory coreio.PortFactory, options ...*Option) *Sensor {
	concentrations := make(chan *pm.Concentration)
	s := &Sensor{
		concentrations:   concentrations,
		portFactory:      portFactory,
		reconnectTimeout: DefaultReconnectTimeout,
		errorHandlerFunc: nil,
	}
	for _, o := range options {
		o.apply(s)
	}
	return s
}

// WithReconnectTimeout specifies the duration to wait before reconnecting after a recoverable error
func WithReconnectTimeout(timeout time.Duration) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.reconnectTimeout = timeout
		},
	}
}

// ReconnectTimeout is the duration to wait before reconnecting after a recoverable error
func (s *Sensor) ReconnectTimeout() time.Duration {
	return s.reconnectTimeout
}

// ShouldTerminate is a function that returns a result indicating whether the Sensor should terminate after a recoverable error
type ShouldTerminate func(error) bool

// WithRecoverableErrorHandler registers a function that will be called when a recoverable error occurs
func WithRecoverableErrorHandler(f ShouldTerminate) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.errorHandlerFunc = f
		},
	}
}

// RecoverableErrorHandler a function that will be called when a recoverable error occurs
func (s *Sensor) RecoverableErrorHandler() ShouldTerminate {
	return s.errorHandlerFunc
}

var (
	// Header of a valid measurement
	ReadingHeader = []byte{0x42, 0x4D}
)

const (
	PM01_0UpperBoundSize = 1 * units.Micrometer
	PM02_5UpperBoundSize = 2_500 * units.Nanometer
	PM10_0UpperBoundSize = 10 * units.Micrometer
)

type measurement struct {
	// Frame length
	Length uint16
	// PM1.0 concentration unit μ g/m3 (CF=1，standard particle)
	Pm10Std uint16
	// PM2.5 concentration unit μ g/m3 (CF=1，standard particle)
	Pm25Std uint16
	// PM10 concentration unit μ g/m3 (CF=1，standard particle)
	Pm100Std uint16
	// PM1.0 concentration unit μ g/m3 (under atmospheric environment)
	Pm10Env uint16
	// PM2.5 concentration unit μ g/m3 (under atmospheric environment)
	Pm25Env uint16
	// PM10 concentration unit μ g/m3 (under atmospheric environment)
	Pm100Env uint16
	// Number of particles with diameter beyond 0.3 um in 0.1L of air.
	Particles3um uint16
	// Number of particles with diameter beyond 0.5 um in 0.1L of air.
	Particles5um uint16
	// Number of particles with diameter beyond 1.0 um in 0.1L of air.
	Particles10um uint16
	// Number of particles with diameter beyond 2.5 um in 0.1L of air.
	Particles25um uint16
	// Number of particles with diameter beyond 5.0 um in 0.1L of air.
	Particles50um uint16
	// Number of particles with diameter beyond 10.0 um in 0.1L of air.
	Particles100um uint16
	// Reserved
	Unused uint16
	// Check code
	Checksum uint16
}

// Run begins reading from the sensor and blocks until either an error occurs or the context is completed
func (s *Sensor) Run(ctx context.Context) error {
	defer close(s.concentrations)
	for {
		port, err := s.portFactory.Open()
		if err != nil {
			return errors.Wrap(err, "failed to open port")
		}

		group, innerCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			<-innerCtx.Done()
			return port.Close()
		})
		group.Go(func() error {
			reader := bufio.NewReader(port)
			for {
				nextHeaderIndex := 0
				for {
					b, err := reader.ReadByte()
					if err != nil {
						if err == io.EOF {
							return err
						}
						return errors.Wrap(err, "failed to read while seeking measurement header")
					}

					if b == ReadingHeader[nextHeaderIndex] {
						nextHeaderIndex++
						if nextHeaderIndex == len(ReadingHeader) {
							break
						}
						continue
					}

					nextHeaderIndex = 0
				}

				measurement := &measurement{}
				err = binary.Read(reader, binary.BigEndian, measurement)
				if err != nil {
					return errors.Wrap(err, "failed to read measurement")
				}

				var bb bytes.Buffer
				bb.Write(ReadingHeader)
				binary.Write(&bb, binary.BigEndian, measurement)
				expectedChecksum := uint16(0)
				for _, b := range bb.Bytes()[:bb.Len()-2] {
					expectedChecksum += uint16(b)
				}

				if measurement.Checksum != expectedChecksum {
					if s.errorHandlerFunc != nil {
						err := &ChecksumError{
							Bytes:            bb.Bytes(),
							ReportedChecksum: measurement.Checksum,
							ActualChecksum:   expectedChecksum,
						}
						if s.errorHandlerFunc(err) {
							return err
						}
					}
					continue
				}

				concentrations := []*pm.Concentration{
					{
						UpperBoundSize: PM01_0UpperBoundSize,
						Amount:         units.MassConcentration(measurement.Pm10Env) * units.MicrogramPerCubicMeter,
					},
					{
						UpperBoundSize: PM02_5UpperBoundSize,
						Amount:         units.MassConcentration(measurement.Pm25Env) * units.MicrogramPerCubicMeter,
					},
					{
						UpperBoundSize: PM10_0UpperBoundSize,
						Amount:         units.MassConcentration(measurement.Pm100Env) * units.MicrogramPerCubicMeter,
					},
				}

				for _, concentration := range concentrations {
					select {
					case <-innerCtx.Done():
						return nil
					case s.concentrations <- concentration:
					}
				}
			}
		})

		err = group.Wait()
		if s.errorHandlerFunc != nil {
			if s.errorHandlerFunc(err) {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.reconnectTimeout):
		}
	}
}

// Concentrations returns a channel of PM concentration readings as they become available from the sensor
func (s *Sensor) Concentrations() <-chan *pm.Concentration {
	return s.concentrations
}

// ConcentrationSpecs returns a collection of specified measurement ranges supported by the sensor
func (*Sensor) ConcentrationSpecs() []*pm.ConcentrationSpec {
	return []*pm.ConcentrationSpec{
		{
			UpperBoundSize:   PM01_0UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 1000 * units.MicrogramPerCubicMeter,
		},
		{
			UpperBoundSize:   PM02_5UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 1000 * units.MicrogramPerCubicMeter,
		},
		{
			UpperBoundSize:   PM10_0UpperBoundSize,
			Resolution:       1 * units.MicrogramPerCubicMeter,
			MinConcentration: 0 * units.MicrogramPerCubicMeter,
			MaxConcentration: 1000 * units.MicrogramPerCubicMeter,
		},
	}
}
