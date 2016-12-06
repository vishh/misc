# Cpu-Sysrq

This program triggers a configurable sysrq whenever cpu availability is below a configurable threshold.

## Usage

`go run cpu-sysrq/main.go -sysrq=l -trigger-percent=10 -period=50ms -v=1 -logtostderr`

### Using Docker

`docker run --rm -it --privileged vish/cpu-sysrq --help`
