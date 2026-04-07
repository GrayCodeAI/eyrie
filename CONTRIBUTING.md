# Contributing to eyrie

Thank you for your interest in contributing to eyrie! This document provides guidelines for contributing to this project.

## Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/GrayCodeAI/eyrie.git
   cd eyrie
   ```

2. **Install dependencies**
   ```bash
   npm install
   ```

3. **Build the project**
   ```bash
   npm run build
   ```

4. **Run type checking**
   ```bash
   npm run typecheck
   ```

## Project Structure

```
eyrie/
├── src/
│   ├── config/        # Provider configuration
│   ├── constants/     # API limits and constants
│   ├── types/         # TypeScript type definitions
│   └── index.ts       # Main exports
├── dist/              # Compiled output (generated)
└── package.json
```

## Coding Standards

- **TypeScript**: All code must be written in TypeScript
- **ES Modules**: Use ES module syntax (`import`/`export`)
- **Strict Mode**: Enable strict TypeScript checking
- **No UI Dependencies**: eyrie must remain UI-framework agnostic
- **Node.js Built-ins Only**: Only use Node.js built-in modules (fs, os, path, etc.)

## Commit Message Guidelines

We follow conventional commits:

- `feat:` New features
- `fix:` Bug fixes
- `docs:` Documentation changes
- `style:` Code style changes (formatting, etc.)
- `refactor:` Code refactoring
- `test:` Adding or updating tests
- `chore:` Maintenance tasks

Example:
```
feat: add support for new provider

Add support for the XYZ LLM provider with proper
type definitions and configuration options.
```

## Pull Request Process

1. **Create a branch** for your changes
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards

3. **Test your changes**
   ```bash
   npm run typecheck
   npm run build
   ```

4. **Commit your changes** with a clear commit message

5. **Push to your fork** and create a Pull Request

## Reporting Issues

When reporting issues, please include:

- **Description**: Clear description of the issue
- **Steps to Reproduce**: How to reproduce the problem
- **Expected Behavior**: What you expected to happen
- **Actual Behavior**: What actually happened
- **Environment**: Node.js version, OS, etc.

## Code of Conduct

This project adheres to a code of conduct. By participating, you are expected to uphold this code:

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Respect different viewpoints and experiences

## Questions?

If you have questions, please:

1. Check the [documentation](https://github.com/GrayCodeAI/eyrie#readme)
2. Search existing [issues](https://github.com/GrayCodeAI/eyrie/issues)
3. Create a new issue with the "question" label

## License

By contributing to eyrie, you agree that your contributions will be licensed under the MIT License.
