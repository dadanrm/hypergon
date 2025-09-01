package plugin

import "github.com/dadanrm/hypergon/routing"

type Plugin interface {
	// Build the entire plugin routing mechanism
	Build() []*routing.RouteGroup
	// Plug in to the app
	Plug()
}
