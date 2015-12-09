# Command Line Interface

Good thing about `wagl` is you almost never need to know about the command-line
arguments. It has good defaults.

By running `wagl --help` (or `docker run --rm ahmet/wagl --help`) you
can see the command line arguments: 

```
$ wagl --help
NAME:
   wagl - DNS service discovery for Docker Swarm clusters

USAGE:
   wagl [options] command [command options]

VERSION:
   0.1

OPTIONS:
   --bind ":53"				IP:port on which the server shoud listen
   --swarm "127.0.0.1:2376"		address of the Swarm manager
   --swarm-cert-path 			directory TLS certs for Swarm manager is stored [$DOCKER_CERT_PATH]
   --swarm-tlsverify			verify remote Swarm's identity using TLS [$DOCKER_TLS_VERIFY]
   --domain "swarm."			DNS domain (FQDN suffix) for which this server is authoritative
   --external				use external nameservers to resolve DNS requests outside the domain (true by default)
   --ns [--ns option --ns option]	external nameserver(s) to forward requests (default: nameservers in /etc/resolv.conf)
   --refresh "15s"			how frequently refresh DNS table from cluster records
   --refresh-timeout "10s"		time alotted for Swarm to list containers in the cluster
   --staleness "1m0s"			how long to serve stale DNS records before exiting
   --help, -h				show help
   --version, -v			print the version
```

Some of these arguments can be also picked up from the environment (those
specified as [$ENV] above).