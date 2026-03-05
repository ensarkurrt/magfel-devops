package swarm

import (
	"fmt"
	"strings"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

func SetNodeLabels(client *sshpkg.Client, nodeName string, labels map[string]string) error {
	for key, value := range labels {
		_, err := client.Run(fmt.Sprintf("docker node update --label-add %s=%s %s", key, value, nodeName))
		if err != nil {
			return fmt.Errorf("setting label %s=%s on %s: %w", key, value, nodeName, err)
		}
	}
	return nil
}

func PromoteNode(client *sshpkg.Client, nodeName string) error {
	_, err := client.Run(fmt.Sprintf("docker node promote %s", nodeName))
	return err
}

func DemoteNode(client *sshpkg.Client, nodeName string) error {
	_, err := client.Run(fmt.Sprintf("docker node demote %s", nodeName))
	return err
}

func ListNodes(client *sshpkg.Client) (string, error) {
	return client.Run("docker node ls --format 'table {{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}'")
}

func RemoveNode(client *sshpkg.Client, nodeName string) error {
	// Drain first
	_, _ = client.Run(fmt.Sprintf("docker node update --availability drain %s", nodeName))
	_, err := client.Run(fmt.Sprintf("docker node rm --force %s", nodeName))
	return err
}

func GetNodeInfo(client *sshpkg.Client, nodeName string) (string, error) {
	return client.Run(fmt.Sprintf("docker node inspect %s --pretty", nodeName))
}

func NodeExists(client *sshpkg.Client, nodeName string) bool {
	_, err := client.Run(fmt.Sprintf("docker node inspect %s", nodeName))
	return err == nil
}

func GetNodeRole(client *sshpkg.Client, nodeName string) string {
	out, err := client.Run(fmt.Sprintf("docker node inspect %s --format '{{.Spec.Role}}'", nodeName))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out)
}
