# RiverMeadow Helper VM

The RiverMeadow helper virtual appliance is a troubleshooting tool to help perform environment validation and identify network communication issues.

## Troubleshooting Checks

The helper virtual appliance has been loaded with tools to help with troubleshooting

| Name | Description |
|------|-------------|
| DHCP Check | The helper virtual appliance initiates a DHCP request to validate if there is a DHCP server available on the network that responds to requests |
| Network ICMP Ping | A generic ICMP request that enables basic network connectivity testing for other systems on the network such as gateways or servers|
| Network Connection Test | The helper appliance initiates a TCP connection to the designated target on the ports utilized by the |
| DNS Check | The helper appliance attempts to resolve a hostname provided |
| Gateway Check | |
| RiverMeadow Platform Check | The helper appliance initiates a connection to the RiverMeadow platform to evaluate|
| Network Proxy Check | The helper appliance initiates a connection to the RiverMeadow platform utilizing configured network proxy details |
| SSL Interception Check | The helper appliance initiates a connection to the RiverMeadow platform and evaluates if the response indicates that SSL interception is being performed |

## Security Configuration
The helper virtual appliance

### Security Hardening

The following security hardening is applied to the helper virtual appliance to align with security best practices:

| Name | Description |
|------|-------------|
| No SSH packages | There are no SSH packages installed on the virtual appliance |
| All ingress network traffic dropped by default | All ingress network traffic to the virtual appliance is denied by the host firewall |
| Egresss network traffic restricted | Only the designated egress network ports and protocols are allowed by the host firewall |
| All unnecessary packages are removed | The virtual appliance contains a limited number for system packages and are only those that are required |
| Limited services are running | The helper virtual appliance runs only the required services |
| Ephemeral deployment | The helper virtual appliance utilizes an ephemeral |

### Egress Network Ports and Protocols

The following network ports and protocols are allowed for egress network traffic from the helper virtual appliance.

| Name | Description | Port | Protocol |
|------|-------------|:------:|:----------:|
| Source Data Transfer | The RiverMeadow migration utility and source worker appliance  | 5994 | TCP |
| | (Migration Appliance, RiverMeadow Platform, Target Platform REST API) | 443 | TCP |
| Vmware ESXi | | 8080 | TCP |
| | | 8888 | TCP |
| Source Linux Migration Utility Deployment | The automated deployment of the RiverMeadow migration utility for Linux systems utilizes SSH. | 22 | TCP |
| Source Windows Migration Utility Deployment | The automated deployment of the RiverMeadow migration utility for Windows systems utilizes SMB. | 445 | TCP |
| Source Windows Migration Utility Deployment | The automated deployment of the RiverMeadow migration utility for Windows systems utilizes WinRM. | 5985 | TCP |
| Vmware ESXi | | 902 | TCP |

### Software Bill of Materials
A software bill of materials is generated with every build that


| Name | Description |
|------|--------------|
| Golang Binary | The |
| Apline OS | The helper virtual appliance is built on Alpine Linux and the installed packages and their versions |

### Vulnerability Scanning

The virtual appliance is scanned for vulnerabilities during the 

### Patch Management

The virtual appliance is ephemeral and stateless by design for improved security. Updates to the troubleshooting utility and system packages are managed by downloading the latest version of the of the virtual appliance ISO file.


## Supported Target Platforms

* HPE Morpheus VM Essentials
* Microsoft Hyper-V


sh aports/scripts/mkimage.sh --tag v3.22 --arch x86_64 --outdir /iso --repository http://10.0.0.85/v3.22/main --repository http://1
0.0.0.85/v3.22/community --profile helper_kvm