package phpnginx

import (
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go

// EntryResolver defines the interface for picking the most relevant entry from
// the Buildpack Plan entries.
type EntryResolver interface {
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface ConfigWriter --output fakes/config_writer.go

// ConfigWriter sets up the default Nginx configuration file and incorporates
// any user configurations.
type ConfigWriter interface {
	Write(layerPath, workingDir, cnbPath string) (string, error)
}

// Build will return a packit.BuildFunc that will be invoked during the build
// phase of the buildpack lifecycle.
//
// Build will create a layer dedicated to Nginx configuration, configure default Nginx
// settings, incorporate other configuration sources, and make the
// configuration available at both build-time and
// launch-time.
func Build(entryResolver EntryResolver, nginxConfigWriter ConfigWriter, nginxFpmConfigWriter ConfigWriter, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		logger.Debug.Process("Getting the layer associated with the Nginx configuration")
		phpNginxLayer, err := context.Layers.Get(PhpNginxConfigLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Debug.Subprocess(phpNginxLayer.Path)
		logger.Debug.Break()

		phpNginxLayer, err = phpNginxLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Process("Setting up the Nginx configuration file")
		nginxConfigPath, err := nginxConfigWriter.Write(phpNginxLayer.Path, context.WorkingDir, context.CNBPath)
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Break()

		logger.Process("Setting up the Nginx-specific FPM configuration file")
		_, err = nginxFpmConfigWriter.Write(phpNginxLayer.Path, context.WorkingDir, context.CNBPath)
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Break()

		launch, build := entryResolver.MergeLayerTypes(PhpNginxConfig, context.Plan.Entries)
		phpNginxLayer.Launch = launch
		phpNginxLayer.Build = build

		// test this
		phpNginxLayer.SharedEnv.Default("PHP_NGINX_PATH", nginxConfigPath)
		phpNginxLayer.Metadata = map[string]interface{}{
			"built_at": clock.Now().Format(time.RFC3339Nano),
		}
		logger.EnvironmentVariables(phpNginxLayer)

		return packit.BuildResult{
			Layers: []packit.Layer{phpNginxLayer},
		}, nil
	}
}
