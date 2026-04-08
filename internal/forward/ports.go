package forward

import (
	"fmt"

	"github.com/docker/go-connections/nat"
)

// PortBindings converts a list of port numbers into Docker SDK port configuration.
// Each port maps host:port → container:port/tcp (same number on both sides).
// Returns (portMap for HostConfig, portSet for ContainerConfig ExposedPorts, error).
func PortBindings(ports []int) (nat.PortMap, nat.PortSet, error) {
	portMap := nat.PortMap{}
	portSet := nat.PortSet{}

	for _, p := range ports {
		containerPort, err := nat.NewPort("tcp", fmt.Sprintf("%d", p))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid port %d: %w", p, err)
		}

		portSet[containerPort] = struct{}{}
		portMap[containerPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", p),
			},
		}
	}

	return portMap, portSet, nil
}
