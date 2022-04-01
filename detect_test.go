package phpnginx_test

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/packit/v2"
	phpnginx "github.com/paketo-buildpacks/php-nginx"
	"github.com/sclevine/spec"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		detect     packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		detect = phpnginx.Detect()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("$BP_PHP_SERVER is set to nginx", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_PHP_SERVER", "nginx")).To(Succeed())
		})
		it.After(func() {
			Expect(os.Unsetenv("BP_PHP_SERVER")).To(Succeed())
		})

		it("requires nothing and provides a nginx config", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Requires: []packit.BuildPlanRequirement{},
				Provides: []packit.BuildPlanProvision{
					{
						Name: phpnginx.PhpNginxConfig,
					},
				},
			}))
		})
	})

	context("$BP_PHP_SERVER is not set to nginx", func() {
		it("detection fails", func() {
			_, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).To(MatchError(packit.Fail))
		})
	})
}
