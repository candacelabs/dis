package forward

import (
	"fmt"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortBindings_MultiplePorts(t *testing.T) {
	portMap, portSet, err := PortBindings([]int{3000, 8080})
	require.NoError(t, err)

	assert.Len(t, portMap, 2)
	assert.Len(t, portSet, 2)

	for _, p := range []int{3000, 8080} {
		cp := nat.Port(fmt.Sprintf("%d/tcp", p))
		assert.Contains(t, portSet, cp)
		assert.Contains(t, portMap, cp)
		assert.Len(t, portMap[cp], 1)
		assert.Equal(t, fmt.Sprintf("%d", p), portMap[cp][0].HostPort)
	}
}

func TestPortBindings_SinglePort(t *testing.T) {
	portMap, portSet, err := PortBindings([]int{3000})
	require.NoError(t, err)

	assert.Len(t, portMap, 1)
	assert.Len(t, portSet, 1)

	cp := nat.Port("3000/tcp")
	assert.Contains(t, portSet, cp)
	assert.Contains(t, portMap, cp)
}

func TestPortBindings_EmptyPorts(t *testing.T) {
	portMap, portSet, err := PortBindings([]int{})
	require.NoError(t, err)

	assert.Empty(t, portMap)
	assert.Empty(t, portSet)
}

func TestPortBindings_PortMapFormat(t *testing.T) {
	portMap, _, err := PortBindings([]int{9090})
	require.NoError(t, err)

	cp := nat.Port("9090/tcp")
	require.Contains(t, portMap, cp)

	bindings := portMap[cp]
	require.Len(t, bindings, 1)
	assert.Equal(t, "0.0.0.0", bindings[0].HostIP)
	assert.Equal(t, "9090", bindings[0].HostPort)
}
