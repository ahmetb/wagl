# DNS Forwarding for External Domains

`wagl` is by default enabled for resolving external domain names (outside
`swarm.` domain), such as `google.com.`

This is done forwarding the DNS query to the external nameservers. If no
external nameserver is configured (with `--ns` argument), then the nameservers
in `/etc/resolv.conf` are automatically picked up.

You can specify different nameservers manually with `--ns` argument, such as:

    $ wagl [...options] --ns 8.8.8.8 --ns 8.8.4.4


### Disabling External Queries

You can entirely disable the external forwarding with `--external=false`
argument. In this case, wagl would return `SERVFAIL` code to external domain
queries.

This can be especially useful if you would like to handle external DNS queries
with a complete and more robust DNS server implementation such as **BIND**. 

### Tips for using with BIND

In order to use BIND with `wagl`, you should start  `wagl` on a different port
such as:

    $ wagl [...options] --external=false --bind=:8053

and then configure BIND as follows for the `swarm.` zone (assuming `wagl` runs
on the specified IP 192.168.0.4):

```
zone "swarm" {
type forward;
forward only;
forwarders { 192.168.0.4 port 8053; }; 
}; 
```