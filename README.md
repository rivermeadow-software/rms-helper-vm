# RiverMeadow Helper VM

The RiverMeadow helper virtual appliance is a troubleshooting tool to help perform environment validation and identify network communication issues.

![](./docs/system_info_pane.png)

## Profiles
The helper virtual appliance includes several testing profiles for performing a suite or collection of tests that are used to validate common issues with the RiverMeadow solution.

![](./docs/troubleshooting_pane.png)


| Name | Description | Checks  |
|------|-------------|---------|
| RiverMeadow Platform | The RiverMeadow workload mobility platform | DNS Resolution, Port Test (443), SSL Interception |
| Migration Appliance | The RiverMeadow migration appliance deployed to the target environment | Port Test (10000, 443, 8888) |
| Source Worker Appliance | The RiverMeadow source worker appliance deployed to the source VMware vSphere environment for VM based migrations | Port Test (5994) |
| Source Server | The source server for OS based migrations | Port Test (5994) |

## Troubleshooting Checks

The helper virtual appliance has been loaded with tools to help with troubleshooting network connectivity between the components of the RiverMeadow solution.

| Name | Description |
|------|-------------|
| DNS Check | The helper appliance attempts to resolve a hostname using the configured DNS servers(s) |
| DHCP Check | The helper virtual appliance initiates a DHCP request to validate if there is a DHCP server available on the network that responds to requests |
| Network ICMP Ping | A generic ICMP request that enables basic network connectivity testing for other systems on the network such as gateways or servers|
| Network Connection Test | The helper appliance initiates a TCP connection to the designated target on the designated port |
| SSL Interception Check | The helper appliance initiates a connection to the RiverMeadow platform and evaluates if the response indicates that SSL interception is being performed |
| Network Proxy Check (Roadmap) | The helper appliance initiates a connection to the RiverMeadow platform utilizing configured network proxy details |
| RiverMeadow Platform Check | The helper appliance initiates a connection to the RiverMeadow platform to evaluate|

## Security Configuration
The helper virtual appliance has been hardened to align with the limited use case to ensure that it is lightweight and it aligns with security best practices.

### Security Hardening

The following security hardening is applied to the helper virtual appliance to align with security best practices:

| Name | Description |
|------|-------------|
| No SSH packages | There are no SSH packages installed on the virtual appliance |
| All ingress network traffic dropped by default | All ingress network traffic to the virtual appliance is denied by the host firewall |
| Egresss network traffic restricted | Only the designated egress network ports and protocols are allowed by the host firewall |
| All unnecessary packages are removed | The virtual appliance contains a limited number for system packages and are only those that are required |
| Limited services are running | The helper virtual appliance runs only the required services |
| Ephemeral deployment | The helper virtual appliance is stateless and utilizes an ephemeral operating model |
| Restricted program access | The helper virtual appliance only runs the troubleshooting utility and only allows access to the troubleshooting utility |

### Egress Network Ports and Protocols

The following network ports and protocols are allowed for egress network traffic from the helper virtual appliance:

| Name | Description | Destination | Port | Protocol |
|------|-------------|------|:------:|:----------:|
| Source Data Transfer | The RiverMeadow migration utility and source worker appliance  | Source Server, Source Worker Appliance | 5994 | TCP |
| | | Migration Appliance, RiverMeadow Platform, Target Platform REST API | 443 | TCP |
| | | 8080 | TCP |
| | | 8888 | TCP |
| Source Linux Migration Utility Deployment | The automated deployment of the RiverMeadow migration utility for Linux systems utilizes SSH. | 22 | TCP |
| Source Windows Migration Utility Deployment | The automated deployment of the RiverMeadow migration utility for Windows systems utilizes SMB. | 445 | TCP |
| Source Windows Migration Utility Deployment | The automated deployment of the RiverMeadow migration utility for Windows systems utilizes WinRM. | 5985 | TCP |

### Ingress Network Port and Protocols

The following network ports and protocols are allowed for ingress network traffic traffic to the source helper virtual appliance:

| Name | Description | Destination | Port | Protocol |
|------|-------------|------|:------:|:----------:|
| N/A  | No ingress network ports are allowed | N/A | N/A | N/A |


### Software Bill of Materials
A software bill of materials is generated with every build that

| Name | Description |
|------|--------------|
| Golang Binary | The |
| Apline Operating System | The helper virtual appliance is built on Alpine Linux and the installed packages and their versions |

### Patch Management

The virtual appliance is ephemeral and stateless by design for improved security. Updates to the troubleshooting utility and system packages are managed by downloading the latest version of the of the virtual appliance ISO file.

## Supported Target Platforms

The helper appliance virtual machine has been tested on the following platforms but should work on any virtualization platform that supports Alpine Linux.

* HPE Morpheus VM Essentials
* Microsoft Hyper-V
* Nutanix AHV