// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package models

import (
	"appengine"
	"appengine/datastore"
	"encoding/gob"
	"time"

	"github.com/HL2-Ghosting-Team/website/runs"
)

type Run struct {
	ID     int64          `datastore:"-" json:"-" goon:"id"`
	User   *datastore.Key `datastore:"-" json:"uploader" goon:"parent"`
	Ranked bool           `json:"ranked" endpoints:"desc=Whether or not the run has been submitted to the rankings"`

	UploadTime time.Time `json:"uploaded_at" endpoints:"desc=The time at which the run was uploaded"`

	Game         int               `json:"game" endpoints:"desc=The ID of the game that this run was made for"` // TODO: We'd like to use a single byte here, but App Engine doesn't support single bytes as a datastore type.
	RunFile      appengine.BlobKey `datastore:",noindex" json:"-"`
	TotalTime    time.Duration     `json:"-"`
	FullAnalysis *datastore.Key    `datastore:",noindex" json:"-"`
}

func init() {
	gob.Register(MapAnalysis{})
}

type MapAnalysis struct {
	Name string        `json:"name" endpoints:"desc=The name of the map"`
	Time time.Duration `json:"time" endpoints:"desc=The time (in nanoseconds) that the map took to complete"`
}

type Analysis struct {
	ID  int64          `datastore:"-" goon:"id" json:"-"`
	Run *datastore.Key `datastore:"-" goon:"parent" json:"-"`

	RawHeader []byte          `json:"-"`
	Header    *runs.RunHeader `datastore:"-" json:"header" endpoints:"desc=The header for the run file"`

	Maps    []MapAnalysis `json:"maps" endpoints:"desc=The analysis of each individual map"`
	Players []string      `json:"runners" endpoints:"desc=The names of all of the players involved in the run"`
}

func (a *Analysis) MakeHeader() {
	a.Header = runs.RecreateHeader(a.RawHeader)
}
