This app is set up to test that a PHP app can be served with FPM and Nginx
together using the configuration set up by this buildpack.

- The `.nginx.conf.d/user-server.conf` file is present to show that a user can
provide their own configuration, which will be included in the final Nginx config file

- The `htdocs` directory is the main web directory containing PHP code that'll be served

- The Procfile is used to mimic what a start-command buildpack would do down
  the line (run FPM and Nginx). It runs the `php-fpm` portion of the `web`
  process with the expectation that the Paketo FPM buildpack has run and set
  this path. This is needed because **this buildpack sets up Nginx
  configuration that is specific to running with FPM**. Without the FPM
  command, the resulting build container will not be able to be reached.

  If this buildpack creates non-FPM specific configuration in the future, the
  FPM part of this app and the associated test can be removed.

- The `plan.toml` file is used to request that `nginx-config` (this buildpack's
  provision) is required, as well as `nginx` and `php-fpm`.
