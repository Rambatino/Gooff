package global

import (
	"net/http"

	"github.com/Rambatino/gooff"
)

func init() {
	http.DefaultTransport = gooff.GoOffline("", true)
}
