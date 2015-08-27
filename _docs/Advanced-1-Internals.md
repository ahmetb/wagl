# Advanced: wagl Internals

### wagl Polls Records

`wagl` refreshes the DNS records by polling the Swarm API periodically (see the
`--refresh` option).

In that sense `wagl` does not rely on Docker Events API as they could be tricky
and could easily end up with message losses. In the future, a combination of
both polling and the events API are planned to be used in the future.