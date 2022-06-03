package main

import (
	"os"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	phpnginx "github.com/paketo-buildpacks/php-nginx"
)

func main() {
	logEmitter := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))
	nginxConfigWriter := phpnginx.NewNginxConfigWriter(logEmitter)
	nginxFpmConfigWriter := phpnginx.NewFpmNginxConfigWriter(logEmitter)
	entryResolver := draft.NewPlanner()

	packit.Run(
		phpnginx.Detect(),
		phpnginx.Build(entryResolver, nginxConfigWriter, nginxFpmConfigWriter, logEmitter),
	)
}
