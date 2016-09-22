package core

import (
	"bytes"
	"fmt"

	"github.com/goburrow/gol"
)

const (
	serverLoggerName = "melon/server"
)

// Server is a managed HTTP server handling incoming connections to both application and admin.
// A server can have multiple connectors (listeners on different ports) sharing
// one ServerHandler.
type Server interface {
	Managed
}

// ServerHandler allows users to register a http.Handler.
type ServerHandler interface {
	// Handle registers the handler for the given pattern.
	// An implementation of ServerHandler must at least support http.Handler.
	Handle(method, pattern string, handler interface{})
	// PathPrefix returns prefix path of this handler.
	PathPrefix() string
}

// ServerFactory builds Server with given configuration and environment.
type ServerFactory interface {
	Build(environment *Environment) (Server, error)
}

// ServerEnvironment contains handlers for server and resources.
type ServerEnvironment struct {
	// ServerHandler belongs to the Server created by ServerFactory.
	// The default implementation is DefaultServerHandler.
	ServerHandler ServerHandler

	components       []interface{}
	resourceHandlers []ResourceHandler

	visibleEndpoints []string
}

// NewServerEnvironment creates a new ServerEnvironment.
func NewServerEnvironment() *ServerEnvironment {
	return &ServerEnvironment{}
}

// Register registers component to the environment. These components will be
// handled by all handlers added by AddResourceHandler.
func (env *ServerEnvironment) Register(component ...interface{}) {
	env.components = append(env.components, component...)
}

// AddResourceHandler adds the resource handler into this environment.
// This method is not concurrent-safe.
func (env *ServerEnvironment) AddResourceHandler(handler ...ResourceHandler) {
	env.resourceHandlers = append(env.resourceHandlers, handler...)
}

// LogEndpoint records all endpoints to display on application start.
// FIXME: recording endpoints automatically in ServerHandler or ResourceHandler?
func (env *ServerEnvironment) LogEndpoint(method, path string, component interface{}) {
	endpoint := fmt.Sprintf("%-7s %s%s (%T)",
		method, env.ServerHandler.PathPrefix(), path, component)
	env.visibleEndpoints = append(env.visibleEndpoints, endpoint)
}

func (env *ServerEnvironment) onStarting() {
	for _, component := range env.components {
		env.handle(component)
	}
	env.logResources()
	env.logEndpoints()
}

func (env *ServerEnvironment) onStopped() {
}

func (env *ServerEnvironment) handle(component interface{}) {
	// Last handler first
	for i := len(env.resourceHandlers) - 1; i >= 0; i-- {
		env.resourceHandlers[i].HandleResource(component)
	}
}

func (env *ServerEnvironment) logResources() {
	logger := gol.GetLogger(serverLoggerName)
	if !logger.DebugEnabled() {
		return
	}
	var buf bytes.Buffer
	for i, component := range env.components {
		if i > 0 {
			fmt.Fprintf(&buf, ",")
		}
		fmt.Fprintf(&buf, "%T", component)
	}
	logger.Debugf("resources = [%v]", buf.String())
}

func (env *ServerEnvironment) logEndpoints() {
	logger := gol.GetLogger(serverLoggerName)
	if !logger.InfoEnabled() {
		return
	}
	var buf bytes.Buffer
	for _, e := range env.visibleEndpoints {
		fmt.Fprintf(&buf, "    %s\n", e)
	}
	logger.Infof("endpoints =\n\n%s", buf.String())
}
