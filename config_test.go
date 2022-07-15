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

		workingDir           string
		nginxConfigWriter    phpnginx.NginxConfigWriter
		nginxFpmConfigWriter phpnginx.NginxFpmConfigWriter
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "workingDir")
		Expect(err).NotTo(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(workingDir, ".php.fpm.bp"), os.ModePerm)).To(Succeed())

		logEmitter := scribe.NewEmitter(bytes.NewBuffer(nil))
		nginxConfigWriter = phpnginx.NewNginxConfigWriter(logEmitter)
		nginxFpmConfigWriter = phpnginx.NewFpmNginxConfigWriter(logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("nginx Config writer", func() {
		it("writes an nginx.conf file into the working dir", func() {
			path, err := nginxConfigWriter.Write(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(path).To(Equal(filepath.Join(workingDir, "nginx.conf")))
			Expect(path).To(BeARegularFile())

			info, err := os.Stat(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().String()).To(Equal("-rw-rw----"))

			contents, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("%s/htdocs;", workingDir)))
			Expect(string(contents)).To(ContainSubstring(`listen       {{env "PORT"}}  default_server;`))
			Expect(string(contents)).To(ContainSubstring("map $http_x_forwarded_proto $redirect_to_https"))
			Expect(string(contents)).To(ContainSubstring("server unix:/tmp/php-fpm.socket;"))
			Expect(string(contents)).NotTo(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-server.conf", workingDir)))
			Expect(string(contents)).NotTo(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-http.conf", workingDir)))
		})

		context("there are user-provided conf files", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(workingDir, ".nginx.conf.d"), os.ModePerm)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, ".nginx.conf.d", "some-server.conf"), nil, 0600)).To(Succeed())
				Expect(os.WriteFile(filepath.Join(workingDir, ".nginx.conf.d", "some-http.conf"), nil, 0600)).To(Succeed())
			})

			it.After(func() {
				Expect(os.RemoveAll(filepath.Join(workingDir, ".nginx.conf.d"))).To(Succeed())
			})

			it("writes an nginx.conf with the user included configurations into workingDir", func() {
				path, err := nginxConfigWriter.Write(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(workingDir, "nginx.conf")))
				Expect(filepath.Join(workingDir, "nginx.conf")).To(BeARegularFile())

				contents, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-server.conf", workingDir)))
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("include %s/.nginx.conf.d/*-http.conf", workingDir)))
			})

			it("sets the permissions on those files to be group-writable", func() {
				_, err := nginxConfigWriter.Write(workingDir)
				Expect(err).NotTo(HaveOccurred())

				info, err := os.Stat(filepath.Join(workingDir, ".nginx.conf.d", "some-server.conf"))
				Expect(err).NotTo(HaveOccurred())
				Expect(info.Mode().String()).To(Equal("-rw-rw----"))

				info, err = os.Stat(filepath.Join(workingDir, ".nginx.conf.d", "some-http.conf"))
				Expect(err).NotTo(HaveOccurred())
				Expect(info.Mode().String()).To(Equal("-rw-rw----"))
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
				path, err := nginxConfigWriter.Write(workingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(path).To(Equal(filepath.Join(workingDir, "nginx.conf")))

				contents, err := os.ReadFile(filepath.Join(workingDir, "nginx.conf"))
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("%s/some-web-dir;", workingDir)))
				Expect(string(contents)).NotTo(ContainSubstring("map $http_x_forwarded_proto $redirect_to_https"))
			})
		})

		context("failure cases", func() {
			context("when the BP_PHP_ENABLE_HTTPS_REDIRECT value cannot be parsed into a bool", func() {
				it.Before(func() {
					Expect(os.Setenv("BP_PHP_ENABLE_HTTPS_REDIRECT", "blah")).To(Succeed())
				})

				it.After(func() {
					Expect(os.Unsetenv("BP_PHP_ENABLE_HTTPS_REDIRECT")).To(Succeed())
				})

				it("returns an error", func() {
					_, err := nginxConfigWriter.Write(workingDir)
					Expect(err).To(MatchError(ContainSubstring("failed to parse $BP_PHP_ENABLE_HTTPS_REDIRECT into boolean:")))
				})
			})

			context("when conf file can't be opened for writing", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, "nginx.conf"), nil, 0400)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := nginxConfigWriter.Write(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})

	context("nginx-fpm config writer", func() {
		it("writes an nginx-fpm.conf file into the working dir", func() {
			path, err := nginxFpmConfigWriter.Write(workingDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(path).To(Equal(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf")))
			Expect(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf")).To(BeARegularFile())

			info, err := os.Stat(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().String()).To(Equal("-rw-r-----"))

			contents, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring("listen = /tmp/php-fpm.socket"))
		})

		context("failure cases", func() {
			context("when conf file can't be opened for writing", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf"), nil, 0400)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := nginxFpmConfigWriter.Write(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
