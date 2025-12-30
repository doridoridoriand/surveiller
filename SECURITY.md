# Security Policy

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in surveiller, please report it responsibly:

### How to Report

1. **Do not** create a public GitHub issue for security vulnerabilities
2. Send an email to the maintainer with details of the vulnerability
3. Include steps to reproduce the issue if possible
4. Allow reasonable time for the issue to be addressed before public disclosure

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if you have one)

### Response Timeline

- **Initial response**: Within 48 hours
- **Status update**: Within 1 week
- **Fix timeline**: Depends on severity, typically within 2-4 weeks

### Security Considerations

surveiller handles network operations and may require elevated privileges for ICMP operations. Key security considerations:

#### ICMP Privileges
- surveiller attempts to use raw ICMP sockets which require CAP_NET_RAW capability on Linux
- When ICMP privileges are unavailable, it falls back to external `ping` command
- Consider running with minimal required privileges

#### Network Security
- surveiller sends ICMP packets to configured targets
- Ensure target addresses are trusted and expected
- Be aware that ping operations may be logged by network security systems

#### Configuration Security
- Configuration files may contain sensitive network information
- Protect configuration files with appropriate file permissions
- Avoid including sensitive information in configuration comments

#### Metrics Endpoint
- When Prometheus metrics are enabled, they expose network status information
- Secure the metrics endpoint appropriately in production environments
- Consider network-level access controls for the metrics port

## Best Practices

1. **Principle of Least Privilege**: Run surveiller with minimal required permissions
2. **Network Segmentation**: Deploy in appropriate network segments
3. **Configuration Management**: Protect configuration files and use version control
4. **Monitoring**: Monitor surveiller logs for unexpected behavior
5. **Updates**: Keep surveiller updated to the latest version

## Acknowledgments

We appreciate responsible disclosure of security vulnerabilities and will acknowledge contributors (with their permission) in security advisories.