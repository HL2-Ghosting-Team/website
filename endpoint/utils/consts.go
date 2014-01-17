// Copyright 2009 Michael Johnson. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package utils

import (
	"github.com/nightexcessive/go-endpoints/endpoints"
)

// The scope(s) which all users are required to use. Scopes should be separated
// by spaces.
const UserScope = endpoints.EmailScope

// Client IDs that are authorized to use the API. These should not be modified.
var AuthorizedClientIDs = []string{endpoints.ApiExplorerClientId}
