package swarm

import (
	"fmt"
	"strings"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

func CreateOverlayNetwork(client *sshpkg.Client, name string) error {
	// Check if exists
	out, _ := client.Run(fmt.Sprintf("docker network ls --filter name=^%s$ --format '{{.Name}}'", name))
	if strings.TrimSpace(out) == name {
		return nil // already exists
	}

	_, err := client.Run(fmt.Sprintf(
		"docker network create --driver overlay --attachable --scope swarm %s", name))
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("creating overlay network %s: %w", name, err)
	}
	return nil
}

func ListNetworks(client *sshpkg.Client) (string, error) {
	return client.Run("docker network ls --format 'table {{.Name}}\t{{.Driver}}\t{{.Scope}}'")
}
