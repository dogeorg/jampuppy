# JamPuppy - a Jam Stack tool for Dogebox 

JamPuppy is a simple go binary that can be used to create and
serve static jam-stack apps on dogebox, as installable pups.

Example:

```sh
jampuppy -p 8080 -v -r "/dogenet http://localhost:8080/dogenet /txindex http://localhost:8081/txindex" /mysite


<requests in/out printed here>
```

Usage:

```sh
jampuppy <args> <static path to files>
```

Arguments:

* -p PORT, default 8080
* -r REVERSE PROXY, default ""
* -v VERBOSE, default false, colour prints ingoing/outgoing requests for debugging

