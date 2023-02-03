# Collection of additional middlewares

![License](https://img.shields.io/dub/l/vibe-d.svg)

Here you'll find middleware ready to use with [Wool Framework](https://github.com/gowool/wool).

| Middleware               | Description                                                                                                                         |
|--------------------------|-------------------------------------------------------------------------------------------------------------------------------------|
| [basicauth](basicauth)   | Basic authentication middleware                                                                                                     |
| [bodylimit](bodylimit)   | Limit request body size middleware                                                                                                  |
| [cors](cors)             | CORS middleware                                                                                                                     |
| [favicon](favicon)       | Favicon middleware that ignores favicon requests or caches a provided icon in memory to improve performance by skipping disk access |
| [gzip](gzip)             | Gzip middleware to enable `GZIP` support                                                                                            |
| [keyauth](keyauth)       | Key authentication middleware                                                                                                       |
| [prometheus](prometheus) | Easily create metrics endpoint for the [prometheus](http://prometheus.io) instrumentation tool                                      |
| [proxy](proxy)           | Proxy inspects common reverse proxy headers and sets the corresponding fields in the HTTP request struct                            |
| [requestid](requestid)   | Request ID middleware that adds an identifier to the response                                                                       |
| [secure](secure)         | Middleware that implements a few quick security wins                                                                                |
| [sse](sse)               | Server-Sent Events implementation                                                                                                   |
| [www](www)               | WWW Middleware                                                                                                                      |

## License

Distributed under MIT License, please see license file within the code for more details.
