# autopilot

*cf plugin for hands-off, zero downtime application deploys*

![Autopilot](http://i.imgur.com/xj2vbwk.jpg)

## installation

**On *nix**
```
$ go get github.com/concourse/autopilot
$ cf install-plugin $GOPATH/bin/autopilot
```

**On Windows**
```
$ go get github.com/concourse/autopilot
$ cf install-plugin $env:GOPATH/bin/autopilot.exe
```

## usage

```
USAGE:
   Push a single app (with or without a manifest):
   cf zero-downtime-push APP [-b BUILDPACK_NAME] [-c COMMAND] [-d DOMAIN] [-f MANIFEST_PATH]
   [-i NUM_INSTANCES] [-k DISK] [-m MEMORY] [-n HOST] [-p PATH] [-s STACK] [-t TIMEOUT]
   [--no-hostname] [--no-manifest] [--no-route] [--no-start]

   Push multiple apps with a manifest:
   cf zero-downtime-push [-f MANIFEST_PATH]


OPTIONS:
   -b 			Custom buildpack by name (e.g. my-buildpack) or GIT URL (e.g. https://github.com/heroku/heroku-buildpack-play.git)
   -c 			Startup command, set to null to reset to default start command
   -d 			Domain (e.g. example.com)
   -f 			Path to manifest
   -i 			Number of instances
   -k 			Disk limit (e.g. 256M, 1024M, 1G)
   -m 			Memory limit (e.g. 256M, 1024M, 1G)
   -n 			Hostname (e.g. my-subdomain)
   -p 			Path to app directory or file
   -s 			Stack to use (a stack is a pre-built file system, including an operating system, that can run apps)
   -t 			Maximum time (in seconds) for CLI to wait for application start, other server side timeouts may apply
   --no-hostname	Map the root domain to this app
   --no-manifest	Ignore manifest file
   --no-route		Do not map a route to this app
   --no-start		Do not start an app after pushing
   --random-route	Create a random route for this app

```

## warning

Your application manifest **must** be up to date or the new application that
is created will not resemble the application that it is replacing.

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
