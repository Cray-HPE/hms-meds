#! /usr/bin/env python3

import yaml

def make_emulator(xname, mockup, network):
    return {
        "hostname": xname,
        "image": "artifactory.algol60.net/csm-docker/stable/csm-rie:1.1.0",
        "environment": [
            "MOCKUPFOLDER={}".format(mockup),
            "XNAME={}".format(xname),
            "AUTH_CONFIG=root:root_password:Administrator",
            "PORT=443",
        ],
        "networks": [network]
    }

# Mountain cabinet EX3000/EX4000
network = "meds"
endpoints = {}
for cabinet in ["x1000"]:
    for chassis in ["c0", "c1", "c2", "c3", "c4", "c5", "c6", "c7"]:
        # Chassis BMC
        chassisBMCXname = cabinet+chassis+"b0"
        endpoints[chassisBMCXname] = make_emulator(chassisBMCXname, "CMM", network)

        for slot in ["s0", "s1", "s2","s3","s4", "s5", "s6", "s7"]:
            for nodeBMC in ["b0", "b1"]:
                # NodeBMC
                nodeBMCXname = cabinet+chassis+slot+nodeBMC
                endpoints[nodeBMCXname] = make_emulator(nodeBMCXname, "EX425", network)
        for slot in ["r0", "r1", "r2","r3","r4", "r5", "r6", "r7"]:
            for routerBMC in ["b0"]:
                # RouterBMC
                routerBMCXname = cabinet+chassis+slot+routerBMC
                endpoints[routerBMCXname] = make_emulator(routerBMCXname, "Slingshot_Switch_Blade", network)

# Build docker-compose
docker_compose = {
    "version": "3.7",
    "networks": {
        "meds": {}
    },
    "services": endpoints
}

print(yaml.dump(docker_compose))