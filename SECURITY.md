# Security Policy

## Supported versions

Only the latest release receives security fixes. Older releases are not
supported.

## Reporting a vulnerability

Please do not open a public issue for a suspected vulnerability. Instead,
report it privately to the maintainer through GitHub's private vulnerability
reporting feature (Security tab of the repository) or by email.

Include, where possible:

- A description of the issue and its impact.
- Steps to reproduce, ideally a minimal example.
- The affected version and operating system.
- Any suggested mitigation.

You will receive an acknowledgement within a few days and a fix or
mitigation plan as soon as practical. Coordinated disclosure is appreciated:
please allow time for a fix before publishing details.

## Scope

filetools walks and modifies the local filesystem. Please report issues such
as path traversal outside the intended directory, symlink following that
escapes the scanned tree, unexpected file deletion or overwrite, and
insecure handling of output paths.