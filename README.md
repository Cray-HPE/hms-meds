# Mountain Endpoint Discovery Service

The Mountain Endpoint Discovery Service is responsible for endpoint discovery, initialization and geolocation of hardware in Mountain cabinets.

## The Tasks

* Endpoint Discovery -- The process of locating a previously-unknown Redfish endpoint on the network
* Initialization -- The process of configuring a previously-unknown Redfish endpoint to work within the Cray.  This may involve setting credentials or running other configuration tasks
* Geolocation -- The process of determining the physical location of the new Redfish endpoint and assigning it an xname

## How it works

MEDS is currently a very simple service.  It it configured with some basic information that allows it to determine which IP addresses should have Redfish endpoints attached.  This information includes the rack numbers present in a system, as well as the IPv4 range assigned to each rack.  In short, this information allows MEDS to determine that (for example) xname x0c0s0b0 should be at IP address 10.4.0.9.

Once MEDS has determined which IP addresses should have endpoints and the xname associated with each IP address, it determines the state of the existing system by querying Hardware State Manager.  The HSM queries determine the state of each piece of MEDS hardware as either "present" (set up in HSM and enabled) or "not present" (not set up in HSM or disabbled in HSM).

Next, MEDS begins performing "Redfish pings" of each IP address to determine their real world state.  During a Redfish ping, MEDS attempts to make an HTTPS request to each endpoint.  If the request returns a valid HTTP status code (including error codes, such as 4xx and 5xx codes), MEDS considers the endpoint to be present.

Finally, MEDS compares the current state to the state it recieved from HSM.  If the state is different than HSM, it notifies HSM of the change.  If a node was not present, MEDS tries to enable or create it in HSM.  If the node was present, MEDS disables it in HSM.

MEDS then begins again at the Redfish ping step.

## Configuration

MEDS should be configured via ansible.  By default MEDS configuration is found in `/opt/cray/crayctl/ansible_framework/roles/cray_meds/defaults/main.yml`, though these variables may be overridden from elsewhere.  Configuration consists of two main items:

* IPv6 prefix configuration
* Per-rack configuration (including IPv4 configuration)

To perform IPv6 configuration, simply set `cray_meds_ip_prefix` to the IPv6 prefix used in the system.  You must also perform per-rack configuration.

To perform per-rack configuration is more complicated.  This requires setting the `cray_meds_racks` variable.  This variable is a list of objects.  Each object must have the `number` key and may have the `ipv4net` key.  Number is the non-negative number representing the rack, .e.g: `3000`.  ipv4net is the first IPv4 address assigned to this rack, with the associated netmask (e.g.: `10.100.106.2/23`) and may be omitted if ipv4 is not in use.  Note that this is NOT the same as the IP subnet assigned to the rack and attempting to use the IP subnet will result in misidentification of hardware.

The general form of the `cray_meds_racks` variable (with example values) is:

```
cray_meds_racks: [
    {
        "number": 3000,
        "ip4net": "10.100.106.2/23"
    },
    ...
]
```

## Future work

This is a list of work that is either known to be coming or that should get done "in the future" (ie: technical debt) or that is left here as a breadcrumb or idea for future improvements.

* Get away from staticly configured IPv4 addresses.  Perhaps we can tie into the DHCP server somehow to see what things come up?  We want to avoid doing ping sweeps.
* Use IPv6 addresses (blocked; waiting for IPv6 to be enabled in the Kubernetes stack)
* Set up consoles to be network accessible during the initialization phase

## Testing with Docker Compose

A `docker-compose` environment can be set up by running:

`docker-compose -f docker-compose.developer.full.yaml up -d --build`

and torn down with:

`docker-compose -f docker-compose.developer.full.yaml down --remove-orphans`

This will set up an environment with one dummy endpoint, named `x1000c1s1b0`.  It will *not* automatically be in SLS, you will have to add it using a command like:

`curl  -d '{"Xname":"x1000","Type":"comptype_cabinet","Class":"Mountain","TypeString":"Cabinet","ExtraProperties":{"Networks":{"cn":{"HMN":{"CIDR":"172.26.0.1/22","Gateway":"172.26.0.254","VLan":3001},"NMN":{"CIDR":"10.100.0.1/22","Gateway":"10.100.3.254","VLan":2001}},"ncn":{}}}}' http://cray-sls:8376/v1/hardware`

Note that on the mac, docker containers do not resolve, so use `localhost` as the hostname to contact.
