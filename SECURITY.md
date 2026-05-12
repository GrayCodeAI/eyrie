# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes |

## Reporting a Vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Email: security@graycode.ai

### Response Timeline
- Acknowledgment: 48 hours
- Initial assessment: 5 business days
- Fix: 7-30 days depending on severity

## Security Design

- Zero external dependencies (only Go stdlib)
- All provider communication over HTTPS (TLS required)
- API keys never logged or included in error messages
- Error response bodies capped at 4KB to prevent OOM
- No `InsecureSkipVerify` usage anywhere
- Rate limiting prevents API key exhaustion
