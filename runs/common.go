// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// This package is used to read and write runs. It currently only supports the
// most recent version of the binary format. In later updates, it may support
// other formats and/or versions.
package runs

import (
	"strconv"
	"strings"
	"time"
)

const CurrentVersion = 0x00

const (
	GameHL2 = 0x00
)

const DefaultGame = GameHL2

var PrettyGameNames = map[byte]string{
	GameHL2: "Half-Life 2",
}

var ShortGameNames = map[byte]string{
	GameHL2: "half-life2",
}

// This function determines the game being used based on the given name (short
// or pretty). It will return 0, false if it was unable to determine the game.
func DetermineGame(from string) (byte, bool) {
	if gameID, err := strconv.ParseUint(from, 10, 64); err == nil {
		return byte(gameID), true
	}

	for gameID, shortName := range ShortGameNames {
		if strings.EqualFold(from, shortName) {
			return gameID, true
		}
	}

	for gameID, longName := range PrettyGameNames {
		if strings.EqualFold(from, longName) {
			return gameID, true
		}
	}

	return 0, false
}

func RecreateHeader(header []byte) *RunHeader {
	if header != nil && len(header) >= 8 {
		return &RunHeader{
			Game: header[0],

			GhostColorR: header[1],
			GhostColorG: header[2],
			GhostColorB: header[3],

			TrailColorR: header[4],
			TrailColorG: header[5],
			TrailColorB: header[6],
			TrailLength: header[7],
		}
	}

	return nil
}

// RunHeader represents the header of a run file. It is universal to all run
// file formats.
type RunHeader struct {
	Game byte `json:"-"`

	GhostColorR byte `json:"ghost_color_r"`
	GhostColorG byte `json:"ghost_color_g"`
	GhostColorB byte `json:"ghost_color_b"`

	TrailColorR byte `json:"trail_color_r"`
	TrailColorG byte `json:"trail_color_g"`
	TrailColorB byte `json:"trail_color_b"`
	TrailLength byte `json:"trail_length" endpoints_desc:"The length of the trail (in seconds)"`
}

// This creates a "raw" version of the run header for use in the datastore. It
// can be converted back using
func (h *RunHeader) MakeRaw() []byte {
	return []byte{
		h.Game,

		h.GhostColorR,
		h.GhostColorG,
		h.GhostColorB,

		h.TrailColorR,
		h.TrailColorG,
		h.TrailColorB,
		h.TrailLength,
	}
}

// This returns the duration of the runner's trail.
func (h *RunHeader) TrailDuration() time.Duration {
	return time.Duration(h.TrailLength) * time.Second
}

type RunLine struct {
	MapName    string
	PlayerName string

	Time float32
	X    float32
	Y    float32
	Z    float32
}
