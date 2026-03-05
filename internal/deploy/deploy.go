package deploy

import (
	"fmt"
	"strings"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
)

func DeployStack(client *sshpkg.Client, name, composePath string) error {
	ui.Debug("Deploying stack: %s from %s", name, composePath)
	_, err := client.Run(fmt.Sprintf(
		"docker stack deploy -c %s %s --with-registry-auth",
		composePath, name))
	if err != nil {
		return fmt.Errorf("deploying stack %s: %w", name, err)
	}
	return nil
}

func RemoveStack(client *sshpkg.Client, name string) error {
	_, err := client.Run(fmt.Sprintf("docker stack rm %s", name))
	return err
}

func UpdateStack(client *sshpkg.Client, name, composePath string) error {
	// docker stack deploy is idempotent and performs rolling updates
	return DeployStack(client, name, composePath)
}

func ListStacks(client *sshpkg.Client) (string, error) {
	return client.Run("docker stack ls --format 'table {{.Name}}\t{{.Services}}'")
}

func GetStackServices(client *sshpkg.Client, name string) (string, error) {
	return client.Run(fmt.Sprintf(
		"docker stack services %s --format 'table {{.Name}}\t{{.Replicas}}\t{{.Image}}\t{{.Ports}}'",
		name))
}

func GetStackLogs(client *sshpkg.Client, name string, follow bool, tail string) (string, error) {
	services, err := client.Run(fmt.Sprintf("docker stack services %s --format '{{.Name}}'", name))
	if err != nil {
		return "", err
	}

	var allLogs []string
	for _, svc := range strings.Split(strings.TrimSpace(services), "\n") {
		if svc == "" {
			continue
		}
		cmd := fmt.Sprintf("docker service logs %s --tail %s --no-trunc 2>&1 | tail -50", svc, tail)
		logs, _ := client.Run(cmd)
		if logs != "" {
			allLogs = append(allLogs, fmt.Sprintf("=== %s ===\n%s", svc, logs))
		}
	}
	return strings.Join(allLogs, "\n\n"), nil
}

func GetServiceLogs(client *sshpkg.Client, service string, tail string) (string, error) {
	return client.Run(fmt.Sprintf("docker service logs %s --tail %s --no-trunc 2>&1", service, tail))
}

func StackExists(client *sshpkg.Client, name string) bool {
	out, err := client.Run("docker stack ls --format '{{.Name}}'")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == name {
			return true
		}
	}
	return false
}
