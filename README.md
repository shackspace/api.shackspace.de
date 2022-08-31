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
