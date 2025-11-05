# vTeam Static HTML Prototype

This directory contains a static HTML prototype of the vTeam user interface with dummy data. The prototype mirrors the actual application's sitemap and functionality for design and testing purposes.

## 🗂️ Site Structure

```
static-prototype/
├── index.html                          # Projects listing page
├── styles.css                          # Global styles
├── projects/
│   ├── new/
│   │   └── page.html                   # Create new project
│   └── sample-project/
│       ├── page.html                   # Project overview
│       ├── rfe/
│       │   └── page.html               # RFE Workspaces
│       ├── sessions/
│       │   └── page.html               # Agentic Sessions
│       ├── keys/
│       │   └── page.html               # API Keys
│       ├── permissions/
│       │   └── page.html               # User Permissions
│       └── settings/
│           └── page.html               # Project Settings
└── integrations/
    └── page.html                       # External Integrations
```

## 🎨 Features

- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Modern UI**: Clean, professional interface with consistent styling
- **Dummy Data**: Realistic sample data for all sections
- **Interactive Elements**: Buttons, forms, and navigation (visual only)
- **Complete Sitemap**: Mirrors the actual vTeam application structure

## 📱 Pages Included

### Main Navigation
- **Projects** (`index.html`) - List of all projects
- **Integrations** (`integrations/page.html`) - External service connections

### Project Pages
- **Project Overview** - Dashboard with stats and recent activity
- **RFE Workspaces** - Request for Enhancement workspaces
- **Agentic Sessions** - AI-powered coding sessions
- **API Keys** - Authentication key management
- **Permissions** - User and team access control
- **Settings** - Project configuration

### Forms & Creation
- **New Project** - Project creation form
- **Permission Management** - User access forms
- **Integration Settings** - Service configuration

## 🚀 Usage

1. **Open in Browser**: Simply open `index.html` in any modern web browser
2. **Navigate**: Click through the links to explore different sections
3. **Responsive Testing**: Resize browser window to test mobile layouts
4. **Design Review**: Use for UI/UX reviews and stakeholder demos

## 🎯 Use Cases

- **Design Reviews**: Share with stakeholders for UI feedback
- **User Testing**: Test navigation and information architecture
- **Development Reference**: Visual guide for implementing the real application
- **Documentation**: Screenshots and examples for documentation
- **Prototyping**: Quick iterations on design concepts

## 📊 Sample Data

The prototype includes realistic dummy data:
- 3 sample projects with different statuses
- Multiple RFE workspaces in various phases
- Agentic sessions with different completion states
- API keys with different permission levels
- User permissions with roles and groups
- Integration status for popular services

## 🔧 Customization

- **Styles**: Edit `styles.css` to modify colors, fonts, and layouts
- **Content**: Update HTML files to change sample data
- **Structure**: Add new pages following the existing pattern
- **Branding**: Replace logo and colors in the header section

## 📝 Notes

- All forms are visual only (no backend functionality)
- Links navigate between static HTML pages
- Responsive breakpoints at 768px for mobile
- Uses system fonts for better performance
- SVG icons for crisp display at all sizes
