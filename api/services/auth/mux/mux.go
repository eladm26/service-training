// Package mux provides support to bind domain level routes
// to the application mux.
package mux

import (
	"os"

	"github.com/ardanlabs/service/api/services/api/mid"
	"github.com/ardanlabs/service/api/services/auth/route/authapi"
	"github.com/ardanlabs/service/api/services/auth/route/checkapi"
	"github.com/ardanlabs/service/business/api/auth"
	"github.com/ardanlabs/service/foundation/logger"
	"github.com/ardanlabs/service/foundation/web"
	"github.com/jmoiron/sqlx"
)

func WebAPI(build string, log *logger.Logger, db *sqlx.DB, auth *auth.Auth, shutdown chan os.Signal) *web.App {
	mux := web.NewApp(shutdown, mid.Logger(log), mid.Errors(log), mid.Metrics(), mid.Panics())

	checkapi.Routes(build, mux, log, db)
	authapi.Routes(mux, auth)

	return mux
}
