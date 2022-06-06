# go-sensors/plantowerpms5003

Go library for interacting with the [Plantower PMS5003][plantowerpms5003] particulate matter sensor for measuring air quality.

## Quickstart

Take a look at [rpi-sensor-exporter][rpi-sensor-exporter] for an example implementation that makes use of this sensor (and others).

[rpi-sensor-exporter]: https://github.com/go-sensors/rpi-sensor-exporter

## Sensor Details

The [Plantower PMS5003][plantowerpms5003] particulate matter sensor is used for measuring air quality, per [vendor specifications][specs]. This [go-sensors] implementation makes use of the sensor's UART-based protocol for obtaining PM1.0, PM2.5, and PM10.0 measurements on an interval defined by the vendor.

[plantowerpms5003]: http://www.plantower.com/en/content/?108.html
[specs]: ./docs/plantower-pms5003-manual_v2-3.pdf
[go-sensors]: https://github.com/go-sensors

## Building

This software doesn't have any compiled assets.

## Code of Conduct

We are committed to fostering an open and welcoming environment. Please read our [code of conduct](CODE_OF_CONDUCT.md) before participating in or contributing to this project.

## Contributing

We welcome contributions and collaboration on this project. Please read our [contributor's guide](CONTRIBUTING.md) to understand how best to work with us.

## License and Authors

[![Daniel James logo](https://secure.gravatar.com/avatar/eaeac922b9f3cc9fd18cb9629b9e79f6.png?size=16) Daniel James](https://github.com/thzinc)

[![license](https://img.shields.io/github/license/go-sensors/plantowerpms5003.svg)](https://github.com/go-sensors/plantowerpms5003/blob/master/LICENSE)
[![GitHub contributors](https://img.shields.io/github/contributors/go-sensors/plantowerpms5003.svg)](https://github.com/go-sensors/plantowerpms5003/graphs/contributors)

This software is made available by Daniel James under the MIT license.
