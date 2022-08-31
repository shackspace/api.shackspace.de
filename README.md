# api.shackspace.de backend implementation

A simple go implementation of https://api.shackspace.de/.

## Building

### Prerequisites

- git
- go (version 1.16 or newer)

### Compiling

```sh-session
[user@host api.shackspace.de]$ go version
go version go1.16 linux/amd64
[user@host api.shackspace.de]$ go build
[user@host api.shackspace.de]$ file ./api
./api: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2, Go BuildID=Rq6DNQxp90f_WbIr2EgB/jYDuGINJqZdXJ-rUk51w/yviNggPxIbCo77Atn-_X/ksm2rpW7EwlrObOtKOEA, with debug_info, not stripped
[user@host api.shackspace.de]$
```

## Usage

```
api <space_api_def> <api_token> <status db> [<binding>]
```

- `space_api_def` is the json file for the space api information that should be served. See `www/space-api.json` for an example file. The user requires read permission to that file.
- `api_token` is a file that contains the auth token that is used by the ping process (see below) to authenticate itself. See `www/auth-token.txt` for an example file. The user requires read permission to that file.
- `status db` is a file that contains the time stamp of when the shackspace was last seen. This file is created automatically and the user running the api needs write permission on the file. It will be updated regularly.
- `binding` is optional and will define the IP and port the api binds to. The default is `127.0.0.1:8910`.

The api should not be exposed directly to the internet but should be used behind a reverse proxy to provide HTTPS and a secure internet facing frontend.

**Example:**

```sh-session
[user@host api.shackspace.de]$ ./api www/space-api.json www/auth-token.txt www/last-seen.txt
```

After that command is run, open http://127.0.0.1:8910/ to see the API.

## Function

The api servers several purposes. The main purpose right now is exposing the [portal](https://wiki.shackspace.de/infrastruktur/portal300) status to the world. Other functions are providing a [SpaceAPI](https://spaceapi.io/) endpoint and some logging of the opening times of shackspace.

The available end points are:

- `/`  
  Prints a human-readable overview over the API.
- `/v1/space`  
  Prints the current space status as a small json string.
- `/v1/online`  
  Prints an empty list of members. Was part of the [shackles](https://wiki.shackspace.de/doku/rz/shackdns?s[]=shackles#shackles) system, is currently not implemented.
- `/v1/plena/next`  
  Prints the next plenum date in a json string. Both in human-readable form, and in an ISO timestamp. Also contains an URL to the next plenum agenda.
- `/v1/plena/next?redirect`  
  HTTP-forwards the user to the next plenum agenda.
- `/v1/spaceapi`  
  The [SpaceAPI](https://spaceapi.io/) endpoint.
- `/v1/stats/portal`  
  _unimplemented endpoint for portal stats_
- `/v1/space/notify-open?auth_token=<value>`
  Notification url that when invoked with the right parameters, will mark the shack open for 5 minutes. should be regularly invoked with `?auth_token=<magic>` to set the status to _open_. 5 minutes after the last invocation, status will change to _closed_.
