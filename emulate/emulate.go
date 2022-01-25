package emulate

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultGRPCHost = "localhost:9010"
	DefaultRestHost = "localhost:9020"

	DefaultContainerName = "spanner-emulator"
	DefaultImage         = "gcr.io/cloud-spanner-emulator/emulator"
)

// Running checks to see if there is something accepting
// connections at the host provided.
func Running(host string) bool {
	conn, err := net.DialTimeout("tcp", host, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Port splits host string (host:port) and returns the port
func Port(host string) (string, error) {
	sp := strings.Split(host, ":")
	if len(sp) < 2 {
		return "", fmt.Errorf("unable to parse port from: %s", host)
	}
	return sp[1], nil
}

// DefaultEmulator is the spanner emulator running in docker.
var DefaultEmulator = Emulator{
	Runner:   DefaultDocker.Run,
	Closer:   DefaultDocker.Close,
	GRPCHost: DefaultGRPCHost,
	RestHost: DefaultRestHost,
}

// Emulator will run and close the spanner emulator
type Emulator struct {
	Runner             func(grpcHost, restHost string) error
	Closer             func() error
	GRPCHost, RestHost string
}

func (e *Emulator) Hosts() (grpc, rest string) {
	return e.GRPCHost, e.RestHost
}

func (e *Emulator) Run() error {
	return e.Runner(e.Hosts())
}

func (e *Emulator) Close() error {
	return e.Closer()
}

var DefaultDocker = Docker{Name: DefaultContainerName, Image: DefaultImage}

type Docker struct {
	Name  string
	Image string
}

func (d *Docker) Run(grpcHost, restHost string) error {
	grpcPort, err := Port(grpcHost)
	if err != nil {
		return err
	}
	restPort, err := Port(restHost)
	if err != nil {
		return err
	}
	params := []string{
		"run", "-d", "--name", d.Name,
		"-p", fmt.Sprintf("%s:%s", grpcPort, grpcPort),
		"-p", fmt.Sprintf("%s:%s", restPort, restPort), d.Image,
	}
	start := exec.Command("docker", params...)
	return start.Run()
}

func (d *Docker) Close() error {
	killParams := []string{
		"kill", d.Name,
	}
	rmParams := []string{
		"rm", d.Name,
	}
	kill := exec.Command("docker", killParams...)
	rm := exec.Command("docker", rmParams...)
	// ignore errors: if container doesn't exists, it will return errors.
	kill.Run()
	rm.Run()
	return nil
}
