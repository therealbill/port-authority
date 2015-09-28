# PORT-AUTHORITY

Port-Authority (PA) is a microservice which does one thing: keep and manage a
listing of services to ports. It offers the ability to PUT a service, which
will either allocate a new port and return it, or get the existing port for
said service and return that instead. You can also pull down information about
the current inventory, list of assigned ports, and lis tof available ports. All
over HTTP with simple REST-like calls.

One thing it doens't do is accept port assignments. Thus it is not intended to
be used to generate an /etc/services file. This service was made for use in
Dockerized environments where you may want to dynamically assign ports to
services in containers (such as web or database service containers).

By default it will manage ports 30,000 to 39,999. You can set this in the
consul config backing store, but keep in mind the end value is not inclusive.


# Components

Since we almost assuredly want this data persistent, the service
requires access to a Redis instance.


# Configuration

The prefered way to run this is as a Docker container itself, and using a
backing store, currently just Consul, to store your config in. With that setup
it requires no local config and no command line or environment variables beyond
the connection string for your Consul service. It does, however, take some
optional ones.

## Required Options

The option `-c` or `--consuladdress` requires an argument of the form "ip:port"
and will tell the service what backing store address to connect to. It assumes
`localhost:8500` if not provided. Not truly required but highly recommended as
not all configurables are in commandline/env variables yet.

## Options in Consul

Each instance of PA you run can be named, either via the `--name` option
or the `PA_NAME` environment variable. This is useful for situations
where you need different config values for specific servers. Not all
configurables support this mode. Those that do will be called out
specifically below.

The base KV path used is "app/port-authority/config" all paths
referenced below are from that base. 

The first configurable you'll want to know about is `api_port`. This
value specifies what port to listen on. If not found it defaults to
`8080`.

The next pair tell PA what port to start the pool on and where to end
it. They are found in two possible places. The first place is an
instance-specific config. Because you may want to run dedicated
instances of the service to handle different pools for different
services PA will look under `NAME/ports_begin` for the start of the
pool, and `NAME/ports_end`, for where it ends. 

In this case NAME is the option passed in via the `--name` commandline
argument or the `PA_NAME` environment variable.  If the key isn't there,
or you did not name the instance PA will look in `ports_begin` and
`ports_end` at the base prefix.


# API

## Using Ports


To use a port you add a service. The URL is `/api/service/ID` where ID
is whatever you are calling this particular service. It must be unique,
no exceptions. To use this API call you call the `PUT` HTTP method on
the full URL.

What you will get back is a JSON mapping structure which contains 'status',
'statusmessage', and 'data' keys. The value of the 'data' key is the
port number assigned to this service. Time for an example.

`curl http://localhost:8080/api/service/webapp-cars`

In this example, say we get back the port "32123" (note: the port number
is a random selection of the avialble ports, don't exect a sequential
listing). Now we have a mapping between 'webapp-cars' and port '32123'.
So however we need to assign that port to our webapp, we do so.

Now, what if you repeated the call? From the client-side, nothing is
different. However, PA checks for an existing port for the service ID
provided and will return that if found. Thus, it can be considered
idempotent as it will do the same thing and get the same results -
unless you delete the service mapping in between.

Sometimes you may want to just look for a service, and not assign it a
port. Issue a `GET` to the above URL for that.

## Releasing Ports

Say you're done with the port, maybe you need to decommission that
service. For that, and keeping the example above, you would issue a
`DELETE` method HTTP call to
`http://localhost:8080/api/service/webapp-cars` and it would be removed
from the store in it's entirety.

## Inventory and Reserved Ports

You can check the current inventory and resrved ports via simple calls
as well.

To see how many ports are available: 
`curl http://localhost:8080/api/ports/inventory/count`

To get the list of them:
`curl http://localhost:8080/api/ports/inventory/count`

Now for listing how many ports have been reserved/assigned:
`curl http://localhost:8080/api/ports/assigned/count`

And the listing:
`curl http://localhost:8080/api/ports/assigned/list`

# Redis 

## Keys

PA uses very few keys in Redis, though not all keys are created at
initialization time. As most keys only exists under certain conditions
a freshly initialized atabase will have but one key.

When the service is first started it will connect to the configured
Redis instance and attempt to initialize the database. This means it
will check for the existence of the `open_ports` key first. If not found
it will assume the DB needs initialized.

For initialization a `sorted set` named `open_ports` is created with
each and every port number the PA is allowed to manage added to it. When
new ports are requested PA calls `SPOP` to get a random member.

Once it has it it will then add it to a sorted set named
`assigned_ports`, then add it to a pair of hashes: `i2port` (to map IDs
to ports) and `port2i` to map ports to IDs). As such, once you've
reserved one port all four keys will be created.

When we the last assigned port/service, Redis will delete the now empty
hashes and `assigned_ports` keys. In this we the key count will be very
small. 

## Memory Consumption

Depending on how large you rport range is, this shuold be quite memory
efficient as the two sets are iteger sets, which Redis optimizes for. To
get the maximum benefit the value of `set-max-intset-entries` in Redis'
configuration should be set to at least the number of ports you will
have PA manage. Eventually this will be configurable in PA as well for
cases where you use a dedicated Redis instance (which you should do in
general).

Since we are storing integers for the `id2port` hash, you could tune
`hash-max-ziplist-entries` similarly to the above setting. The
`hash-max-ziplist-value` may be of some value as well if you have a good
understanding of the size of your IDs. Generally this last setting would
mostly be useful, if at all, in cases of very tight memory limitations
such as if your Redis is running on a RaspberryPi (especially with this
service there as well).



## Port-Authority Configuration

Currently it is hard coded to talk to localhost, but that will be
configured in the config store Real Soon Now (tm). When this is added
the paths will likely be `redis/ip`, `redis/port`, and `redis/auth` to
store the IP address, port, and authentication information for Redis
connectivity respectively. Additional Redis settings, when added, will
go in this space as well.



# TODO

 * Add a call to get the full mapping of assigned ports
 * Add configuration support for setting Redis memory settings during
   initialization
 * Write the Web interface portion 
 * Write the 0MQ based RPC
 * Get all configurables in ENV and CLI as well.
 * Finish getting Airbrake support added and documented
 * Perhaps NewRelic support as well?
 * Add in go-metrics stats
 * Add in circuitbreaker for talking to Redis.
 * Add Dockerfile
