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

type Run struct {
	ID   int64          `datastore:"-" goon:"id"`
	User *datastore.Key `datastore:"-" goon:"parent"`

	UploadTime time.Time

	Game         string
	RunFile      appengine.BlobKey
	TotalTime    time.Duration
	FullAnalysis *datastore.Key
}

func init() {
	gob.Register(MapAnalysis{})
}

type MapAnalysis struct {
	Name string
	Time time.Duration
}

type Analysis struct {
	ID  int64          `datastore:"-" goon:"id"`
	Run *datastore.Key `datastore:"-" goon:"parent"`

	RawHeader []byte // TODO: Unhackify.

	Maps    []MapAnalysis
	Players []string

	Fail       bool
	FailReason string
}

func (a *Analysis) Header() *RunHeader {
	return &RunHeader{
		a.RawHeader[0],
		a.RawHeader[1],
		a.RawHeader[2],
		a.RawHeader[3],

		a.RawHeader[4],
		a.RawHeader[5],
		a.RawHeader[6],
		a.RawHeader[7],
	}
}

type RunReader struct {
	io.Reader
}

type RunHeader struct {
	GhostType   byte
	GhostColorR byte
	GhostColorG byte
	GhostColorB byte

	TrailColorR byte
	TrailColorG byte
	TrailColorB byte
	TrailLength byte
}

func (h *RunHeader) MakeRaw() []byte {
	return []byte{h.GhostType, h.GhostColorR, h.GhostColorG, h.GhostColorB, h.TrailColorR, h.TrailColorG, h.TrailColorB, h.TrailLength}
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

func (r *RunReader) VerifyPreamble() (bool, error) {
	firstByte, err := readByte(r)

	return firstByte == 0xAF, err
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
