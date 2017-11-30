# Drops Protocol

* Listen on TCP port
* Line-based protocol (similar to Redis)
* SSL client verification (to avoid unwanted access)

* Long-lived connections, stream data / commands back and forth.

**All commands listed here are newline-delimited.**

**Devices can serve as both clients and stations.**

---

## Stations
"Stations" are devices that perform some action for a smart home setup. A station
connects to the Drops server and waits for commands, occasionally sending
measurement telemetry to the server as needed.

### Station Commands
*Commands marked with `<-` are sent to the station, whereas commands marked
with `->` come from the station to the Drops server. The `<-` and `->` symbols
do not appear on the wire.*

**Register with the server.**

Stations register their presence, so that clients and servers can keep track
of what's currently online. We can also use this to alert if something drops off. If the station's TCP connection drops, then it will be removed the the list.

`[type]` is used to signal to clients what sort of device a station is, so they can do device-specific handling. For instance, a `watersource` type would be displayed in a client very differently than a `heater` type would be.
```
-> REGISTER [name] [type]
<- ACK
```

---
**The following commands will only be possible to receive / send once a client is registered as a "station".**

---

**Run a function on the remote device.**

The included `nonce` will be sent back to the server once the station has
completed the given operation. It is up to the server to decide how this
is generated.

A single parameter is provided for functions that need it. Unused parameters
should be omitted, although Drops itself won't actually validate this since
it has no conception of what the station might support.
```
<- RUN [function] [nonce] [parameter]
```

**Return the result of a function call.**

`nonce` should come from the `RUN` command that triggered the function in the first place. A `[result]` is provided for functions that wish to return data directly back to the caller.

Unused results (i.e., a function that does not return anything) should be omitted, for clarity.
```
-> DONE [function] [nonce] [result]
<- ACK
```

**Return an error to the client of a function call.**
```
-> ERR [function] [nonce]
<- ACK
```

**Report a metric up to the server.**

Drops will store up to 100 of these values for each metric name for each connected station. It's up to other systems to make sense of this data.
```
-> METRIC [name] [value as float]
<- ACK
```

---

## Client
Clients are things like apps, etc, that connect to the Drops server and relay
commands from the user, as well as report back aggregated measurements.

*Commands marked with `<-` are sent to the client, whereas commands marked
with `->` are sent from client to server. The `<-` and `->` symbols do not
appear on the wire.*

**Trigger a function of a connected station.**
```
-> RUN [name] [function] [nonce] [parameter]
<- ACK
```

**Signal the interested client that the function is done.**
```
<- DONE [name] [function] [nonce] [result]
```

**Signal the interested client that the function did not complete.**
```
<- ERR [name] [function] [nonce]
```

**Request a list of the current stations.**
```
-> LIST
<- LIST [name]:[type] ...
```

**Request a list of available metrics from a given station.**
```
-> METRICS [name]
<- METRICS [name] [metric] ...
```

**Request measurements for a given metric from a station.**
```
-> METRICS [name] [metric]
<- METRICS [name] [metric] [ts]:[value] ...
```
