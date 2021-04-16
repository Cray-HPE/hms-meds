module stash.us.cray.com/HMS/hms-meds

go 1.13

replace (
	stash.us.cray.com/HMS/hms-base => stash.us.cray.com/HMS/hms-base v1.12.0
	stash.us.cray.com/HMS/hms-certs => stash.us.cray.com/HMS/hms-certs v1.2.2
	stash.us.cray.com/HMS/hms-compcredentials => stash.us.cray.com/HMS/hms-compcredentials v1.10.0
	stash.us.cray.com/HMS/hms-securestorage => stash.us.cray.com/HMS/hms-securestorage v1.11.0
	stash.us.cray.com/HMS/hms-sls => stash.us.cray.com/HMS/hms-sls v1.8.1
	stash.us.cray.com/HMS/hms-smd => stash.us.cray.com/HMS/hms-smd v1.28.7
)

require (
	github.com/hashicorp/go-retryablehttp v0.6.7
	github.com/mitchellh/mapstructure v1.3.0
	gopkg.in/square/go-jose.v2 v2.4.1 // indirect
	stash.us.cray.com/HMS/hms-base v1.12.0
	stash.us.cray.com/HMS/hms-bmc-networkprotocol v1.4.2
	stash.us.cray.com/HMS/hms-certs v1.2.6
	stash.us.cray.com/HMS/hms-compcredentials v1.10.0
	stash.us.cray.com/HMS/hms-dns-dhcp v1.4.3
	stash.us.cray.com/HMS/hms-securestorage v1.11.0
	stash.us.cray.com/HMS/hms-sls v1.5.1
	stash.us.cray.com/HMS/hms-smd v1.28.0
)
