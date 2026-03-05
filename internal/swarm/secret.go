package swarm

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

func CreateSecret(client *sshpkg.Client, name, value string) error {
	// Check if exists
	_, err := client.Run(fmt.Sprintf("docker secret inspect %s", name))
	if err == nil {
		return nil // already exists
	}

	_, err = client.Run(fmt.Sprintf("echo -n '%s' | docker secret create %s -", value, name))
	if err != nil {
		return fmt.Errorf("creating secret %s: %w", name, err)
	}
	return nil
}

func RemoveSecret(client *sshpkg.Client, name string) error {
	_, err := client.Run(fmt.Sprintf("docker secret rm %s", name))
	return err
}

func ListSecrets(client *sshpkg.Client) (string, error) {
	return client.Run("docker secret ls --format 'table {{.ID}}\t{{.Name}}\t{{.CreatedAt}}'")
}

func UpdateSecret(client *sshpkg.Client, name, value string) error {
	// Docker secrets are immutable — remove and recreate
	RemoveSecret(client, name)
	return CreateSecret(client, name, value)
}

func GeneratePassword(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

func SecretExists(client *sshpkg.Client, name string) bool {
	_, err := client.Run(fmt.Sprintf("docker secret inspect %s", name))
	return err == nil
}

func EnsureSecrets(client *sshpkg.Client, secrets map[string]string) error {
	for name, value := range secrets {
		if value == "" {
			value = GeneratePassword(32)
			secrets[name] = value
		}
		if err := CreateSecret(client, name, value); err != nil {
			return err
		}
	}
	return nil
}

func GetSecretNames(client *sshpkg.Client) ([]string, error) {
	out, err := client.Run("docker secret ls --format '{{.Name}}'")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSpace(out), "\n"), nil
}
