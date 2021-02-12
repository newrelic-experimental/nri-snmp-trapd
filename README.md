
[![New Relic Experimental header](https://github.com/newrelic/opensource-website/raw/master/src/images/categories/Experimental.png)](https://opensource.newrelic.com/oss-category/#new-relic-experimental)

# New Relic integration for SNMP traps

New Relic Integration for SNMP Traps is a service that listens for SNMP traps and forwards them to New Relic

## Installation

* Copy config.yml.sample to config.yml and edit it to update the following properties
	* account_id
	* insert_key
	* nr_region (specify either "EU" or "US" datacenter)
	* http_proxy_host (uncomment and specify if using HTTP proxy)
	* http_proxy_port (uncomment and specify if using HTTP proxy)
	* event_type (default NRQL event_type, can be overriden in traps.yml)
	* snmp_device (default  device name for use in NRQL, can be overriden in traps.yml)
	* snmp_host (hostname to use by this service listener, typically just localhost)
	* snmp_port (port on which this service should start and listen for traps)
	* community (SNMP version 2 community string)
	* v3 (SNMP version, default: false)		
	* trap_definition_files (one or more comma separated list of traps.yml files)
If v3 is true, SNMP v3 connection is attempted and it requires the following additional properties
	* username (the security name that identifies the SNMPv3 user)
	* auth_protocol (the algorithm used for SNMPv3 authentication (SHA or MD5))
	* auth_passphrase (the password used to generate the key used for SNMPv3 authentication)
	* priv_protocol (the algorithm used for SNMPv3 message integrity)
	* priv_passphrase (the password used to generate the key used to verify SNMPv3 message integrity (AES or DES))

* Copy traps.yml.sample to traps.yml and enter or one or more trap definitions in the **traps** section. Each trap definition has the following properties
	* name (any descriptive name for this trap type)
	* type (must be "trap")
	* event_type (NRQL event type)
	* trap_oid (a unique object identifier (OID) number defined for this trap type as specified in the MIB)
	* metrics (a list of attribute OIDs to collect for this trap type as specified in the MIB)
		* metric_name
		* oid


## Getting Started

* Run the service executable from the command line to start this New Relic integration as a SNMP trapd listener

```
nri-trapd -config_file {path-to-config.yml}
```

For troubleshooting also specify the verbose flag

```
nri-trapd -config_file {path-to-config.yml} -verbose
```

## Building



## Support

New Relic has open-sourced this project. This project is provided AS-IS WITHOUT WARRANTY OR DEDICATED SUPPORT. Issues and contributions should be reported to the project here on GitHub.

We encourage you to bring your experiences and questions to the [Explorers Hub](https://discuss.newrelic.com) where our community members collaborate on solutions and new ideas.

## Contributing

We encourage your contributions to improve nri-snmp-trapd! Keep in mind when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project. If you have any questions, or to execute our corporate CLA, required if your contribution is on behalf of a company, please drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [HackerOne](https://hackerone.com/newrelic).

## License

nri-snmp-trapd is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License. 

nri-snmp-trapd also uses source code from third-party libraries. You can find full details on which libraries are used and the terms under which they are licensed in the third-party notices document.

