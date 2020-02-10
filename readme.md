

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

