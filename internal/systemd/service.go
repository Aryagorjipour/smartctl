package systemd

import (
	"fmt"
	"os/exec"
	"strings"
)

type Service struct {
	Name        string
	Description string
	Status      string
	Enabled     bool
}

func ListServices() ([]Service, error) {
	fmt.Println("Processing ListServices")

	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("error on fetching list of services: %w", err)
	}

	services := []Service{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := fields[0]
		status := fields[3]
		description := strings.Join(fields[4:], " ")

		enabled, _ := isEnabled(name)

		services = append(services, Service{
			Name:        name,
			Description: description,
			Status:      status,
			Enabled:     enabled,
		})
	}

	println("founded services", len(services))

	return services, nil
}

func isEnabled(name string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", name)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(string(output)) == "enabled", nil
}

func StartService(name string) error {
	cmd := exec.Command("systemctl", "start", name)
	return cmd.Run()
}

func StopService(name string) error {
	cmd := exec.Command("systemctl", "stop", name)
	return cmd.Run()
}

func EnableService(name string) error {
	cmd := exec.Command("systemctl", "enable", name)
	return cmd.Run()
}

func DisableService(name string) error {
	cmd := exec.Command("systemctl", "disable", name)
	return cmd.Run()
}

func RestartService(name string) error {
	cmd := exec.Command("systemctl", "restart", name)
	return cmd.Run()
}
