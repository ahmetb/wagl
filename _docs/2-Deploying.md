# Deploying and Running wagl

Recommended way to deploy wagl is to use its Docker image
(`ahmet/wagl`) and **run it in a container**.

Also, it is easier to deploy `wagl` to the machines where the Swarm managers are
running at (co-locating). Rest of this guide will assume you will be practicing
as such.

If your Docker Swarm cluster is running without TLS authentication, deploying
and running `wagl` is as easy as the following:

```sh
$ docker run -d --restart=always  \
    --link=<swarm-manager-container>:swarm \
    -p 53:53/udp \
    --name=dns \
    ahmet/wagl \
      --swarm tcp://swarm:3376 \
```

In the example above, replace `<swarm-manager-container>` with the name of your
Swarm manager and then by using linking `wagl` can now talk to the Swarm manager
and query the containers to create DNS records.


### Using TLS

If your Swarm manager is secured with TLS certificates, `wagl` needs to access
these certificates. In order to do that, you can mount the certs into the
container by using Docker Volumes feature and specify `--swarm-cert-path`
argument. This assumes the specified directory has `key.pem`, `cert.pem`,
`ca.pem` files:

For instance when you set up a cluster with `docker-machine` the configuration
looks like the following:

```sh
$ docker run -d --restart=always  \
    --link=swarm-agent-master:swarm \
    -v /var/lib/boot2docker/ca.pem:/certs/ca.pem \
    -v /var/lib/boot2docker/server.pem:/certs/cert.pem \
    -v /var/lib/boot2docker/server-key.pem:/certs/key.pem \
    -p 53:53/udp \
    --name=dns \
    ahmet/wagl \
      --swarm tcp://swarm:3376 \
      --swarm-cert-path /certs
```

(Replace certificate paths and `swarm-agent-master` name according to your set
up.)