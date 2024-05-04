# QUECTOOL

This is a GO server to interface with a Quectel Modem

## Compiling
This is designed as a go module aware program and thus requires go 1.11 or better
You can clone it anywhere, just run `make` inside the cloned directory to build

Generally it's designed to go on Quectel Modems and you can make an armv7 binary with
`make armv7` and it will output `build/quectool-armv7`

## Configuration
The configuration is designed to be specified with environment variables in all caps with underscores instead of periods. 
```
Example:
LOGGER_LEVEL=debug
```

### Options:
| Setting                         | Description                                                 | Default                 |
| ------------------------------- | ----------------------------------------------------------- | ----------------------- |
| logger.level                    | The default logging level                                   | "info"                  |
| logger.encoding                 | Logging format (console, json or stackdriver)               | "console"               |
| logger.color                    | Enable color in console mode                                | true                    |
| logger.dev_mode                 | Dump additional information as part of log messages         | true                    |
| logger.disable_caller           | Hide the caller source file and line number                 | false                   |
| logger.disable_stacktrace       | Hide a stacktrace on debug logs                             | true                    |
| ---                             | ---                                                         | ---                     |
| pidfile                         | If set, creates a pidfile at the given path                 | ""                      |
| ---                             | ---                                                         | ---                     |
| server.host                     | The host address to listen on (blank=all addresses)         | ""                      |
| server.port                     | The port number to listen on                                | 8900                    |
| server.tls                      | Enable https/tls                                            | false                   |
| server.devcert                  | Generate a development cert                                 | false                   |
| server.certfile                 | The HTTPS/TLS server certificate                            | "server.crt"            |
| server.keyfile                  | The HTTPS/TLS server key file                               | "server.key"            |
| server.log.enabled              | Log server requests                                         | true                    |
| server.log.level                | Log level for server requests                               | "info                   |
| server.log.request_body         | Log the request body                                        | false                   |
| server.log.response_body        | Log the response body                                       | false                   |
| server.log.ignore_paths         | The endpoint prefixes to not log                            | []string{"/version"}    |
| server.cors.enabled             | Enable CORS middleware                                      | false                   |
| server.cors.allowed_origins     | CORS Allowed origins                                        | []string{"*"}           |
| server.cors.allowed_methods     | CORS Allowed methods                                        | []string{...everything} |
| server.cors.allowed_headers     | CORS Allowed headers                                        | []string{"*"}           |
| server.cors.allowed_credentials | CORS Allowed credentials                                    | false                   |
| server.cors.max_age             | CORS Max Age                                                | 300                     |
| modem.port                      | The port the modem is on                                    | /dev/smd11              |
| modem.timeout                   | The timeout for modem commands                              | 5s                      |


## TLS/HTTPS
You can enable https by setting the config option server.tls = true and pointing it to your keyfile and certfile.
To create a self-signed cert: `openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 -keyout server.key -out server.crt`
It also has the option to automatically generate a development cert every time it runs using the server.devcert option.

## API
It's safe to call and endpoints concurrently. The tool will manage concurrent access to the port. 

* GET /api/atcmd?cmd=AT+whatever (make sure to urlencode values)

```
GET /api/atcmd?cmd=AT%2BCGMM%3B%2BQGMR%3B%2BCGCONTRDP%3D1%3B%2BQUIMSLOT%3F
{
  "status": "OK",
  "response": [
    "RM521F-GL",
    "RM521FGLEAR05A02M4G_01.200.01.200",
    "+CGCONTRDP: 1,4,\"fbb.home\",\"38.7.251.145.22.29.24.56.10.210.41.81.126.70.43.228\", \"254.128.0.0.0.0.0.0.180.109.87.255.254.69.69.69\", \"253.0.151.106.0.0.0.0.0.0.0.0.0.0.0.9\", \"253.0.151.106.0.0.0.0.0.0.0.0.0.0.0.16\"",
    "+QUIMSLOT: 1"
  ]
}
```

* GET /api/probe/http?target=https://google.com
* GET /api/probe/ping?target=1.1.1.1
