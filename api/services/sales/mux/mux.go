// Package mux provides support to bind domain level routes
// to the application mux.
package mux

import (
	"os"

	"github.com/ardanlabs/service/api/services/api/mid"
	"github.com/ardanlabs/service/api/services/sales/route/sys/checkapi"
	"github.com/ardanlabs/service/app/api/authclient"
	"github.com/ardanlabs/service/foundation/logger"
	"github.com/ardanlabs/service/foundation/web"
	"github.com/jmoiron/sqlx"
)

func WebAPI(build string, log *logger.Logger, db *sqlx.DB, authClient *authclient.Client, shutdown chan os.Signal) *web.App {
	mux := web.NewApp(shutdown, mid.Logger(log), mid.Errors(log), mid.Metrics(), mid.Panics())

	checkapi.Routes(build, mux, log, db, authClient)

	return mux
}
