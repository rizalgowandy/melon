package main

import (
	"fmt"
	"os"

	"github.com/goburrow/gomelon"
	"github.com/goburrow/gomelon/core"
	"github.com/goburrow/gomelon/rest"
	"golang.org/x/net/context"
)

type User struct {
	Name string
}

// REST resource.
type UserResource struct {
}

func (r *UserResource) Path() string {
	return "/user/:name"
}

func (r *UserResource) GET(c context.Context) (interface{}, error) {
	params, _ := rest.PathParamsFromContext(c)
	return &User{Name: params["name"]}, nil
}

func (r *UserResource) POST(c context.Context) (interface{}, error) {
	user := &User{}
	if err := rest.RequestBodyFromContext(c, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserResource) DELETE(c context.Context) (interface{}, error) {
	params, _ := rest.PathParamsFromContext(c)
	return fmt.Sprintf("Deleted: user %v", params["name"]), nil
}

// Main application.
type application struct {
	gomelon.Application
}

// Initialize adds RESTful bundle.
func (app *application) Initialize(bootstrap *core.Bootstrap) {
	app.Application.Initialize(bootstrap)
	bootstrap.AddBundle(&rest.Bundle{})
}

// Run registers application resources.
func (app *application) Run(conf interface{}, env *core.Environment) error {
	if err := app.Application.Run(conf, env); err != nil {
		return err
	}
	env.Server.Register(&UserResource{})
	env.Server.Register(&rest.XMLProvider{})
	return nil
}

func main() {
	app := &application{}
	app.SetName("rest")

	err := gomelon.Run(app, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%#v\n", err)
		os.Exit(1)
	}
}