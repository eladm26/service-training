// Package mux provides support to bind domain level routes
// to the application mux.
package mux

import (
	"net/http"
	"os"

	"github.com/ardanlabs/service/api/services/sales/route/sys/checkapi"
	"github.com/ardanlabs/service/foundation/web"
)

func WebAPI(shutdown chan os.Signal) *web.App {
	mux := web.NewApp(shutdown)

	checkapi.Routes(mux)

	return mux
}
