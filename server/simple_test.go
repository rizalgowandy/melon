package server

import (
	"testing"

	"github.com/goburrow/melon/core"
)

var _ core.ServerFactory = (*SimpleFactory)(nil)

func TestSimpleFactory(t *testing.T) {
	env := core.NewEnvironment()
	factory := &SimpleFactory{}

	s, err := factory.BuildServer(env)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("server is nil")
	}
	if env.Server.Router == nil {
		t.Fatal("Server.ServerHandler is nil")
	}
	if env.Admin.Router == nil {
		t.Fatal("Admin.ServerHandler is nil")
	}
}
