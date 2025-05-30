# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.25.0] - 2025-05-02

### Updated

- Updated module dependencies
- Use hms-base for draining and closing request and response bodies
- Fixed bug with jq use in runSnyk.sh
- Internal tracking ticket: CASMHMS-6395

## [1.24.0] - 2025-04-04

### Updated

- Added support for pprof builds
- Updated image and module dependencies
- Updated from Go 1.23 to 1.24 and fixed build warnings due to that
- Internal tracking ticket: CASMHMS-6361

## [1.23.0] - 2025-03-07

### Changed

- Bump the app version by one

## [1.22.0] - 2025-03-07

### Security

- Updated image and module dependencies
- Various code changes to accomodate module updates
- Resolved build warnings in Dockerfiles and docker compose files
- Fixed failing unit test related to chassis endpoints

## [1.21.0] - 2024-12-03

### Changed

- Updated go to 1.23

## [1.20.0] - 2022-07-12

### Changed

- CASMHMS-5567: Allow MEDS to support variable number of Chassis within a Cabinet. Instead of always assuming 8 chassis in a cabinet MEDS will now look to SLS for the chassis present in the system.

## [1.19.0] - 2022-07-06

### Changed

- Change HSM v1 API references to v2

## [1.18.0] - 2022-03-07

### Removed

- Removed the CT test rpm.  

### Changed

- Converted to build in Github actions.

## [1.17.0] - 2021-10-27

### Added

- CASMHMS-5055 - Added MEDS CT test RPM.

## [1.16.8] - 2021-09-21

### Changed

- Changed cray-service version to ~6.0.0

## [1.16.7] - 2021-09-08

### Changed

- Changed docker image to run as the user nobody

## [1.16.6] - 2021-09-07

### Changed

- Fixed bug in syslog/NTP use of IP addrs.

## [1.16.5] - 2021-08-10

### Changed

- Added GitHub configuration files.

## [1.16.4] - 2021-07-27

### Changed

- Github migration phase 3.

## [1.16.3] - 2021-07-20

### Changed

- Add support for building within the CSM Jenkins.

## [1.16.2] - 2021-07-12

### Security
- CASMHMS-4933 - Updated base container images for security updates.

## [1.16.1] - 2021-07-01

### Changed
- CASMHMS-4928 - When MEDS initializes a cabinet it will verify that Redfish Endpoints for ChassisBMCs have the correct FQDN and Hostname set. If the FQDN and Hostname do not have a `b0` suffix MEDS will update the RedfishEndpoint in HSM to have it.   

## [1.16.0] - 2021-06-18

### Changed
- Bump minor version for CSM 1.2 release branch

## [1.15.0] - 2021-06-18

### Changed
- Bump minor version for CSM 1.1 release branch

## [1.14.8] - 2021-05-04

### Changed
- Updated docker-compose files to pull images from Artifactory instead of DTR.

## [1.14.7] - 2021-05-03

### Changed
- CASMHMS-4806 - MEDS is now smarter about adding/patching EthernetInterfaces into HSM. It will now only make POST and PATCH requests to HSM when there is actually something to change.

## [1.14.6] - 2021-04-26

### Changed
- CASMHMS-4798 - Internally do not mark a Redfish Endpoint as not present in HSM. This will prevent MEDS from triggering a rediscovery on BMC due to network hiccups or hardware replacement. 

## [1.14.5] - 2021-04-14

### Changed
- CASMHMS-4715 - Modified MEDS to only push the global default BMC credentials into vault when the BMC credentials do not exist in vault.

## [1.14.4] - 2021-04-14

### Changed
- CASMHMS-4660 - Fixed HTTP response body leaks.

## [1.14.3] - 2021-03-31

### Changed

- CASMHMS-4605 - Update the loftsman/docker-kubectl image to use a production version.

## [1.14.2] - 2021-03-29

### Changed

- Removed hack to allow CMMs to not have 'b0' names.

## [1.14.1] - 2021-2-26

### Changed

- MEDS now has options for syslog and NTP server hostnames,  settable in customizations.yaml.

## [1.14.0] - 2021-2-09

### Changed

- MEDS no longer expects specific IP addresses for nodes and instead looks them up using DNS.

## [1.13.1] - 2021-02-09

### Changed

- Added User-Agent headers to outbound HTTP requests.

## [1.13.0] - 2021-02-05

### Changed

- updated vendors and licenses

## [1.12.2] - 2021-01-22

### Changed

- CASMHMS-4367 - MEDS no longer marks redfishEndpoints as disabled in HSM when they are no longer on the network, as this is not necessary.

## [1.12.1] - 2021-01-19

### Changed

- CASMINST-960 - Use `rsyslog-aggregator.hmnlb` as the host the syslog aggregator, as the host `rsyslog_agg_service_hmn.local` is no longer available in Shasta v1.4 or later.

## [1.12.0] - 2021-01-14

### Changed

- Updated license file.


## [1.11.4] - 2020-11-25

- Updated curl image used by the Helm chart.

## [1.11.3] - 2020-11-24

- CASMINST-368 - Added checks in the Helm chart to prevent CLBO due to NTP and syslog hostnames not resolving.

## [1.11.2] - 2020-11-13

- CASMHMS-4217 - Added final CA bundle configmap handling to Helm chart.

## [1.11.1] - 2020-10-21

- CASMHMS-4105 - Updated base Golang Alpine image to resolve libcrypto vulnerability.

## [1.11.0] - 2020-10-15

- Added TLS cert support for Redfish operations.

## [1.10.2] - 2020-10-07

- CASMHMS-1373 - Randomized start delay in checkup threads to avoid thundering herd issues.

## [1.10.1] - 2020-10-01

- CASMHMS-4065 - Update base image to alpine-3.12.

## [1.10.0] - 2020-09-15

- Change MEDS to do lookups via DNS for nodes, in addition to pinging derived IPs.

## [1.9.0] - 2020-09-15

- Picked up various chart updates, see CASMCLOUD-1023

## [1.8.2] - 2020-09-03

- Ensure MEDS does not add hardware with empty username/password to HSM.

## [1.8.1] - 2020-06-30

- Fixed xname bug with CMMs.

## [1.8.0] - 2020-06-29

- CASMHMS-3610 - Added CT smoke test for MEDS.

## [1.7.1] - 2020-06-29

- Made HSM inform use xname, not IP.

## [1.7.0] - 2020-06-12

- Added logic for MEDS to add all precomputed hardware from SLS into the HSM EthernetInterfaces API for use by DHCP.

## [1.6.1] - 2020-06-11

- Removed code for handling some arguments; removed (most) direct configuration via configmap

## [1.6.0] - 2020-05-27

- Added polling SLS for cabinets

## [1.5.5] - 2020-05-05

- Enabled online upgrade/downgrade of helm charts.

## [1.5.4] - 2020-05-12

- Now uses default mac prefix if SLS data doesn't have it.

## [1.5.3] - 2020-05-04

- CASMHMS-2961 - change to use tusted baseOS image in docker build.

## [1.5.2] - 2020-04-24

- Fixed a change to the sls loader check.

## [1.5.1] - 2020-04-06

- Allow empty IP6Prefix from SLS.

## [1.5.0] - 2020-04-06

- Added looking up Hill Cabinets in addition to Mountain cabinets.

## [1.4.0] - 2020-03-23

- Fixed incorrect service account for deployment preventing wait-for container from starting.

## [1.3.0] - 2020-03-23

- Added wait for on SLS loader.

## [1.2.11] - 2020-03-10

- Reduced the volume of trace messages when pinging for hardware.

## [1.2.10] - 2020-03-09

- CASMHMS-2730 - Refactored to use the new base packages instead of hms-common.

## [1.2.9] - 2020-03-06

- Added flag for enabling SLS.

## [1.2.8] - 2020-03-05

- Now uses the updated hms-bmc-networkprotocol library to fix JSON rendering issues.

## [1.2.7] - 2020-03-02

### Fixed

- CASMHMS-3012 - use the service name instead of a static IP address for remote logging.

## [1.2.6] - 2020-02-12

### Fixed

- CASMHMS-2947 - fixed the syslog forwarding info to use the correct service name.

## [1.2.5] - 2020-02-13

### Changed

- MEDS now uses PATCHs instead of PUTs to modify existing redfish endpoints in HSM.
- MEDS now triggers HSM rediscovery with the PATCH calls instead of POSTs to /Inventory/Discover

## [1.2.4] - 2019-12-18

### Fixed

- fixed the syslog forwarding info to include the missing port number.

## [1.2.3] - 2019-12-16

### Changed

- Moved call to HSM to do discover to **AFTER** the storage of the credentials in Vault.
- Defaulted the Vault connection to prepend `secret` to the credentials path.

## [1.2.2] - 2019-12-12

### Changed

- Updated hms-common lib.

## [1.2.1] - 2019-12-10

### Changed

- MEDS now resyncs its knowledge of RedfishEndpoints with HSM periodically.

## [1.2.0] - 2019-11-27

### Added

- Added loader to transfer credentials from secret to Vault. This loader is part of the primary MEDS image.

### Changed

- Moved credentials code to internal location.
- Revamped Docker image build and testing scripts.

## [1.1.6] - 2019-11-08

### Removed

- Removed generation of the ConfigMap for the options from this deployment and moved them into the installer where they can be programmatically generated based on a configuration file.

## [1.1.5] - 2019-11-04

### Fixed

- Fixed issue preventing credentials from getting placed into Vault in a location HSM could read them.

## [1.1.4] - 2019-10-29

### Changed

- Moved options for runtime from the deployment to a ConfigMap.

## [1.1.3] - 2019-10-29

### Changed

- Now gets cab info from SLS if --sls=xxxx is specified.  If not the old way is used.

## [1.1.1] - 2019-10-08

### Changed

- Now transmits SSH keys to mountain controllers along with syslog and NTP info.

## [1.1.0] - 2019-10-03

### Changed

- Read global credentials from Vault, but use defaults if not available

## [1.0.2] - 2019-09-06

### Changed

- MEDS now gets syslog and NTP server info from k8s and sets this info on mountain controllers when they are discovered.

## [1.0.1] - 2019-08-16

### Changed

- MEDS now reuses a single http.Client for all conenctions.  This should allow conenction reuse and reduce network load.

## [1.0.0] - 2019-08-07

### Added

- This is the initial versioned release. It contains everything that was in `hms-services` at the time with the major exception of being `go mod` based now.

- Added the ability to set DNS and DHCP entries via the SMNetManager service.

### Changed

### Deprecated

### Removed

### Fixed

### Security
