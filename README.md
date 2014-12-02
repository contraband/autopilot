# autopilot

*cf plugin for hands-off, zero downtime, blue-green application deploys*

![Autopilot](http://i.imgur.com/xj2vbwk.jpg)

## installation

```
$ go get github.com/concourse/autopilot
$ cf install-plugin $GOPATH/bin/autopilot
```

## usage

```
$ cf zero-downtime-push application-to-replace -m path/to/new_manifest.yml -p path/to/new/path
```

