[![Go Report](https://goreportcard.com/badge/github.com/topi314/gobin/v2)](https://goreportcard.com/report/github.com/topi314/gobin/v2)
[![Go Version](https://img.shields.io/github/go-mod/go-version/topi314/gobin)](https://golang.org/doc/devel/release.html)
[![KittyBot License](https://img.shields.io/github/license/topi314/gobin)](LICENSE)
[![KittyBot Version](https://img.shields.io/github/v/tag/topi314/gobin?label=release)](https://github.com/topi314/gobin/releases/latest)
[![Server](https://github.com/topi314/gobin/actions/workflows/server.yml/badge.svg)](https://github.com/topi314/gobin/actions/workflows/server.yml)
[![CLI](https://github.com/topi314/gobin/actions/workflows/cli.yml/badge.svg)](https://github.com/topi314/gobin/actions/workflows/cli.yml)
[![Discord](https://discordapp.com/api/guilds/608506410803658753/embed.png?style=shield)](https://discord.gg/sD3ABd5)

# gobin

gobin is a simple lightweight haste-server alternative written in Go, HTML, JS and CSS. It is aimed to be easy to use
and deploy. You can find an instance running
at [xgob.in](https://xgob.in).

<details>
<summary>Table of Contents</summary>

- [Features](#features)
- [Installation](#installation)
    - [Server](#server)
        - [Docker](#docker)
        - [Manual](#manual)
            - [Requirements](#requirements)
            - [Build](#build)
            - [Run](#run)
    - [CLI](#cli)
        - [Release](#release)
        - [Manual](#manual-1)
            - [Requirements](#requirements-1)
            - [Build](#build-1)
            - [Run](#run-1)
- [Configuration](#configuration)
- [Custom Themes](#custom-themes)
- [Rate Limit](#rate-limits)
- [API](#api)
    - [Errors](#errors)
    - [Formatter Enum](#formatter-enum)
    - [Language Enum](#language-enum)
    - [Create a document](#create-a-document)
        - [Single file](#single-file)
        - [Multiple files](#multiple-files)
    - [Get a document (version)](#get-a-document-version)
    - [Get a document (version) file](#get-a-document-version-file)
    - [Get a documents versions](#get-a-documents-versions)
    - [Update a document](#update-a-document)
        - [Single file](#single-file-1)
        - [Multiple files](#multiple-files-1)
    - [Delete a document (version)](#delete-a-document-version)
    - [Share a document](#share-a-document)
    - [Document webhooks](#document-webhooks)
        - [Create a document webhook](#create-a-document-webhook)
        - [Update a document webhook](#update-a-document-webhook)
        - [Delete a document webhook](#delete-a-document-webhook)
    - [Other endpoints](#other-endpoints)
- [License](#license)
- [Contributing](#contributing)
- [Credits](#credits)
- [Contact](#contact)

</details>

## Features

- Easy to deploy and use
- Built-in rate-limiting
- Create, update and delete documents
- Document update/delete webhooks
- Syntax highlighting
- Social Media PNG previews
- Document expiration
- Supports [PostgreSQL](https://www.postgresql.org/) or [SQLite](https://sqlite.org/)
- One binary and config file
- Docker image available
- ~~Metrics (to be implemented)~~
- [base16](https://github.com/chriskempson/base16) & [chroma](https://github.com/topi314/chroma) custom themes

## Installation

### Server

#### Docker

The easiest way to deploy gobin is using docker with [Docker Compose](https://docs.docker.com/compose/). You can find
the docker image
on [Packages](https://github.com/topi314/gobin/pkgs/container/gobin).

Create a new `docker-compose.yml` file with the following content:

> [!Note]
> You should change the password in the `docker-compose.yml` and `gobin.toml` file.

```yaml
version: "3.8"

services:
  gobin:
    image: ghcr.io/topi314/gobin:latest
    container_name: gobin
    restart: unless-stopped
    volumes:
      - ./gobin.toml:/var/lib/gobin/gobin.toml
      # use this for sqlite
      - ./gobin.db:/var/lib/gobin/gobin.db
    ports:
      - 80:80

  # or use this for postgres
  postgres:
    image: postgres:latest
    container_name: postgres
    restart: unless-stopped
    volumes:
      - ./data:/var/lib/postgresql/data
    environment:
      POSTGRES_DB: gobin
      POSTGRES_USER: gobin
      POSTGRES_PASSWORD: password
```

For `gobin.toml`/database schema see [Configuration](#configuration).

```bash
docker-compose up -d
```

---

#### Manual

##### Requirements

- Go 1.21 or higher

##### Build

```bash
git clone https://github.com/topi314/gobin.git
cd gobin
go build -o gobin github.com/topi314/gobin/v2
```

or

```bash
go install github.com/topi314/gobin/v2@latest
```

##### Run

```bash
gobin --config=gobin.toml
```

---

### CLI

#### Release

You can find the latest release on [Releases](https://github.com/topi314/gobin/releases).

#### Manual

##### Requirements

- Go 1.21 or higher

##### Build

```bash
git clone https://github.com/topi314/gobin.git
cd gobin
go build -o gobin github.com/topi314/gobin/v2/cli
```

or

```bash
go install github.com/topi314/gobin/v2/cli@latest
# rename binary to gobin
mv $(go env GOPATH)/bin/cli $(go env GOPATH)/bin/gobin
# or move binary into /usr/local/bin
mv $(go env GOPATH)/bin/cli /usr/local/bin/gobin
# change file ownership to root 
chown 0:0 /usr/local/bin/gobin
```

##### Run

```bash
gobin help
```

---

## Configuration

The database schema is automatically created or migrated when you start gobin.

Create a new `gobin.toml` file with the following content:

> [!Note]
> Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

```json5
{
  "log": {
    // level can be -4 (debug), 0 (info), 4 (warn), 8 (error)
    "level": 0,
    // log format, either "json" or "text"
    "format": "text",
    // whether to add the source file and line to the log output
    "add_source": false,
    // whether to add color to the log output (only for text format)
    "no_color": false
  },
  // enable or disable debug profiler endpoint
  "debug": false,
  // enable or disable hot reload of templates and assets
  "dev_mode": false,
  "listen_addr": "0.0.0.0:80",
  // secret for jwt tokens, replace with a long random string
  "jwt_secret": "...",
  "database": {
    // either "postgres" or "sqlite"
    "type": "postgres",
    "debug": false,
    "expire_after": "168h",
    "cleanup_interval": "10m",
    // path to sqlite database
    // if you run gobin with docker make sure to set it to "/var/lib/gobin/gobin.db"
    "path": "gobin.db",
    // postgres connection settings
    "host": "localhost",
    "port": 5432,
    "username": "gobin",
    "password": "password",
    "database": "gobin",
    "ssl_mode": "disable"
  },
  // max character count for all files in a document combined (0 to disable)
  "max_document_size": 0,
  // max_highlight_size is the max character count for a single file in a document to be highlighted (0 to disable)
  "max_highlight_size": 0,
  // omit or set values to 0 or "0" to disable rate limit
  "rate_limit": {
    // number of requests which can be done in the duration
    "requests": 10,
    // the duration of the requests
    "duration": "1m",
    // a list of ip addresses which are exempt from rate limiting
    "whitelist": [
      "127.0.0.1"
    ],
    // a list of ip addresses which are blocked from rate limited endpoints
    "blacklist": [
      "123.456.789.0"
    ]
  },
  // settings for social media previews, omit to disable
  "preview": {
    // path to inkscape binary https://inkscape.org/
    "inkscape_path": "/usr/bin/inkscape",
    // how many lines should be shown in the preview
    "max_lines": 10,
    // how high the resolution of the preview should be, 96 is the default
    "dpi": 96,
    // how many previews should be maximally cached
    "cache_size": 1024,
    // how long should previews be cached
    "cache_duration": "1h"
  },
  // open telemetry settings, omit to disable
  "otel": {
    // the instance id of the server
    "instance_id": "1",
    // otel trace settings, omit to disable
    "trace": {
      // the address of the tempo instance
      "endpoint": "tempo:4318",
      // whether to use an insecure connection
      "insecure": true
    },
    // otel metrics settings, omit to disable
    "metrics": {
      // the address where the metrics should be exposed
      "listen_addr": ":9100"
    }
  },
  // settings for webhooks, omit to disable
  "webhook": {
    // webhook reqauest timeout
    "timeout": "10s",
    // max number of tries to send a webhook
    "max_tries": 3,
    // how long to wait before retrying a webhook
    "backoff": "1s",
    // how much the backoff should be increased after each retry
    "backoff_factor": 2,
    // max backoff time
    "max_backoff": "5m"
  },
  // load custom chroma xml or base16 yaml themes from this directory, omit to disable
  "custom_styles": "custom_styles",
  "default_style": "snazzy"
}
```

Alternatively you can use environment variables to configure gobin. The environment variables are prefixed with `GOBIN_`
and are in uppercase. For example `GOBIN_DATABASE_TYPE`
or `GOBIN_RATE_LIMIT_REQUESTS`.

<details>
<summary>Here is a list of all environment variables</summary>

```env
GOBIN_LOG_LEVEL=info
GOBIN_LOG_FORMAT=text
GOBIN_LOG_ADD_SOURCE=false

GOBIN_DEBUG=false
GOBIN_DEV_MODE=false
GOBIN_LISTEN_ADDR=0.0.0.0:80
GOBIN_JWT_SECRET=...

GOBIN_DATABASE_TYPE=postgres
GOBIN_DATABASE_DEBUG=false
GOBIN_DATABASE_EXPIRE_AFTER=168h
GOBIN_DATABASE_CLEANUP_INTERVAL=10m

GOBIN_DATABASE_PATH=gobin.db

GOBIN_DATABASE_HOST=localhost
GOBIN_DATABASE_PORT=5432
GOBIN_DATABASE_USERNAME=gobin
GOBIN_DATABASE_PASSWORD=password
GOBIN_DATABASE_DATABASE=gobin
GOBIN_DATABASE_SSL_MODE=disable

GOBIN_MAX_DOCUMENT_SIZE=0
GOBIN_MAX_HIGHLIGHT_SIZE=0

GOBIN_RATE_LIMIT_REQUESTS=10
GOBIN_RATE_LIMIT_DURATION=1m

GOBIN_PREVIEW_INKSCAPE_PATH=/usr/bin/inkscape
GOBIN_PREVIEW_MAX_LINES=10
GOBIN_PREVIEW_DPI=96
GOBIN_PREVIEW_CACHE_SIZE=1024
GOBIN_PREVIEW_CACHE_TTL=1h

GOBIN_WEBHOOK_TIMEOUT=10s
GOBIN_WEBHOOK_MAX_TRIES=3
GOBIN_WEBHOOK_BACKOFF=1s
GOBIN_WEBHOOK_BACKOFF_FACTOR=2
GOBIN_WEBHOOK_MAX_BACKOFF=5m

GOBIN_CUSTOM_STYLES=custom_styles
GOBIN_DEFAULT_STYLE=snazzy
```

</details>

---

## Custom Themes

You can add your own themes to gobin by adding them to the `themes` directory.

### Gobin Themes

Gobin themes are written in TOML file format.

All keys specified under `colors` are variables which can be referenced by `$name` in the theme file.

To style UI elements you can use the `ui` key. See below for all available keys.

To style individual scopes you can use the `styles` key. The key is the scope and the value is the style.
Make sure to quote the keys if the scope name contain a `.`.

```toml file=themes/name.toml
name = 'name'
color_scheme = 'dark' # or 'light'

tab_size = 4

[colors]
text0 = '#F8F8F2'
text1 = '#F8F8F2'
text2 = '#8A8A8A'

background0 = '#212122'
background1 = '#2B2B2B'
background2 = '#3C3C3C'
background3 = '#43494A'

white = '#FEFEF8'
red = '#FF4352'
blue = '#73FBF1'
green = '#B8E466'
yellow = '#FFD750'
magenta = '#A578EA'
gray = '#6D7070'

[ui]
status_bar = '$text1'
status_bar_background = '$background1'
status_bar_active_background = '$background2'

code = '$text0'
code_background = '$background0'

line_number = '$text2'
line_number_background = '$background1'
highlight = '$background2'

symbols = '$text1'
symbols_background = '$background1'
symbols_active_background = '$background3'
synbols_kind_background = '$background1'

[styles]
"variable" = { text = '$text' }
"variable.other.member" = { text = '$red' }
"function" = { text = '$blue' }
"method" = { text = '$blue' }
"string" = { text = '$green' }
"type" = { text = '$yellow' }
"keyword" = { text = '$magenta' }
"comment" = { text = '$gray' }
"comment.todo" = { text = '$white' }
```

### Base16 Themes

Base16 themes are supported and should be placed in the `themes/base16` directory.

See [base16](https://github.com/chriskempson/base16) for more information about base16 themes.

```yaml file=custom_styles/name.yaml
scheme: "name"
author: "author"
color_scheme: "dark" # or "light"
base00: "282a36"
base01: "34353e"
base02: "43454f"
base03: "78787e"
base04: "a5a5a9"
base05: "e2e4e5"
base06: "eff0eb"
base07: "f1f1f0"
base08: "ff5c57"
base09: "ff9f43"
base0A: "f3f99d"
base0B: "5af78e"
base0C: "9aedfe"
base0D: "57c7ff"
base0E: "ff6ac1"
base0F: "b2643c"
```


## Rate Limits

All `POST`, `PATCH` and `DELETE` endpoints are rate limited. The rate limit can be configured in the config file.
The bucket is based on the IP address and the path of the request. So each of these unique combinations has its own bucket/rate limit.

It's based on a sliding window algorithm, but instead of a fixed window the window will start at the first request and
end after the duration. So if you set the duration to 1 minute and send 10 requests in the first 10 seconds you will be rate limited for 50 seconds. After that you can send 10 requests
again.

Gobin will return these headers to help clients keep track of the rate limit:

| Header                | Description                                                                    |
|-----------------------|--------------------------------------------------------------------------------|
| X-RateLimit-Limit     | The maximum number of requests which can be done in the duration.              |
| X-RateLimit-Remaining | The number of remaining requests which can be done in the duration.            |
| X-RateLimit-Reset     | The time when the rate limit will be reset in unix timestamp.                  |
| Retry-After           | The time when the rate limit will be reset in seconds. (only when hit a `429`) |

---

## API

Fields marked with `?` are optional and types marked with `?` are nullable.

### Errors

In case of an error gobin will return the following JSON body with the corresponding HTTP status code:

```json5
{
  "message": "document not found",
  // error message
  "status": 404,
  // HTTP status code
  "path": "/documents/7df3vw",
  // request path
  "request_id": "fbe0a365387f/gVAMGuraLW-003490"
  // request id
}
```

---

### Formatter Enum

Document formatting is done using [chroma](https://github.com/topi314/chroma). The following formatters are
available:

| Value           | Description             |
|-----------------|-------------------------|
| terminal8       | 8-bit terminal colors   |
| terminal16      | 16-bit terminal colors  |
| terminal256     | 256-bit terminal colors |
| terminal16m     | true terminal colors    |
| html            | HTML                    |
| html-standalone | Standalone HTML         |
| svg             | SVG                     |

---

### Language Enum

The following languages are available:

<details>
<summary>Click to expand</summary>

| Prefix | Language                                                                                                                                                                                                          |
|--------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| A      | ABAP, ABNF, ActionScript, ActionScript 3, Ada, Angular2, ANTLR, ApacheConf, APL, AppleScript, Arduino, Awk                                                                                                        |
| B      | Ballerina, Bash, Batchfile, BibTeX, Bicep, BlitzBasic, BNF, Brainfuck, BQN                                                                                                                                        |
| C      | C, C#, C++, Caddyfile, Caddyfile Directives, Cap'n Proto, Cassandra CQL, Ceylon, CFEngine3, cfstatement, ChaiScript, Chapel, Cheetah, Clojure, CMake, COBOL, CoffeeScript, Common Lisp, Coq, Crystal, CSS, Cython |
| D      | D, Dart, Diff, Django/Jinja, Docker, DTD, Dylan                                                                                                                                                                   |
| E      | EBNF, Elixir, Elm, EmacsLisp, Erlang                                                                                                                                                                              |
| F      | Factor, Fish, Forth, Fortran, FSharp                                                                                                                                                                              |
| G      | GAS, GDScript, Genshi, Genshi HTML, Genshi Text, Gherkin, GLSL, Gnuplot, Go, Go HTML Template, Go Text Template, GraphQL, Groff, Groovy                                                                           |
| H      | Handlebars, Haskell, Haxe, HCL, Hexdump, HLB, HLSL, HTML, HTTP, Hy                                                                                                                                                |
| I      | Idris, Igor, INI, Io                                                                                                                                                                                              |
| J      | J, Java, JavaScript, JSON, Julia, Jungle                                                                                                                                                                          |
| K      | Kotlin                                                                                                                                                                                                            |
| L      | Lighttpd configuration file, LLVM, Lua                                                                                                                                                                            |
| M      | Makefile, Mako, markdown, Mason, Mathematica, Matlab, MiniZinc, MLIR, Modula-2, MonkeyC, MorrowindScript, Myghty, MySQL                                                                                           |
| N      | NASM, Newspeak, Nginx configuration file, Nim, Nix                                                                                                                                                                |
| O      | Objective-C, OCaml, Octave, OnesEnterprise, OpenEdge ABL, OpenSCAD, Org Mode                                                                                                                                      |
| P      | PacmanConf, Perl, PHP, PHTML, Pig, PkgConfig, PL/pgSQL, plaintext, Pony, PostgreSQL SQL dialect, PostScript, POVRay, PowerShell, Prolog, PromQL, Properties, Protocol Buffer, PSL, Puppet, Python 2, Python       |
| Q      | QBasic                                                                                                                                                                                                            |
| R      | R, Racket, Ragel, Raku, react, ReasonML, reg, reStructuredText, Rexx, Ruby, Rust                                                                                                                                  |
| S      | SAS, Sass, Scala, Scheme, Scilab, SCSS, Sed, Smalltalk, Smarty, Snobol, Solidity, SPARQL, SQL, SquidConf, Standard ML, stas, Stylus, Svelte, Swift, SYSTEMD, systemverilog                                        |
| T      | TableGen, TASM, Tcl, Tcsh, Termcap, Terminfo, Terraform, TeX, Thrift, TOML, TradingView, Transact-SQL, Turing, Turtle, Twig, TypeScript, TypoScript, TypoScriptCssData, TypoScriptHtmlData                        |
| V      | VB.net, verilog, VHDL, VHS, VimL, vue                                                                                                                                                                             |
| W      | WDTE                                                                                                                                                                                                              |
| X      | XML, Xorg                                                                                                                                                                                                         |
| Y      | YAML, YANG                                                                                                                                                                                                        |
| Z      | Zig                                                                                                                                                                                                               |

</details>

--- 

### Create a document

You can create a document with a single file or multiple files. When creating a document with a single file you can
simply `POST` the content to `/documents`.
When creating a document with multiple files you have to `POST` the content to `/documents` as `multipart/form-data`.
See below for more information.

#### Single file

To create a document with a single file you have to send a `POST` request to `/documents` with the `content` as body.

| Header               | Type      | Description                                             |
|----------------------|-----------|---------------------------------------------------------|
| Content-Disposition? | string    | The file name of the document.                          |
| Content-Type?        | string    | The content type of the document.                       |
| Language?            | string    | The language of the document.                           |
| Expires?             | Timestamp | When the document file should expire in RFC 3339 format |

| Query Parameter | Type                         | Description                                             |
|-----------------|------------------------------|---------------------------------------------------------|
| language?       | [language](#language-enum)   | The language of the document.                           |
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document.            |
| style?          | style name                   | Which style to use for the formatter                    |
| expires?        | Timestamp                    | When the document file should expire in RFC 3339 format |

<details>
<summary>Example</summary>

```go
package main

func main() {
	println("Hello World!")
}
```

</details>

#### Multiple files

To create a document with multiple files you have to send a `POST` request to `/documents` with the `content`
as `multipart/form-data` body.
Each file has to be in its own part with the name `file-{index}`. The first file has to be named `file-0`, the
second `file-1` and so on.

| Query Parameter | Type                         | Description                                             |
|-----------------|------------------------------|---------------------------------------------------------|
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document.            |
| style?          | style name                   | Which style to use for the formatter                    |
| expires?        | Timestamp                    | When the document file should expire in RFC 3339 format |

| Header   | Type      | Description                                             |
|----------|-----------|---------------------------------------------------------|
| Expires? | Timestamp | When the document file should expire in RFC 3339 format |

| Part Header         | Type      | Description                                                                                  |
|---------------------|-----------|----------------------------------------------------------------------------------------------|
| Content-Disposition | string    | The form & file name of the document.                                                        |
| Content-Type?       | string    | The content type/language of the document.                                                   |
| Language?           | string    | The language of the document.                                                                |
| Expires?            | Timestamp | When the document file should expire in RFC 3339 format, overwrites the query param & header |

<details>
<summary>Example</summary>

```multipart/form-data
-----------------------------302370379826172687681786440755
Content-Disposition: form-data; name="file-0"; filename="main.go"
Content-Type: text/x-gosrc
Language: Go
Expires: 2023-10-10T10:10:10Z

package main

func main() {
	println("Hello World!")
}
-----------------------------302370379826172687681786440755
Content-Disposition: form-data; name="file-1"; filename="untitled1"
Content-Type: text/plain; charset=utf-8

Hello World!
-----------------------------302370379826172687681786440755--
```

</details>

A successful request will return a `201 Created` response with a JSON body containing the document key and token to
update the document.

```json5
{
  "key": "hocwr6i6",
  "version": 1,
  "files": [
    {
      "name": "main.go",
      "content": "package main\n\nfunc main() {\n    println(\"Hello World!\")\n}",
      // only if formatter is set
      "formatted": "...",
      "language": "Go",
      "expires_at": null
    },
    {
      "name": "untitled1",
      "content": "Hello World!",
      // only if formatter is set
      "formatted": "...",
      "language": "plaintext",
      "expires_at": null
    }
  ],
  "token": "kiczgez33j7qkvqdg9f7ksrd8jk88wba"
}
```

---

### Get a document (version)

To get a document you have to send a `GET` request to `/documents/{key}` or `/documents/{key}/versions/{version}`.

| Query Parameter | Type                         | Description                                                                                        |
|-----------------|------------------------------|----------------------------------------------------------------------------------------------------|
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document                                                        |
| style?          | style name                   | Which style to use for the formatter                                                               |
| file?           | file name                    | Which file to return                                                                               |
| language?       | [language](#language-enum)   | In which language the document should be rendered. Only works in combination with the `file` param |

The response will be a `200 OK` with the document content as `application/json` body.

```json5
{
  "key": "hocwr6i6",
  "version": 1,
  "files": [
    {
      "name": "main.go",
      "content": "package main\n\nfunc main() {\n    println(\"Hello World!\")\n}",
      // only if formatter is set
      "formatted": "...",
      "language": "Go",
      "expires_at": null
    },
    {
      "name": "untitled1",
      "content": "Hello World!",
      // only if formatter is set
      "formatted": "...",
      "language": "plaintext",
      "expires_at": null
    }
  ]
}
```

In case you provide a `file` query param the response will be like from [Get a document file](#get-a-document-version-file)

---

### Get a document (version) file

To get a document (version) file you have to send a `GET` request to `/documents/{key}/files/{fileName}`or `/documents/{key}/versions/{version}/files/{fileName}`

| Query Parameter | Type                         | Description                                  |
|-----------------|------------------------------|----------------------------------------------|
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document. |
| style?          | style name                   | Which style to use for the formatter         |
| language?       | language name                | Which language to use for the formatter      |

The response will be a `200 OK` with the document content as `application/json` body.

```json5
{
  "name": "main.go",
  // only if withContent is set
  "content": "package main\n\nfunc main() {\n    println(\"Hello World!\")\n}",
  // only if formatter is set
  "formatted": "...",
  "language": "Go",
  "expires_at": null
}
```

---

### Get a documents versions

To get a documents versions you have to send a `GET` request to `/documents/{key}/versions`.

| Query Parameter | Type                         | Description                                        |
|-----------------|------------------------------|----------------------------------------------------|
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document        |
| style?          | style name                   | Which style to use for the formatter               |
| withContent?    | bool                         | If the content should be included in the response. |

The response will be a `200 OK` with the document content as `application/json` body.

```json5
[
  {
    "key": "hocwr6i6",
    "version": 2,
    "files": [
      {
        "name": "main.go",
        // only if withContent is set
        "content": "package main\n\nfunc main() {\n    println(\"Hello World!\")\n}",
        // only if formatter is set
        "formatted": "...",
        "language": "Go",
        "expires_at": null
      },
      {
        "name": "untitled1",
        // only if withContent is set
        "content": "Hello World!",
        // only if formatter is set
        "formatted": "...",
        "language": "plaintext",
        "expires_at": null
      }
    ]
  },
  {
    "key": "hocwr6i6",
    "version": 1,
    "files": [
      {
        "name": "main.go",
        "content": "package main\n\nfunc main() {\n    println(\"Hello!\")\n}",
        // only if formatter is set
        "formatted": "...",
        "language": "Go",
        "expires_at": null
      },
      {
        "name": "untitled1",
        "content": "Hello!",
        // only if formatter is set
        "formatted": "...",
        "language": "plaintext",
        "expires_at": null
      }
    ]
  }
]
```

---

### Update a document

You can update a document with a single file or multiple files. When updating a document with a single file you can
simply `PATCH` the content to `/documents/{key}`.
When updating a document with multiple files you have to `PATCH` the content to `/documents/{key}`
as `multipart/form-data`. See below for more information.

#### Single file

To create a document with a single file you have to send a `PATCH` request to `/documents/{key}` with the `content` as
body.

| Header              | Type      | Description                                               |
|---------------------|-----------|-----------------------------------------------------------|
| Content-Disposition | string    | The form & file name of the document.                     |
| Content-Type?       | string    | The content type of the document.                         |
| Language?           | string    | The language of the document.                             |
| Authorization?      | string    | The update token of the document. (prefix with `Bearer `) |
| Expires?            | Timestamp | When the document file should expire in RFC 3339 format   |

| Query Parameter | Type                         | Description                                             |
|-----------------|------------------------------|---------------------------------------------------------|
| language?       | [language](#language-enum)   | The language of the document.                           |
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document.            |
| style?          | style name                   | Which style to use for the formatter                    |
| expires?        | Timestamp                    | When the document file should expire in RFC 3339 format |

<details>
<summary>Example</summary>

```go
package main

func main() {
	println("Hello World Updated!")
}
```

</details>

#### Multiple files

To update a document with multiple files you have to send a `PATCH` request to `/documents/{key}` with the `content`
as `multipart/form-data` body.
Each file has to be in its own part with the name `file-{index}`. The first file has to be named `file-0`, the
second `file-1` and so on.

| Header         | Type      | Description                                               |
|----------------|-----------|-----------------------------------------------------------|
| Authorization? | string    | The update token of the document. (prefix with `Bearer `) |
| Expires?       | Timestamp | When the document file should expire in RFC 3339 format   |

| Query Parameter | Type                         | Description                                             |
|-----------------|------------------------------|---------------------------------------------------------|
| formatter?      | [formatter](#formatter-enum) | With which formatter to render the document.            |
| style?          | style name                   | Which style to use for the formatter                    |
| expires?        | Timestamp                    | When the document file should expire in RFC 3339 format |

| Part Header         | Type      | Description                                                                                  |
|---------------------|-----------|----------------------------------------------------------------------------------------------|
| Content-Disposition | string    | The form & file name of the document.                                                        |
| Content-Type?       | string    | The content type of the document.                                                            |
| Language?           | string    | The language of the document.                                                                |
| Expires?            | Timestamp | When the document file should expire in RFC 3339 format, overwrites the query param & header |

<details>
<summary>Example</summary>

```
Authorization: kiczgez33j7qkvqdg9f7ksrd8jk88wba
```

```multipart/form-data
-----------------------------302370379826172687681786440755
Content-Disposition: form-data; name="file-0"; filename="main.go"
Content-Type: text/x-gosrc
Language: Go
Expires: 2023-10-10T10:10:10Z

package main

func main() {
	println("Hello World!")
}
-----------------------------302370379826172687681786440755
Content-Disposition: form-data; name="file-1"; filename="untitled1"
Content-Type: text/plain; charset=utf-8

Hello World!
-----------------------------302370379826172687681786440755--
```

</details>

A successful request will return a `201 Created` response with a JSON body containing the document key and token to
update the document.

> [!Note]
> The update token will not change after updating the document. You can use the same token to update the document again.

```json5
{
  "key": "hocwr6i6",
  "version": 2,
  "files": [
    {
      "name": "main.go",
      "content": "package main\n\nfunc main() {\n    println(\"Hello World Updated!\")\n}",
      // only if formatter is set
      "formatted": "...",
      "language": "Go",
      "expires_at": null
    },
    {
      "name": "untitled1",
      "content": "Hello World Updated!",
      // only if formatter is set
      "formatted": "...",
      "language": "plaintext",
      "expires_at": null
    }
  ],
  "token": "kiczgez33j7qkvqdg9f7ksrd8jk88wba"
}
```

---

### Delete a document (version)

To delete a document you have to send a `DELETE` request to `/documents/{key}` or `/documents/{key}/versions/{version}` with the `token` as `Authorization`
header.

| Header         | Type   | Description                                               |
|----------------|--------|-----------------------------------------------------------|
| Authorization? | string | The update token of the document. (prefix with `Bearer `) |

A successful request will return a `204 No Content` response with an empty body or a `200 OK` with a JSON body
containing the count of remaining document versions:

```json5
{
  "versions": 1
}
```

---

### Share a document

To share a document you have to send a `POST` request to `/documents/{key}/share`.

| Header         | Type   | Description                                               |
|----------------|--------|-----------------------------------------------------------|
| Authorization? | string | The update token of the document. (prefix with `Bearer `) |

```json5
{
  "permissions": [
    "write",
    "delete",
    "share"
  ]
}
```

A successful request will return a `200 OK` response with a JSON body containing the share token.
You can append the token to URLs like this: `https://xgob.in/{key}?token={token}` to make the frontend auto import the
token for editing/deleting/sharing the document.

```json5
{
  "token": "kiczgez33j7qkvqdg9f7ksrd8jk88wba"
}
```

---

### Document webhooks

You can listen for document changes using webhooks. The webhook will send a `POST` request to the specified url with the
following JSON body:

```json5
{
  // the id of the webhook
  "webhook_id": "hocwr6i6",
  // the event which triggered the webhook (update or delete)
  "event": "update",
  // when the event was created
  "created_at": "2021-08-01T12:00:00Z",
  // the updated or deleted document
  "document": {
    // the key of the document
    "key": "hocwr6i6",
    // the version of the document
    "version": 2,
    // the files of the document
    "files": [
      {
        "name": "main.go",
        "content": "package main\n\nfunc main() {\n    println(\"Hello World Updated!\")\n}",
        "language": "Go",
        "expires_at": null
      }
    ]
  }
}
```

Gobin will include the webhook secret in the `Authorization` header in the following format: `Secret {secret}`.

When sending an event to a webhook fails gobin will retry it up to x times with an exponential backoff. The retry
settings can be configured in the config file.
When an event fails to be sent after x retries, the webhook will be dropped.

> [!Important]
> Authorizing for the following webhook endpoints is done using the `Authorization` header in the following
> format: `Secret {secret}`.

#### Create a document webhook

To create a webhook you have to send a `POST` request to `/documents/{key}/webhooks` with the following JSON body:

```json5
{
  // the url to send a request to
  "url": "https://example.com/webhook",
  // the secret to include in the request
  "secret": "secret",
  // the events you want to receive
  "events": [
    // update event is sent when a document is updated. This includes content and language changes
    "update",
    // delete event is sent when a document is deleted
    "delete"
  ]
}
```

A successful request will return a `200 OK` response with a JSON body containing the webhook.

```json5
{
  // the id of the webhook
  "id": 1,
  // the url to send a request to
  "url": "https://example.com/webhook",
  // the secret to include in the request
  "secret": "secret",
  // the events you want to receive
  "events": [
    // update event is sent when a document is updated. This includes content and language changes
    "update",
    // delete event is sent when a document is deleted
    "delete"
  ]
}
```

---

#### Get a document webhook

To get a webhook you have to send a `GET` request to `/documents/{key}/webhooks/{id}` with the `Authorization` header.

A successful request will return a `200 OK` response with a JSON body containing the webhook.

```json5
{
  // the id of the webhook
  "id": 1,
  // the url to send a request to
  "url": "https://example.com/webhook",
  // the secret to include in the request
  "secret": "secret",
  // the events you want to receive
  "events": [
    // update event is sent when a document is updated. This includes content and language changes
    "update",
    // delete event is sent when a document is deleted
    "delete"
  ]
}
```

---

#### Update a document webhook

To update a webhook you have to send a `PATCH` request to `/documents/{key}/webhooks/{id}` with the `Authorization`
header and the following JSON body:

> [!Note]
> All fields are optional, but at least one field is required.

```json5
{
  // the url to send a request to
  "url": "https://example.com/webhook",
  // the secret to include in the request
  "secret": "secret",
  // the events you want to receive
  "events": [
    // update event is sent when a document is updated. This includes content and language changes
    "update",
    // delete event is sent when a document is deleted
    "delete"
  ]
}
```

A successful request will return a `200 OK` response with a JSON body containing the webhook.

```json5
{
  // the id of the webhook
  "id": 1,
  // the url to send a request to
  "url": "https://example.com/webhook",
  // the secret to include in the request
  "secret": "secret",
  // the events you want to receive
  "events": [
    // update event is sent when a document is updated. This includes content and language changes
    "update",
    // delete event is sent when a document is deleted
    "delete"
  ]
}
```

---

#### Delete a document webhook

To delete a webhook you have to send a `DELETE` request to `/documents/{key}/webhooks/{id}` with the `Authorization`
header.

A successful request will return a `204 No Content` response with an empty body.

---

### Other endpoints

- `GET`/`HEAD` `/{key}/files/{filename}` - Get the content of a file in a document, query parameters are the same as
  for `GET /documents/{key}`.
- `GET`/`HEAD` `/{key}/versions/{version}/files/{filename}` - Get the content of a file in a document with a specific
  version, query parameters are the same as for `GET /documents/{key}`.
- `GET`/`HEAD` `/assets/theme.css?style={style}` - Get the css of a style, this is used for the syntax highlighting in
  the frontend.
- `GET`/`HEAD` `/{key}/preview` - Get the preview of a document, query parameters are the same as
  for `GET /documents/{key}`.
- `GET`/`HEAD` `/{key}/{version}/preview` - Get the preview of a document version, query parameters are the same as
  for `GET /documents/{key}/versions/{version}`.
- `GET`/`HEAD` `/raw/{key}` - Get the raw content of a document, query parameters are the same as
  for `GET /documents/{key}`.
- `GET`/`HEAD` `/raw/{key}/files/{filename}` - Get the raw content of a document file, query parameters are the same as
  for `GET /documents/{key}`.
- `GET`/`HEAD` `/raw/{key}/versions/{version}` - Get the raw content of a document version, query parameters are the
  same as for `GET /documents/{key}/versions/{version}`.
- `GET`/`HEAD` `/raw/{key}/versions/{version}/files/{filename}` - Get the raw content of a document version file, query
  parameters are the same as for `GET /documents/{key}/versions/{version}`.
- `GET` `/ping` - Get the status of the server.
- `GET` `/debug` - Proof debug endpoint (only available in debug mode).
- `GET` `/version` - Get the version of the server.

---

## License

gobin is licensed under the [Apache License 2.0](/LICENSE).

---

## Contributing

Contributions are always welcome! Just open a pull request or discussion and I will take a look at it.

---

## Credits

- [@Damon](https://github.com/day-mon) for helping me.

---

## Contact

- [Discord](https://discord.gg/sD3ABd5)
- [Twitter](https://twitter.com/topi314)
- [Email](mailto:git@topi.wtf)
