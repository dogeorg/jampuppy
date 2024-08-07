# JamPuppy - a Jam Stack tool for Dogebox 

This project is hosted on [radicle.xyz](https://radicle.xyz) at [rad:zFn9j7QAVA1juvUkjNxKDHRREn35](https://app.radicle.xyz/nodes/ash.radicle.garden/zFn9j7QAVA1juvUkjNxKDHRREn35)

JamPuppy is a simple go binary that can be used to create and
serve static jam-stack apps on dogebox, as installable pups.

Example:

```sh
jampuppy -p 8080 -v -r "/dogenet http://localhost:8080/dogenet /txindex http://localhost:8081/txindex" -d /mysite


<requests in/out printed here>
```

Installation:

```sh
go install
```

This installs the binary to ~/go/bin which you may need to add to your path.

Alternatively you can `go build` and run the binary as `./jampuppy`

Usage:

```sh
jampuppy <args> -d <directory of static files>
```

Help:

```sh
jampuppy --help
```

Arguments:

```sh
-A, --app-index string    Index file to serve in place of 404 (for SPA)
-d, --dir string          Directory of static files to serve (default ".")
-h, --host string         Bind to network interface (default "localhost")
-I, --index string        Index file to serve for directores (default "index.html")
-p, --port int            Listen port (default 8080)
-r, --proxy stringArray   Reverse Proxy: '/thing http://localhost:8085/thing' (one or more)
-v, --verbose             Colour prints ingoing/outgoing requests for debugging
```
