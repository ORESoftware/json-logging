<!-- BEGIN k8s-cluster-submodule-notice -->
> [!NOTE]
> **Canonical source.** This repository is the source of truth for its code. It
> is also vendored as a **secondary** git submodule of
> [ORESoftware/k8s-cluster](https://github.com/ORESoftware/k8s-cluster) at
> `remote/modules/github/oresoftware/json-logging` — make changes here, not in that submodule checkout.
>
> On disk: source clone `~/codes/ores/json-logging` · submodule checkout `~/codes/ores/k8s-cluster/remote/modules/github/oresoftware/json-logging`.
<!-- END k8s-cluster-submodule-notice -->

# @oresoftware/json-logging

### json-logging is:

>
> * very high-performance
> * developer friendly
> * opinionated 
> * only writes to stdout, never to stderr
>


### json-logging accomplishes two things:

1. When stdout is connected to a terminal (tty), it writes beautiful output like so:



2. When stdout is not connected to a tty, then it writes optimized JSON in an array form:


To force the use of JSON output, you can do:

```bash
myapp | cat
```

or set an env var:

```bash
jlog_force_json=yes myapp 
```


### Implementation details / features

1. Handles pointers, structs, slices/arrays, maps, and primitives
2. Includes facilities for easily writing raw serialized data (string, byte) to stdout (and stderr if you want).
3. The array format is optimized for performance and also developer friendliness since it is much less verbose.


### The array format:

```
[date, level, appname, pid, hostname, {customFields}, [...messages]]
```

Custom fields are used to filter the logs easily, for example if you want to filter the logs by request id:

```
[date, level, appname, pid, hostname, {requestId: ""}, [...messages]]
```

where requestId is a uuid for a particular request to an HTTP server.





