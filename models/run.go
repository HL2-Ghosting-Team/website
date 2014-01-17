// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package models

import (
	"appengine"
	"appengine/datastore"
	"encoding/binary"
	"encoding/gob"
	"io"
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

type Run struct {
	ID      int64          `datastore:"-" json:"-" goon:"id"`
	User    *datastore.Key `datastore:"-" json:"uploader" goon:"parent"`
	Deleted bool           `datastore:",noindex" json:"-"`
	Ranked  bool           `json:"ranked"`

	UploadTime time.Time `json:"uploaded_at"`

	Game         int               `json:"game"` // TODO: We'd like to use a single byte here, but App Engine doesn't support single bytes as a datastore type.
	RunFile      appengine.BlobKey `datastore:",noindex" json:"-"`
	TotalTime    time.Duration     `json:"-"`
	FullAnalysis *datastore.Key    `datastore:",noindex" json:"-"`
}

func init() {
	gob.Register(MapAnalysis{})
}

type MapAnalysis struct {
	Name string
	Time time.Duration
}

type Analysis struct {
	ID  int64          `datastore:"-" goon:"id" json:"-"`
	Run *datastore.Key `datastore:"-" goon:"parent" json:"-"`

	RawHeader []byte     `json:"-"` // TODO: Unhackify.
	Header    *RunHeader `datastore:"-" json:"header"`

	Maps    []MapAnalysis `json:"maps"`
	Players []string      `json:"runners"`

	Fail       bool   `json:"failed"`
	FailReason string `json:"fail_reason"`
}

func (a *Analysis) MakeHeader() {
	if a.RawHeader != nil && len(a.RawHeader) >= 8 {
		a.Header = &RunHeader{
			Game: a.RawHeader[0],

			GhostColorR: a.RawHeader[1],
			GhostColorG: a.RawHeader[2],
			GhostColorB: a.RawHeader[3],

			TrailColorR: a.RawHeader[4],
			TrailColorG: a.RawHeader[5],
			TrailColorB: a.RawHeader[6],
			TrailLength: a.RawHeader[7],
		}
	} else {
		panic("Attempted to make a header while no suitable RawHeader existed.")
	}
}

type RunReader struct {
	io.Reader
}

type RunHeader struct {
	Game byte

	GhostColorR byte
	GhostColorG byte
	GhostColorB byte

	TrailColorR byte
	TrailColorG byte
	TrailColorB byte
	TrailLength byte
}

func (h *RunHeader) MakeRaw() []byte {
	return []byte{h.Game, h.GhostColorR, h.GhostColorG, h.GhostColorB, h.TrailColorR, h.TrailColorG, h.TrailColorB, h.TrailLength}
}

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

func readByte(r io.Reader) (byte, error) {
	arr := make([]byte, 1)
	if _, err := io.ReadFull(r, arr); err != nil {
		return 0, err
	}

	return arr[0], nil
}

// Verifies the beginning of the file.
// It always returns false if an error occurs.
// It will return false if the first byte of the file is not 0xAF or the second byte is not a known verison of the run file format.
func (r *RunReader) VerifyPreamble() (bool, error) {
	if firstByte, err := readByte(r); err != nil {
		return false, err
	} else if firstByte != 0xAF {
		return false, nil
	}

	if versionNumber, err := readByte(r); err != nil {
		return false, err
	} else if versionNumber != CurrentVersion {
		return false, nil
	}

	return true, nil
}

func (r *RunReader) ReadHeader() (header *RunHeader, err error) {
	header = new(RunHeader)
	err = binary.Read(r, binary.LittleEndian, header)
	return
}

func (r *RunReader) ReadLine() (*RunLine, error) {
	mapNameLength, err := readByte(r)
	if err != nil {
		if err == io.ErrUnexpectedEOF {
			err = io.EOF
		}
		return nil, err
	}
	mapNameArray := make([]byte, mapNameLength)
	if _, err := io.ReadFull(r, mapNameArray); err != nil {
		return nil, err
	}
	mapName := string(mapNameArray)

	playerNameLength, err := readByte(r)
	if err != nil {
		return nil, err
	}
	playerNameArray := make([]byte, playerNameLength)
	if _, err := io.ReadFull(r, playerNameArray); err != nil {
		return nil, err
	}
	playerName := string(playerNameArray)

	runLine := &RunLine{
		MapName:    mapName,
		PlayerName: playerName,
	}

	if err := binary.Read(r, binary.LittleEndian, &runLine.Time); err != nil {
		return nil, err
	}

	if err := binary.Read(r, binary.LittleEndian, &runLine.X); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &runLine.Y); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &runLine.Z); err != nil {
		return nil, err
	}

	return runLine, nil
}
