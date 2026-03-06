package swarm

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
)

// shellEscape escapes a string for safe use in single-quoted shell arguments.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func CreateSecret(client *sshpkg.Client, name, value string) error {
	// Check if exists
	_, err := client.Run(fmt.Sprintf("docker secret inspect '%s'", shellEscape(name)))
	if err == nil {
		return nil // already exists
	}

	// Write secret value to a temp file to avoid shell injection via echo.
	// The value is transferred over the SSH channel (SCP), not interpolated into a shell command.
	tmpPath := fmt.Sprintf("/tmp/secret-%s", name)
	if writeErr := client.WriteContent(tmpPath, value); writeErr != nil {
		return fmt.Errorf("writing secret temp file: %w", writeErr)
	}

	_, err = client.Run(fmt.Sprintf("docker secret create '%s' '%s' && rm -f '%s'",
		shellEscape(name), shellEscape(tmpPath), shellEscape(tmpPath)))
	if err != nil {
		// Clean up temp file on failure
		_, _ = client.Run(fmt.Sprintf("rm -f '%s'", shellEscape(tmpPath)))
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
	_ = RemoveSecret(client, name)
	return CreateSecret(client, name, value)
}

func GeneratePassword(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:length]
}

// GenerateBase64Key generates a standard base64-encoded random key.
// Use this when the consumer expects valid base64 (e.g. NetBird encryptionKey).
func GenerateBase64Key(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
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
