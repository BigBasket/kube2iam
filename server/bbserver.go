package server

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"net/http/pprof"
)

func startServer(server *Server) {
	echoInstance := echo.New()

	echoInstance.Any("/debug/pprof/trace", func(context echo.Context) error {
		pprof.Trace(context.Response().Writer, context.Request())

		return nil
	})

	echoInstance.Any("/debug/pprof/", func(context echo.Context) error {
		pprof.Index(context.Response().Writer, context.Request())

		return nil
	})

	echoInstance.Any("/:version/meta-data/iam/security-credentials", server.getInstanceRole)
	echoInstance.Any("/:version/meta-data/iam/security-credentials/", server.getInstanceRole)

	echoInstance.Any("/:version/meta-data/iam/security-credentials/:role", server.getPodRole)

	echoInstance.Any("/:path", server.allAWSOtherRoutes)

	logrus.Infof("Listening on port %s", server.AppPort)
	if err := echoInstance.Start(":" + server.AppPort); err != nil {
		logrus.Fatalf("Error creating kube2iam http server: %+v", err)
	}
}
