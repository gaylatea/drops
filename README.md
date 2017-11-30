# Drops
"drops" is a small IOT routing / telemetry server I wrote for my home automation system.

See [`PROTOCOL.md`](PROTOCOL.md) for details on how it works.

## SSL
"drops" uses SSL (with client CA verification) to prevent unauthorized access
to the system. Sample certs are included in `ssl/insecure` for testing, but
please do not deploy any production system with them.

## Building
"drops" is built using Bazel.

Building for my production system (a linux VPS):
```
bazel build --cpu=k8 //server:server
```