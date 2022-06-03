package phpnginx_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	phpnginx "github.com/paketo-buildpacks/php-nginx"
	"github.com/paketo-buildpacks/php-nginx/fakes"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir   string
		workingDir string
		cnbDir     string

		buffer               *bytes.Buffer
		nginxConfigWriter    *fakes.ConfigWriter
		nginxFpmConfigWriter *fakes.ConfigWriter

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layerDir, err = os.MkdirTemp("", "layer")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		nginxConfigWriter = &fakes.ConfigWriter{}
		nginxFpmConfigWriter = &fakes.ConfigWriter{}

		nginxConfigWriter.WriteCall.Returns.String = "some-workspace/nginx.conf"
		nginxFpmConfigWriter.WriteCall.Returns.String = "some-workspace/nginx-fpm.conf"

		build = phpnginx.Build(nginxConfigWriter, nginxFpmConfigWriter, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	it("writes an nginx config file and an nginx-fpm config file into its layer", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "some-entry"},
				},
			},
			Layers: packit.Layers{Path: layerDir},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(nginxConfigWriter.WriteCall.Receives.LayerPath).To(Equal(filepath.Join(layerDir, "php-nginx-config")))
		Expect(nginxConfigWriter.WriteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(nginxConfigWriter.WriteCall.Receives.CnbPath).To(Equal(cnbDir))

		Expect(nginxFpmConfigWriter.WriteCall.Receives.LayerPath).To(Equal(filepath.Join(layerDir, "php-nginx-config")))
		Expect(nginxFpmConfigWriter.WriteCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(nginxFpmConfigWriter.WriteCall.Receives.CnbPath).To(Equal(cnbDir))

		Expect(result.Layers).To(HaveLen(1))
		Expect(result.Layers[0].Name).To(Equal("php-nginx-config"))
		Expect(result.Layers[0].Path).To(Equal(filepath.Join(layerDir, "php-nginx-config")))
		Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{
			"PHP_NGINX_PATH.default": "some-workspace/nginx.conf",
		}))

		Expect(result.Layers[0].Build).To(BeFalse())
		Expect(result.Layers[0].Cache).To(BeFalse())
		Expect(result.Layers[0].Launch).To(BeFalse())
	})

	context("when nginx-config is required at launch time", func() {
		it("makes the layer available at launch time", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Stack:      "some-stack",
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "some-version",
				},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: phpnginx.PhpNginxConfig,
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layerDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			Expect(result.Layers[0].Name).To(Equal("php-nginx-config"))
			Expect(result.Layers[0].Path).To(Equal(filepath.Join(layerDir, "php-nginx-config")))
			Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{
				"PHP_NGINX_PATH.default": "some-workspace/nginx.conf",
			}))

			Expect(result.Layers[0].Launch).To(BeTrue())
			Expect(result.Layers[0].Build).To(BeFalse())
			Expect(result.Layers[0].Cache).To(BeFalse())
		})
	})

	context("failure cases", func() {
		context("when config layer cannot be gotten", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layerDir, fmt.Sprintf("%s.toml", phpnginx.PhpNginxConfigLayer)), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: phpnginx.PhpNginxConfigLayer},
						},
					},
					Layers: packit.Layers{Path: layerDir},
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("when nginx config file cannot be written", func() {
			it.Before(func() {
				nginxConfigWriter.WriteCall.Returns.Error = errors.New("nginx config writing error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: phpnginx.PhpNginxConfigLayer},
						},
					},
					Layers: packit.Layers{Path: layerDir},
				})
				Expect(err).To(MatchError(ContainSubstring("nginx config writing error")))
			})
		})

		context("when nginx-fpm config file cannot be written", func() {
			it.Before(func() {
				nginxFpmConfigWriter.WriteCall.Returns.Error = errors.New("nginx-fpm config writing error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					CNBPath:    cnbDir,
					Stack:      "some-stack",
					BuildpackInfo: packit.BuildpackInfo{
						Name:    "Some Buildpack",
						Version: "some-version",
					},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: phpnginx.PhpNginxConfigLayer},
						},
					},
					Layers: packit.Layers{Path: layerDir},
				})
				Expect(err).To(MatchError(ContainSubstring("nginx-fpm config writing error")))
			})
		})
	})
}
