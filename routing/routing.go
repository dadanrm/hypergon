package routing

import (
	"fmt"
	"path"
	"slices"

	"github.com/dadanrm/hypergon"
)

type Route struct {
	Method  string
	Path    string
	Handler hypergon.HandlerFunc
}

type RouteGroup struct {
	Prefix     string
	Middleware []hypergon.Middleware
	Routes     []Route
	SubGroups  []*RouteGroup
}

// This manager handles the handlers and its endpoint by grouping and
// emitting the grouped handlers into a more compact approach.
type Manager interface {
	Register(groups []*RouteGroup)
}

func NewRoutingManager(app *hypergon.Hyper) Manager {
	return &routingmanagerimpl{app}
}

type routingmanagerimpl struct {
	// parent routing app
	app *hypergon.Hyper
}

// Build implements RoutingManager.
func (m *routingmanagerimpl) Register(groups []*RouteGroup) {
	m.registerGroups("", nil, groups)
}

func (m *routingmanagerimpl) registerGroups(parentPrefix string, parentMiddleware []hypergon.Middleware, groups []*RouteGroup) {
	for _, group := range groups {
		// 1. Calculate the full prefix for the current group.
		fullPrefix := path.Join(parentPrefix, group.Prefix)

		// 2. Collect all middleware from the parent and this group.
		allMiddleware := slices.Clone(parentMiddleware)
		allMiddleware = append(allMiddleware, group.Middleware...)

		// 3. Process all routes within this group.
		for _, route := range group.Routes {
			// Calculate the final, absolute path for the route.
			absolutePath := path.Join(fullPrefix, route.Path)
			handleString := fmt.Sprintf("%s %s", route.Method, absolutePath)

			fmt.Printf("[Routing Manager] Registering route: %s\n", handleString)

			// Wrap the route's handler with all collected middleware.
			// We loop backwards to apply them in the correct order (like an onion).
			finalHandler := route.Handler
			for i := len(allMiddleware) - 1; i >= 0; i-- {
				finalHandler = allMiddleware[i](finalHandler)
			}

			// Register the final handler with its absolute path DIRECTLY on the app.
			m.app.Action(handleString, finalHandler)
		}

		// 4. Continue the recursion for any subgroups.
		if len(group.SubGroups) > 0 {
			m.registerGroups(fullPrefix, allMiddleware, group.SubGroups)
		}
	}
}

// This will handle the defining of handler functions and its method and path.
type Controller interface {
	Build() []*RouteGroup
}
