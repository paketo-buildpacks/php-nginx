package phpnginx

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type NginxConfig struct {
	UserServerConf       string
	UserHttpConf         string
	DisableHTTPSRedirect bool
	AppRoot              string
	WebDirectory         string
	FpmSocket            string
}

type NginxFpmConfig struct {
	FpmSocket string
}

type NginxConfigWriter struct {
	logger scribe.Emitter
}

type NginxFpmConfigWriter struct {
	logger scribe.Emitter
}

func NewNginxConfigWriter(logger scribe.Emitter) NginxConfigWriter {
	return NginxConfigWriter{
		logger: logger,
	}
}

func NewFpmNginxConfigWriter(logger scribe.Emitter) NginxFpmConfigWriter {
	return NginxFpmConfigWriter{
		logger: logger,
	}
}

func (c NginxConfigWriter) Write(layerPath, workingDir, cnbPath string) (string, error) {
	tmpl, err := template.New("nginx.conf").ParseFiles(filepath.Join(cnbPath, "config", "nginx.conf"))
	if err != nil {
		return "", fmt.Errorf("failed to parse Nginx config template: %w", err)
	}

	data := NginxConfig{
		AppRoot: workingDir,
	}

	// Configuration set by this buildpack
	// If there's a user-provided Nginx conf, include it in the base configuration.
	userServerConf := filepath.Join(workingDir, ".nginx.conf.d", "*-server.conf")
	userServerMatches, err := filepath.Glob(userServerConf)
	if err != nil {
		// untested
		return "", fmt.Errorf("failed to glob %s: %w", userServerConf, err)
	}
	if len(userServerMatches) > 0 {
		data.UserServerConf = userServerConf
		c.logger.Debug.Subprocess(fmt.Sprintf("Including user-provided Nginx server configuration from: %s", userServerConf))
	}

	userHttpConf := filepath.Join(workingDir, ".nginx.conf.d", "*-http.conf")
	userHttpMatches, err := filepath.Glob(userHttpConf)
	if err != nil {
		// untested
		return "", fmt.Errorf("failed to glob %s: %w", userHttpConf, err)
	}
	if len(userHttpMatches) > 0 {
		data.UserHttpConf = userHttpConf
		c.logger.Debug.Subprocess(fmt.Sprintf("Including user-provided Nginx HTTP configuration from: %s", userHttpConf))
	}

	webDir := os.Getenv("BP_PHP_WEB_DIR")
	if webDir == "" {
		webDir = "htdocs"
	}
	data.WebDirectory = webDir
	c.logger.Debug.Subprocess(fmt.Sprintf("Web directory: %s", webDir))

	enableHTTPSRedirect := true
	enableHTTPSRedirectStr, ok := os.LookupEnv("BP_PHP_ENABLE_HTTPS_REDIRECT")
	if ok {
		enableHTTPSRedirect, err = strconv.ParseBool(enableHTTPSRedirectStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse $BP_PHP_ENABLE_HTTPS_REDIRECT into boolean: %w", err)
		}
	}
	data.DisableHTTPSRedirect = !enableHTTPSRedirect
	c.logger.Debug.Subprocess(fmt.Sprintf("Enable HTTPS redirect: %t", enableHTTPSRedirect))

	fpmSocket := filepath.Join(layerPath, "php-fpm.socket")
	data.FpmSocket = fpmSocket
	c.logger.Debug.Subprocess(fmt.Sprintf("FPM socket: %s", fpmSocket))

	var b bytes.Buffer
	err = tmpl.Execute(&b, data)
	if err != nil {
		// not tested
		return "", err
	}

	f, err := os.OpenFile(filepath.Join(layerPath, "nginx.conf"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, &b)
	if err != nil {
		// not tested
		return "", err
	}

	return f.Name(), nil
}

func (c NginxFpmConfigWriter) Write(layerPath, workingDir, cnbPath string) (string, error) {
	tmpl, err := template.New("nginx-fpm.conf").ParseFiles(filepath.Join(cnbPath, "config", "nginx-fpm.conf"))
	if err != nil {
		return "", fmt.Errorf("failed to parse Nginx-Fpm config template: %w", err)
	}

	// Configuration set by this buildpack

	// If there's a user-provided Nginx conf, include it in the base configuration.
	fpmSocket := filepath.Join(layerPath, "php-fpm.socket")
	c.logger.Debug.Subprocess(fmt.Sprintf("FPM socket: %s", fpmSocket))

	data := NginxFpmConfig{
		FpmSocket: fpmSocket,
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, data)
	if err != nil {
		// not tested
		return "", err
	}

	f, err := os.OpenFile(filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = io.Copy(f, &b)
	if err != nil {
		// not tested
		return "", err
	}

	return f.Name(), nil
}
