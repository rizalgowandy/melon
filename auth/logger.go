package auth

import "github.com/goburrow/gol"

var logger gol.Logger

func init() {
	logger = gol.GetLogger("melon/auth")
}
