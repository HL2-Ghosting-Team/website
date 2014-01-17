// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package endpoint

import (
	"github.com/nightexcessive/go-endpoints/endpoints"
)

func mustRegister(service interface{}, name, ver, desc string, isDefault bool) *endpoints.RpcService {
	api, err := endpoints.RegisterService(service, name, ver, desc, isDefault)
	if err != nil {
		panic(err)
	}

	return api
}

// The primary API service provided by the Ghosting website. It contains all of
// the necessary APIs.
type GhostingService struct {
}

var (
	ghostingService = mustRegister(new(GhostingService), "ghosting", "v0", "Ghosting API", true)
)

func init() {
	endpoints.HandleHttp()
}
