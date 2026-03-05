package deploy

import (
	"fmt"
	"strings"

	sshpkg "github.com/ensarkurrt/swarmforge/internal/ssh"
	"github.com/ensarkurrt/swarmforge/internal/ui"
)

type HealthCheck struct {
	Name    string
	Command string
}

var HealthChecks = []HealthCheck{
	{"PostgreSQL", "docker exec $(docker ps -q -f name=data-postgresql_postgresql) pg_isready -U admin 2>/dev/null || echo 'FAIL'"},
	{"Redis", "docker exec $(docker ps -q -f name=data-redis_redis) redis-cli ping 2>/dev/null || echo 'FAIL'"},
	{"MinIO", "curl -sf http://localhost:9000/minio/health/live 2>/dev/null && echo 'OK' || echo 'FAIL'"},
	{"Loki", "curl -sf http://localhost:3100/ready 2>/dev/null && echo 'OK' || echo 'FAIL'"},
	{"Grafana", "curl -sf http://localhost:3000/api/health 2>/dev/null && echo 'OK' || echo 'FAIL'"},
	{"Traefik", "curl -sf http://localhost:8080/ping 2>/dev/null && echo 'OK' || echo 'FAIL'"},
}

func RunHealthChecks(clients map[string]*sshpkg.Client) {
	ui.Header("Health Checks")

	for _, check := range HealthChecks {
		passed := false
		for _, client := range clients {
			out, err := client.Run(check.Command)
			if err == nil && !strings.Contains(out, "FAIL") {
				passed = true
				break
			}
		}
		ui.StatusLine(check.Name, statusText(passed), passed)
	}
}

func CheckServiceReplicas(client *sshpkg.Client) {
	ui.Header("Service Replicas")

	out, err := client.Run("docker service ls --format '{{.Name}}\t{{.Replicas}}\t{{.Image}}'")
	if err != nil {
		ui.Error("Failed to list services: %s", err)
		return
	}

	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		name := parts[0]
		replicas := parts[1]
		healthy := !strings.Contains(replicas, "0/")
		ui.StatusLine(name, replicas, healthy)
	}
}

func WaitForService(client *sshpkg.Client, serviceName string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		out, _ := client.Run(fmt.Sprintf("docker service ls --filter name=%s --format '{{.Replicas}}'", serviceName))
		out = strings.TrimSpace(out)
		if out != "" && !strings.HasPrefix(out, "0/") {
			return nil
		}
		_, _ = client.Run("sleep 5")
	}
	return fmt.Errorf("service %s did not become healthy", serviceName)
}

func statusText(ok bool) string {
	if ok {
		return "healthy"
	}
	return "unhealthy"
}
