# Service Naming for DNS

`wagl` discovers containers in your Swarm cluster using
[Docker container labels][docker-labels]:

* `dns.service`
* `dns.domain` (optional)

Labels can be specified with `-l` option to `docker run` command. 

If you run a container with `dns.service` label such as:

    docker run -d -l dns.service=web -p 5000:80 [image]

The following DNS resource records will be created:

> | Class | Domain | 
> |-------|---------|
> | A | `web.swarm.` |
> | SRV | `_web._tcp._swarm.` |

If you add `dns.domain` label with `-l dns.domain=a.b` the generated DNS records
will be:

> | Class | Domain | 
> |-------|---------|
> | A | `web.a.b.swarm.` |
> | SRV | `_web._tcp.a.b.swarm.` |

### Port and Protocol for SRV records

If the container has multiple ports open, only the port **appearing first** in
the list is used to generate DNS SRV records. 

If it is an UDP port then the protocol segment in the SRV record generated would
have an `_udp` segment instead of `_tcp`, such as
`_servicename._udp[.domain.name].swarm`.

[docker-labels]: https://docs.docker.com/userguide/labels-custom-metadata/