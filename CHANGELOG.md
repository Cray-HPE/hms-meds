# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
## [1.14.0] - 2021-2-09

### Changed

- MEDS no longer expects specific IP addresses for nodes and instead looks them up using DNS.

## [1.11.6] - 2021-01-19

### Changed

- CASMINST-960 - Use `rsyslog-aggregator.hmnlb` as the host the syslog aggregator, as the host `rsyslog_agg_service_hmn.local` is no longer available in Shasta v1.4 or later.

## [1.11.5] - 2020-12-10

### Changed
- CASMHMS-4278 - Update to loftsman/docker-kubectl image to production version.

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
