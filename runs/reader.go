// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package runs

import (
	"encoding/binary"
	"io"
)

type RunReader interface {
	VerifyPreamble() (bool, error)
	ReadHeader() (*RunHeader, error)
	ReadLine() (*RunLine, error)
}

func NewReader(reader io.Reader) RunReader {
	return &runReader{reader}
}

type runReader struct {
	io.Reader
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
// It will return false if the first byte of the file is not 0xAF or the second
// byte is not a known verison of the run file format.
func (r *runReader) VerifyPreamble() (bool, error) {
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

func (r *runReader) ReadHeader() (header *RunHeader, err error) {
	header = new(RunHeader)
	err = binary.Read(r, binary.LittleEndian, header)
	return
}

func (r *runReader) ReadLine() (*RunLine, error) {
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
