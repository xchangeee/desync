package tools

import (
	"encoding/json"
	"list"
	"strings"

	"github.com/xchangeee/desync/schema/core@v0"
)

// Renders the provided sysdef struct in syslet json format, to be consumed by
// syslet via file or stdin.
#DesyncJsonFromDesec: {
	X1="in": core.#Desec
	_out: core.#Desync & {
		domains: [
			for k, v in desec.domains {
				core.#Domain & {name: k, rrsets: v}
			},
		]
		tokenPolicies: [
			for k, v in desec.tokenPolicies {
				core.#TokenPolicy & {tokenId: k, policies: v}
			},
		]
	}
	configRendered: json.Marshal(_out)
}
