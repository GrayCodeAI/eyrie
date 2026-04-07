# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1.0 | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in eyrie, please report it responsibly.

### How to Report

**Please do not create public GitHub issues for security vulnerabilities.**

Instead, please report security issues via:

- Email: security@graycode.ai
- Or through GitHub's [private vulnerability reporting](https://github.com/GrayCodeAI/eyrie/security/advisories/new)

### What to Include

When reporting a vulnerability, please include:

1. **Description**: Clear description of the vulnerability
2. **Impact**: What could an attacker do with this?
3. **Reproduction**: Steps to reproduce the issue
4. **Environment**: Version, Node.js version, OS
5. **Suggestions**: Any ideas for mitigation (optional)

### Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 1 week
- **Fix Released**: As soon as possible depending on severity

### Security Best Practices

When using eyrie:

1. **Keep dependencies updated**: Regularly update to latest versions
2. **Validate inputs**: Always validate inputs before passing to eyrie functions
3. **Secure API keys**: Never commit API keys to version control
4. **Use HTTPS**: Always use HTTPS for API endpoints
5. **Monitor logs**: Be aware of what information is being logged

## Disclosure Policy

We follow responsible disclosure:

1. Reporter submits vulnerability privately
2. We acknowledge and assess the vulnerability
3. We work on a fix
4. We release the fix
5. We publicly disclose the vulnerability (with reporter's permission)

## Security-related Configuration

### API Keys

eyrie handles API keys for LLM providers. Ensure:

- API keys are stored securely (environment variables, not hardcoded)
- Keys have minimal required permissions
- Keys are rotated regularly

### Dependencies

eyrie has minimal dependencies to reduce attack surface:

- `@graycode-ai/sdk`: Official SDK
- `zod`: Schema validation

Regular security audits are performed on these dependencies.

## Contact

For security questions or concerns:

- Security Team: security@graycode.ai
- General Issues: [GitHub Issues](https://github.com/GrayCodeAI/eyrie/issues) (for non-security issues)

Thank you for helping keep eyrie secure!
