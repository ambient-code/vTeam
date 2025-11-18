# Ambient Code Platform Documentation

This directory contains the complete documentation for the Ambient Code Platform, built with MkDocs and Material theme.

## Quick Start

### View Documentation Locally

```bash
# Install documentation dependencies
pip install -r requirements-docs.txt

# Serve documentation locally
mkdocs serve

# Open in browser
open http://127.0.0.1:8000
```

### Build Static Documentation

```bash
# Build for production
mkdocs build

# Output in site/ directory
ls site/
```

## Documentation Structure

```
docs/
├── index.md                 # Landing page
├── user-guide/
│   ├── index.md             # User guide overview
│   └── getting-started.md   # 5-minute setup guide
├── developer-guide/
│   └── index.md             # Developer overview
├── labs/
│   ├── index.md             # Labs overview
│   └── basic/
│       └── lab-1-first-rfe.md
└── reference/
    ├── index.md             # Reference overview
    └── glossary.md          # Terms and definitions
```

## Contributing to Documentation

### Writing Guidelines

- **Use clear, concise language** - aim for accessibility
- **Include code examples** - show, don't just tell
- **Add validation checkpoints** - help users verify progress  
- **Cross-reference sections** - link related content
- **Follow markdown standards** - consistent formatting

### Preview Changes

```bash
# Start live-reload development server
mkdocs serve

# Preview builds automatically as you edit
# Check http://127.0.0.1:8000 for updates
```

### Content Standards

- **User-focused content** - written from the user's perspective
- **Step-by-step procedures** - numbered lists with clear actions
- **Troubleshooting sections** - anticipate common issues
- **Success criteria** - help users know when they're done
- **Cross-platform considerations** - include Windows/Mac/Linux

## MkDocs Configuration

Key configuration in `mkdocs.yml`:

- **Material theme** with Red Hat branding
- **Navigation tabs** for main sections
- **Search functionality** with highlighting
- **Mermaid diagrams** for system architecture
- **Code syntax highlighting** with copy buttons
- **Dark/light mode toggle**

## Deployment

### GitHub Pages (Recommended)

```bash
# Deploy to gh-pages branch
mkdocs gh-deploy

# Automatically builds and publishes to the gh-pages branch
```

### Custom Hosting

```bash
# Build static site
mkdocs build

# Deploy site/ directory to your web server
rsync -av site/ user@server:/var/www/acp-docs/
```

## Maintenance

### Regular Tasks

- **Review for accuracy** - validate against code changes
- **Update screenshots** - keep UI examples current
- **Check external links** - ensure they still work
- **Gather user feedback** - improve based on real usage

### Automated Checks

```bash
# Link checking (if plugin installed)
mkdocs build --strict

# Spell checking (with plugin)  
mkdocs build --plugin spellcheck

# Markdown linting
markdownlint docs/**/*.md
```

## Getting Help

### Documentation Issues

- **Typos or errors**: Submit a quick PR with fixes
- **Missing content**: Create an issue with details about what's needed
- **Unclear instructions**: Add feedback about which steps are confusing

### Technical Support

- **MkDocs issues**: Check [MkDocs documentation](https://www.mkdocs.org/)
- **Material theme**: Review [Material theme docs](https://squidfunk.github.io/mkdocs-material/)
- **Plugin problems**: Consult individual plugin documentation

---

This documentation system is designed to scale with the Ambient Code Platform. As features are added and the system evolves, the documentation structure can accommodate new content while maintaining clear organization and navigation.