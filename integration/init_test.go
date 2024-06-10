package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/onsi/gomega/format"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/occam/packagers"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	buildpack          string
	buildPlanBuildpack string
	nginxBuildpack     string
	phpBuildpack       string
	phpFpmBuildpack    string
	procfileBuildpack  string

	offlineBuildpack       string
	offlineNginxBuildpack  string
	offlinePhpBuildpack    string
	offlinePhpFpmBuildpack string

	root string

	buildpackInfo struct {
		Buildpack struct {
			ID   string
			Name string
		}
	}
)

func TestIntegration(t *testing.T) {
	Expect := NewWithT(t).Expect

	format.MaxLength = 0

	var config struct {
		BuildPlan string `json:"build-plan"`
		Nginx     string `json:"nginx"`
		Php       string `json:"php"`
		PhpFpm    string `json:"php-fpm"`
		Procfile  string `json:"procfile"`
	}

	file, err := os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&config)).To(Succeed())

	file, err = os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())

	_, err = toml.NewDecoder(file).Decode(&buildpackInfo)
	Expect(err).NotTo(HaveOccurred())

	root, err = filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()
	libpakBuildpackStore := occam.NewBuildpackStore().WithPackager(packagers.NewLibpak())

	buildpack, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	buildPlanBuildpack, err = buildpackStore.Get.
		Execute(config.BuildPlan)
	Expect(err).NotTo(HaveOccurred())

	nginxBuildpack, err = buildpackStore.Get.
		Execute(config.Nginx)
	Expect(err).NotTo(HaveOccurred())

	phpBuildpack, err = buildpackStore.Get.
		Execute(config.Php)
	Expect(err).NotTo(HaveOccurred())

	phpFpmBuildpack, err = buildpackStore.Get.
		Execute(config.PhpFpm)
	Expect(err).NotTo(HaveOccurred())

	offlineBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	offlineNginxBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(config.Nginx)
	Expect(err).NotTo(HaveOccurred())

	offlinePhpBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(config.Php)
	Expect(err).NotTo(HaveOccurred())

	offlinePhpFpmBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		Execute(config.PhpFpm)
	Expect(err).NotTo(HaveOccurred())

	procfileBuildpack, err = libpakBuildpackStore.Get.
		Execute(config.Procfile)
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(10 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Default", testDefault)
	suite("Offline", testOffline)
	suite.Run(t)
}
