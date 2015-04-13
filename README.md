# autopilot

*cf plugin for hands-off, zero downtime application deploys*

![Autopilot](http://i.imgur.com/xj2vbwk.jpg)

## installation

```
$ go get github.com/concourse/autopilot
$ cf install-plugin $GOPATH/bin/autopilot
```

## usage

```
$ cf zero-downtime-push APP [-b URL] [-c COMMAND] [-d DOMAIN] [-f MANIFEST] / 
[-i NUM_INSTANCES] [-m MEMORY] [-n HOST] [-p PATH] [-s STACK] / 
[--no-hostname] [--no-route] [--no-start]
```

Optional arguments include:

- -b — Custom buildpack URL, for example, https://github.com/heroku/heroku-buildpack-play.git or https://github.com/heroku/heroku-buildpack-play.git#stable to select stable branch
- -c — Start command for the application.
- -d — Domain, for example, example.com.
- -f — replaces --manifest
- -i — Number of instances of the application to run.
- -m — Memory limit, for example, 256, 1G, 1024M, and so on.
- -n — Hostname, for example, my-subdomain.
- -p — Path to application directory or archive.
- -s — Stack to use.
- -t — Timeout to start in seconds.
- --no-hostname — Map the root domain to this application (NEW).
- --no-manifest — Ignore manifests if they exist.
- --no-route — Do not map a route to this application (NEW).
- --no-start — Do not start the application after pushing.

Note: The `–no-route` option also removes existing routes from previous pushes of this app.


## method

*Autopilot* takes a different approach to other zero-downtime plugins. It
doesn't perform any [complex route re-mappings][indiana-jones] instead it leans
on the manifest feature of the Cloud Foundry CLI. The method also has the
advantage of treating a manifest as the source of truth and will converge the
state of the system towards that. This makes the plugin ideal for continuous
delivery environments.

1. The old application is renamed to `<APP-NAME>-venerable`. It keeps its old route
   mappings and this change is invisible to users.

2. The new application is pushed to `<APP-NAME>` (assuming that the name has
   not been changed in the manifest). It binds to the same routes as the old
   application (due to them being defined in the manifest) and traffic begins to
   be load-balanced between the two applications.

3. The old application is deleted along with its route mappings. All traffic
   now goes to the new application.

[indiana-jones]: https://www.youtube.com/watch?v=0gU35Tgtlmg

## tests

Tests are run by calling the script *test.sh* in the *scripts* directory. You may need to modify the GOROOT and GOPATH in the script to reflect the proper paths in your environment.
