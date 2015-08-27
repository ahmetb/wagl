# Best Practices


### Where to run wagl container?

Ideally, you might want to co-locate the `wagl` container with the Swarm
manager(s) so that it is not affected by IP changes etc. 

Alternatively, if your Swarm manager(s) have hostnames in the Virtual Network
that can resolve into IP addresses (such as `swarm-master-0`).


### Running multiple instances of wagl

Should be safe.

Deploying multiple `wagl` containers in the case of multiple Swarm managers
should just work fine even though  `wagl` queries different Swarm managers. This
is because Swarm manager nodes in the *follower* state should be proxying the
query to the Swarm *leader* already.


### Locating wagl with `docker run --dns`

It's suggested to have a fixed private IP address for Swarm managers in your
cluster (such as 10.0.0.1/2/3). This way you can specify multiple `--dns`
arguments to `docker run` such as:

    docker run --dns 10.0.0.1 --dns 10.0.0.2 --dns 10.0.0.3 [image]

Also, instead of passing `--dns` argument to `docker run` every time, you might
want to provide it directly to the docker daemons on Swarm nodes:

    docker daemon --dns 10.0.0.1 --dns 10.0.0.0.2 [...]

This way the container can use redundancy of having multiple DNS servers (`wagl`
containers) and tolerate failures.

### Handling external DNS queries

If you really care about external DNS queries, you should use BIND to route
`swarm.` queries to `wagl` and the rest to a proper DNS server rather than using
`wagl`'s primitive forwarding feature.

Read [External DNS topic](4-External-DNS.md) about this.


### Starting wagl with `--restart=always`

`wagl` prefers consistency over liveness.

This means `wagl` will exit if the DNS records go stale a lot (see `--staleness`
argument).

This could be because of a problem in the network or communicating with Swarm
manager.

If `wagl` is running inside a Docker container, then it's suggested to start
`wagl` with `docker run --restart=always`

If it is running standalone, it is suggested to run `wagl` on top of an init
system such as supervisor, runit.

### Running on the host or in a container

There is not much difference running wagl directly on a host or inside a
container.

If your docker daemon is not running as root, you may not bind DNS server to
port 53 (requires sudo).

Also managing the container and handling crashes is easier within a container.
Therefore it is the recommended way.