package phpnginx

import (
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

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
func Build(nginxConfigWriter ConfigWriter, nginxFpmConfigWriter ConfigWriter, logger scribe.Emitter) packit.BuildFunc {
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

		planner := draft.NewPlanner()
		launch, build := planner.MergeLayerTypes(PhpNginxConfig, context.Plan.Entries)
		phpNginxLayer.Launch = launch
		phpNginxLayer.Build = build

		// test this
		phpNginxLayer.SharedEnv.Default("PHP_NGINX_PATH", nginxConfigPath)
		logger.EnvironmentVariables(phpNginxLayer)

		return packit.BuildResult{
			Layers: []packit.Layer{phpNginxLayer},
		}, nil
	}
}
