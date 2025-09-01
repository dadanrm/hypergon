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
	for _, group := range groups {
		routerGroup := m.app.Group(group.Prefix)

		if len(group.Middleware) > 0 {
			routerGroup.Chain(group.Middleware...)
		}

		for _, route := range group.Routes {
			fullPath := fmt.Sprintf("%s %s", route.Method, route.Path)
			routerGroup.Action(fullPath, route.Handler)
		}

		if len(group.SubGroups) > 0 {
			m.registerSubGroups(group.Prefix, group.Middleware, group.SubGroups)
		}
	}
}

func (m *routingmanagerimpl) registerSubGroups(parentPrefix string, parentMiddleware []hypergon.Middleware, subGroups []*RouteGroup) {
	for _, subGroup := range subGroups {
		fullPrefix := path.Join(parentPrefix, subGroup.Prefix)

		allMiddleware := slices.Clone(parentMiddleware)
		allMiddleware = append(allMiddleware, subGroup.Middleware...)

		routerGroup := m.app.Group(fullPrefix)

		if len(allMiddleware) > 0 {
			routerGroup.Chain(allMiddleware...)
		}

		for _, route := range subGroup.Routes {
			fullPath := fmt.Sprintf("%s %s", route.Method, route.Path)
			routerGroup.Action(fullPath, route.Handler)
		}

		if len(subGroup.SubGroups) > 0 {
			m.registerSubGroups(fullPrefix, allMiddleware, subGroup.SubGroups)
		}
	}
}

// This will handle the defining of handler functions and its method and path.
type Controller interface {
	Build() []*RouteGroup
}
