package deploy

// DeploymentOrder defines the order in which stacks should be deployed.
// Dependencies flow top-to-bottom: DNS first, then data, then infra, then monitoring, then tools.
var DeploymentOrder = []string{
	"infra-dns",
	"data-postgresql",
	"data-redis",
	"data-minio",
	"infra-traefik",
	"infra-portainer",
	"infra-registry",
	"infra-netbird",
	"log-loki",
	"log-promtail",
	"mon-prometheus",
	"mon-grafana",
	"mon-alertmanager",
	"tool-plane",
	"tool-openpanel",
	"tool-openstatus",
}

// StackDir maps stack name to its compose file directory
func StackDir(name string) string {
	return "/opt/stacks/" + name
}

// ComposePath returns the full path to the compose file on the remote node
func ComposePath(name string) string {
	return StackDir(name) + "/docker-compose.yml"
}
