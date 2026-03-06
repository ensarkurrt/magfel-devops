package swarm

import (
	"fmt"
	"strings"
	"time"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

type Tokens struct {
	Worker  string
	Manager string
}

func Init(client *sshpkg.Client, advertiseAddr string) (*Tokens, error) {
	// Check if already in swarm
	out, err := client.Run("docker info --format '{{.Swarm.LocalNodeState}}'")
	if err == nil && strings.TrimSpace(out) == "active" {
		return GetTokens(client)
	}

	_, err = client.Run(fmt.Sprintf("docker swarm init --advertise-addr %s", advertiseAddr))
	if err != nil {
		if strings.Contains(err.Error(), "already part of a swarm") {
			return GetTokens(client)
		}
		return nil, fmt.Errorf("swarm init: %w", err)
	}

	return GetTokens(client)
}

func GetTokens(client *sshpkg.Client) (*Tokens, error) {
	workerToken, err := client.Run("docker swarm join-token -q worker")
	if err != nil {
		return nil, fmt.Errorf("getting worker token: %w", err)
	}
	managerToken, err := client.Run("docker swarm join-token -q manager")
	if err != nil {
		return nil, fmt.Errorf("getting manager token: %w", err)
	}
	return &Tokens{
		Worker:  strings.TrimSpace(workerToken),
		Manager: strings.TrimSpace(managerToken),
	}, nil
}

func Join(client *sshpkg.Client, token, managerIP string) error {
	// Check if already in swarm
	out, _ := client.Run("docker info --format '{{.Swarm.LocalNodeState}}'")
	if strings.TrimSpace(out) == "active" {
		return nil
	}

	const maxAttempts = 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := client.Run(fmt.Sprintf("docker swarm join --token %s %s:2377", token, managerIP))
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "already part of a swarm") {
			return nil
		}
		// Docker returns exit 1 on timeout but continues in background.
		// Check if the node actually joined despite the error.
		time.Sleep(5 * time.Second)
		info, _ := client.Run("docker info --format '{{.Swarm.LocalNodeState}}'")
		if strings.TrimSpace(info) == "active" {
			return nil
		}
		if attempt < maxAttempts {
			time.Sleep(10 * time.Second)
		} else {
			return fmt.Errorf("swarm join: %w", err)
		}
	}
	return nil
}

func Leave(client *sshpkg.Client, force bool) error {
	cmd := "docker swarm leave"
	if force {
		cmd += " --force"
	}
	_, err := client.Run(cmd)
	return err
}
