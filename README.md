## wagl: DNS Service Discovery for Docker Swarm

`wagl` runs inside your [Docker Swarm][sw] cluster and provides
DNS-based service discovery (using DNS A and SRV records) and
simple load balancing by rotating the list of IP addresses in
DNS records.

For instance, if you run your API container with command:

    docker run -d -l dns.service=api -p 80:80 nginx

other containers in the cluster will be able to reach this container using URL
**`http://api.swarm`**. It is a **minimalist** solution, yet handles most of the
basic DNS service discovery functionality well ––but we're open to pull
requests.

`wagl` runs inside a container in the Swarm cluster (preferably on manager
nodes) and is easy to deploy.


### [User Guide](_docs/0-Index.md)

0. [**wagl Command-Line Interface**](_docs/1-Command-Line.md)
0. [**Deploying wagl**](_docs/2-Deploying.md)
0. [**Service Naming for DNS**](_docs/3-Service-Naming.md)
0. [**DNS Forwarding for External Domains**](_docs/4-External-DNS.md)
0. [**Best Practices**](_docs/5-Best-Practices.md)

### Demo

Watch the demo at: https://www.youtube.com/watch?v=H7dr6lZqw6I
[![](http://cl.ly/image/330U0j280J27/Image%202015-10-15%20at%201.03.49%20PM.png)](https://www.youtube.com/watch?v=H7dr6lZqw6I)

### 5-Minute Tutorial

Let's create a Docker Swarm cluster with [`docker-machine`][machine] and deploy
a `wagl` container to serve as a DNS server to this cluster:

**Step 0:** Download docker client and docker-swarm on your machine.

**Step 1:** Obtain a Swarm discovery token:

```sh
$ docker run --rm swarm create
9746027c20071fdabf9347203fc380fa 
```

**Step 2:** Create a single master and 3-node Swarm cluster with docker-machine

```sh
TOKEN=9746027c20071fdabf9347203fc380fa # <-- paste your token
docker-machine create -d virtualbox --swarm --swarm-master --swarm-discovery token://$TOKEN swarm-m && \
  for i in {0..2}; do docker-machine create -d virtualbox --swarm --swarm-discovery token://$TOKEN swarm-$i; done
```

**Step 3:** Deploy the `wagl` DNS container to the Swarm master node:

```sh
docker-machine ssh swarm-m
```

and then run:

```sh
docker run -d --restart=always  \
    --link=swarm-agent-master:swarm \
    -v /var/lib/boot2docker/ca.pem:/certs/ca.pem \
    -v /var/lib/boot2docker/server.pem:/certs/cert.pem \
    -v /var/lib/boot2docker/server-key.pem:/certs/key.pem \
    -p 53:53/udp \
    --name=dns \
    ahmet/wagl \
      wagl --swarm tcp://swarm:3376 \
      --swarm-cert-path /certs
```

The following command deploys a `wagl` container (named `dns`) pointing it to a
“Swarm manager” running on the same node on `:3376` and starts listening for DNS
queries on port 53.

After the container is working (verify with `docker ps`), exit the SSH prompt.

**Step 4:** Schedule some web server containers on your cluster.

Pay attention to how we use Docker labels (`-l` argument) to name our services:

```sh
$ eval $(docker-machine env --swarm swarm-m)
$ docker run -d -l dns.service=blog -p 80:80 nginx
$ docker run -d -l dns.service=api -l dns.domain=billing -p 80:80 nginx
$ docker run -d -l dns.service=api -l dns.domain=billing -p 80:80 nginx
```

**Step 5:** Verify the DNS works! Start a container in the cluster
with `--dns` argument as IP address where the `dns` container running (in this
case, master node) and make a request for `http://blog.swarm`:

```sh
$ master=$(docker-machine ip swarm-m)
$ docker run -it --dns $master busybox
/ # wget -qO- http://blog.swarm
Connecting to blog.swarm (192.168.99.101:80)
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>...
```
Let's quit this container and launch a Debian container in the cluster to make DNS lookups
to these `api` containers for A/SRV records:

```sh
$ docker run -it --dns $master debian
/# apt-get -q update && apt-get -qqy install dnsutils
...

/# dig +short A api.billing.swarm
192.168.99.103
192.168.99.102

/# dig +short SRV _api._tcp.billing.swarm
1 1 80 192.168.99.102.
1 1 80 192.168.99.103.
```

As you can notice the IP addresses are returned in random order for very naive
load-balancing via the DNS records.

**This is `wagl` in a nutshell. Play and experiment with it!**



### Not Implemented Properties

Some features are not implemented for the sake of minimalism. Please be aware of
these before using.

* DNSSEC
* IPv6 records (such as type AAAA)
* Not-so-needed record types (NS, SOA, MX etc)
* HTTP REST API to query records
* Proper and configurable DNS message exchange timeouts
* DNS over TCP: currently we only do UDP, I have no idea what happens to large
  DNS queries or answers.
* Recursion on external nameservers: We just randomly pick an external NS to
  forward the request and if that fails we don't try others, we just call it
  failed.
* Staleness checks are fragile to system clock changes because Go language does
  not have monotonically increasing clock implementation.

For the not implemented features, we return `NOTIMP` status code in DNS answers
and any server failure returns `SERVFAIL` status code.



### Authors

* [Ahmet Alp Balkan](http://www.ahmetalpbalkan.com/)



### License

This project is licensed under Apache License Version 2.0. Please refer to
[LICENSE](LICENSE).



### Disclaimer

This project is affiliated neither with Microsoft Corporation nor Docker Inc.



#### Why the name?

It turns out the scientists obvserved that the honeybees coming back from a food
source to the bee hive, they tended to waggle about excitedly in a figure 8
pattern which **shares the location of the food source** with other bees. This
is called **“The Waggle Dance”**. It is actually pretty amazing, you should just
[watch the video][waggle-dance].

[![](http://cl.ly/image/1b3B3q410e0z/Image%202015-08-27%20at%204.01.12%20PM.png)][waggle-dance]

[sw]: https://github.com/docker/swarm
[machine]: https://github.com/docker/machine
[waggle-dance]: https://www.youtube.com/watch?v=bFDGPgXtK-U
