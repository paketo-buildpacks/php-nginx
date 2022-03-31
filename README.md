# PHP Nginx Cloud Native Buildpack
A Cloud Native Buildpack for configuring Nginx settings for PHP apps.

The buildpack generates the Nginx configuration file with the minimal set of
options to get Nginx to work with FPM (FastCGI Process Manager), and
incorporates configuration from users and environment variables. The final
Nginx configuration file is available at
`/workspace/nginx.conf`, or locatable
through the buildpack-set `$PHP_NGINX_PATH` environment variable at
launch-time.

## Integration

The PHP Nginx CNB provides `php-nginx`, which can be required by subsequent
buildpacks. In order to configure Nginx, the user must declare the intention to
use Nginx as the web-server by setting the `$BP_PHP_SERVER` environment
variable to `nginx` at build-time.

```shell
pack build my-nginx-app --env BP_PHP_SERVER="nginx"
```

## Nginx Configuration Sources 
The base configuration file generated in this buildpack includes some default configuration, FPM-specific configuration, and
has `include` sections for user-included configuration.

#### FPM-specific Configuration
This buildpack is written to provide Nginx configuration that should always be
used in conjunction with FPM. The Nginx configuration file is generated to
include FPM-specific configuration. This buildpack also sets up an FPM
configuration file with Nginx-specific socket settings and makes it available
in the `/workspace`.

#### User Included Configuration
User-included configuration should be found in the application source directory
under `<app-directory>/.nginx.conf.d/`. Server-specific configuration should be
inside a file named `*-server.conf`, and HTTP configuration should be inside a
file with the naming structure `*-http.conf`.

If files at these paths exist, it
will be included in `include` sections at the appropriate places in the generated
Nginx configuration.

#### Environment Variables
The following environment variables can be used to override default settings in
the Nginx configuration file.

| Variable | Default |
| -------- | -------- |
| `BP_PHP_ENABLE_HTTPS_REDIRECT`   | true    |
| `BP_PHP_WEB_DIR`    | htdocs    |

## Usage

To package this buildpack for consumption:

```
$ ./scripts/package.sh
```

This builds the buildpack's Go source using `GOOS=linux` by default. You can
supply another value as the first argument to `package.sh`.

## Run Tests

To run all unit tests, run:
```
./scripts/unit.sh
```

To run all integration tests, run:
```
./scripts/integration.sh
```

## Debug Logs
For extra debug logs from the image build process, set the `$BP_LOG_LEVEL`
environment variable to `DEBUG` at build-time (ex. `pack build my-app --env
BP_LOG_LEVEL=DEBUG` or through a  [`project.toml`
file](https://github.com/buildpacks/spec/blob/main/extensions/project-descriptor.md).
