//go:build !android && !ios && !freebsd && !js

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/netbirdio/netbird/client/ui/instancecolor"
)

// SBS-COL-02: the default installation's tray icons stay byte-identical.
func TestColourIconDefaultUnchanged(t *testing.T) {
	require.Nil(t, instanceColour, "the default installation has no colour")
	for _, icon := range [][]byte{iconConnected, iconConnectedDark, iconError, iconUpdateDisconnected} {
		assert.Equal(t, icon, colourIcon(icon, instancecolor.RingOnLight), "icon bytes")
	}
}

func TestColourIconNamed(t *testing.T) {
	c, err := instancecolor.Parse("skyblue")
	require.NoError(t, err)
	instanceColour = &c
	t.Cleanup(func() { instanceColour = nil })

	out := colourIcon(iconConnected, instancecolor.RingOnLight)
	assert.NotEqual(t, iconConnected, out, "dot drawn")
	again := colourIcon(iconConnected, instancecolor.RingOnLight)
	assert.Same(t, &out[0], &again[0], "composited once and cached")
	dark := colourIcon(iconConnected, instancecolor.RingOnDark)
	assert.NotEqual(t, out, dark, "the ring follows the taskbar theme")
}
