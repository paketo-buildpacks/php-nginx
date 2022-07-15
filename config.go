package phpnginx

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:embed assets/nginx.conf
var NGINXConfTemplate string

//go:embed assets/nginx-fpm.conf
var NGINXFPMConfTemplate string

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

func (c NginxConfigWriter) Write(workingDir string) (string, error) {
	tmpl, err := template.New("nginx.conf").Parse(NGINXConfTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Nginx config template: %w", err)
	}

	data := NginxConfig{
		AppRoot: workingDir,
	}

	var configFiles []string
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
		configFiles = append(configFiles, userServerMatches...)
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
		configFiles = append(configFiles, userHttpMatches...)
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

	fpmSocket := "/tmp/php-fpm.socket"
	data.FpmSocket = fpmSocket
	c.logger.Debug.Subprocess(fmt.Sprintf("FPM socket: %s", fpmSocket))

	var b bytes.Buffer
	err = tmpl.Execute(&b, data)
	if err != nil {
		// not tested
		return "", err
	}

	path := filepath.Join(workingDir, "nginx.conf")
	err = os.WriteFile(path, b.Bytes(), 0600)
	if err != nil {
		return "", err
	}

	for _, file := range append(configFiles, path) {
		info, err := os.Stat(file)
		if err != nil {
			return "", err
		}

		err = os.Chmod(file, info.Mode()|0060)
		if err != nil {
			return "", err
		}
	}

	return path, nil
}

func (c NginxFpmConfigWriter) Write(workingDir string) (string, error) {
	tmpl, err := template.New("nginx-fpm.conf").Parse(NGINXFPMConfTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse Nginx-Fpm config template: %w", err)
	}

	// Configuration set by this buildpack

	// If there's a user-provided Nginx conf, include it in the base configuration.
	fpmSocket := "/tmp/php-fpm.socket"
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

	path := filepath.Join(workingDir, ".php.fpm.bp", "nginx-fpm.conf")
	err = os.WriteFile(path, b.Bytes(), 0600)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	err = os.Chmod(path, info.Mode()|0040)
	if err != nil {
		return "", err
	}

	return path, nil
}
