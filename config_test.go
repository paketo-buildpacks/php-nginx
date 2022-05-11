package phpnginx_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/v2/scribe"
	phpnginx "github.com/paketo-buildpacks/php-nginx"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testConfig(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layerDir             string
		workingDir           string
		cnbDir               string
		nginxConfigWriter    phpnginx.NginxConfigWriter
		nginxFpmConfigWriter phpnginx.NginxFpmConfigWriter
	)

	it.Before(func() {
		var err error
		layerDir, err = os.MkdirTemp("", "php-nginx-layer")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())

		workingDir, err = os.MkdirTemp("", "workingDir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(cnbDir, "config"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbDir, "config", "nginx.conf"), []byte(`
root               {{.AppRoot}}/{{.WebDirectory}};
listen       {{"{{"}}env "PORT"{{"}}"}}  default_server;
if !{{.DisableHTTPSRedirect }};
server unix:{{.FpmSocket}};

{{ if ne .UserServerConf "" }}
include {{.UserServerConf}};
{{- end}}

{{ if ne .UserHttpConf "" }}
include {{.UserHttpConf}};
{{- end}}
`), os.ModePerm)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(cnbDir, "config", "nginx-fpm.conf"), []byte(`
listen = {{.FpmSocket}};
`), os.ModePerm)).To(Succeed())

		Expect(os.MkdirAll(filepath.Join(workingDir, ".php.fpm.bp"), os.ModePerm)).To(Succeed())

		logEmitter := scribe.NewEmitter(bytes.NewBuffer(nil))
		nginxConfigWriter = phpnginx.NewNginxConfigWriter(logEmitter)
		nginxFpmConfigWriter = phpnginx.NewFpmNginxConfigWriter(logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(layerDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("nginx Config writer", func() {
		it("writes an nginx.conf file into the working dir", func() {
			path, err := nginxConfigWriter.Write(layerDir, workingDir, cnbDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(path).To(Equal(filepath.Join(layerDir, "nginx.conf")))
			Expect(filepath.Join(layerDir, "nginx.conf")).To(BeARegularFile())

			contents, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("%s/htdocs;", workingDir)))
			Expect(string(contents)).To(ContainSubstring(`listen       {{env "PORT"}}  default_server;`))
			Expect(string(contents)).To(ContainSubstring("if !false"))
			Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("server unix:%s/php-fpm.socket;", layerDir)))
			Expect(string(contents)).NotTo(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-server.conf", workingDir)))
			Expect(string(contents)).NotTo(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-http.conf", workingDir)))
		})

		context("there are user-provided conf files", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, ".nginx.conf.d"), os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, ".nginx.conf.d", "some-server.conf"), nil, os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, ".nginx.conf.d", "some-http.conf"), nil, os.ModePerm)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, ".nginx.conf.d"))).To(Succeed())
			})

			it("writes an nginx.conf with the user included configurations into workingDir", func() {
				path, err := nginxConfigWriter.Write(layerDir, workingDir, cnbDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(layerDir, "nginx.conf")))
				Expect(filepath.Join(layerDir, "nginx.conf")).To(BeARegularFile())

				contents, err := os.ReadFile(filepath.Join(layerDir, "nginx.conf"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-server.conf", workingDir)))
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-http.conf", workingDir)))
			})
		})

		context("all config env. vars are set", func() {
			it.Before(func() {
				Expect(os.Setenv("BP_PHP_WEB_DIR", "some-web-dir")).To(Succeed())
				Expect(os.Setenv("BP_PHP_ENABLE_HTTPS_REDIRECT", "false")).To(Succeed())
			})

			it.After(func() {
				Expect(os.Unsetenv("BP_PHP_ENABLE_HTTPS_REDIRECT")).To(Succeed())
				Expect(os.Unsetenv("BP_PHP_WEB_DIR")).To(Succeed())
			})

			it("writes an nginx.conf that includes the environment variable values", func() {
				path, err := nginxConfigWriter.Write(layerDir, workingDir, cnbDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(layerDir, "nginx.conf")))

				contents, err := os.ReadFile(filepath.Join(layerDir, "nginx.conf"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("%s/some-web-dir;", workingDir)))
				Expect(string(contents)).To(ContainSubstring("if !true"))
			})
		})

		context("failure cases", func() {
			context("when template is not parseable", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(cnbDir, "config", "nginx.conf"), []byte(`
{{ .UserInclude
		`), os.ModePerm)).To(Succeed())
				})
				it("returns an error", func() {
					_, err := nginxConfigWriter.Write(layerDir, workingDir, cnbDir)
					Expect(err).To(MatchError(ContainSubstring("unclosed action")))
				})
			})

			context("when the BP_PHP_ENABLE_HTTPS_REDIRECT value cannot be parsed into a bool", func() {
				it.Before(func() {
					Expect(os.Setenv("BP_PHP_ENABLE_HTTPS_REDIRECT", "blah")).To(Succeed())
				})
				it.After(func() {
					Expect(os.Unsetenv("BP_PHP_ENABLE_HTTPS_REDIRECT")).To(Succeed())
				})
				it("returns an error", func() {
					_, err := nginxConfigWriter.Write(layerDir, workingDir, cnbDir)
					Expect(err).To(MatchError(ContainSubstring("failed to parse $BP_PHP_ENABLE_HTTPS_REDIRECT into boolean:")))
				})
			})

			context("when conf file can't be opened for writing", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(layerDir, "nginx.conf"), nil, 0400)).To(Succeed())
				})
				it("returns an error", func() {
					_, err := nginxConfigWriter.Write(layerDir, workingDir, cnbDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})

	context("nginx-fpm config writer", func() {
		it("writes an nginx-fpm.conf file into the working dir", func() {
			path, err := nginxFpmConfigWriter.Write(layerDir, workingDir, cnbDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(path).To(Equal(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf")))
			Expect(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf")).To(BeARegularFile())

			contents, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("listen = %s/php-fpm.socket;", layerDir)))
		})

		context("failure cases", func() {
			context("when template is not parseable", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(cnbDir, "config", "nginx-fpm.conf"), []byte(`
	{{ .UserInclude
			`), os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := nginxFpmConfigWriter.Write(layerDir, workingDir, cnbDir)
					Expect(err).To(MatchError(ContainSubstring("unclosed action")))
				})
			})

			context("when conf file can't be opened for writing", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf"), nil, 0400)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := nginxFpmConfigWriter.Write(layerDir, workingDir, cnbDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
