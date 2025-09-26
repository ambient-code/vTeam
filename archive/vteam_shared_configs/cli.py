"""Command-line interface for vTeam shared configuration management."""

import click
from .installer import ConfigInstaller


@click.group()
@click.version_option(version="1.0.0")
def main():
    """vTeam Shared Claude Code Configuration Manager.
    
    Manage shared Claude Code configuration for team development standards.
    """
    pass


@main.command()
@click.option('--force', is_flag=True, help='Force reinstallation even if already installed')
def install(force):
    """Install vTeam shared Claude Code configuration.
    
    Sets up symlinks for global configuration and project templates.
    Automatically backs up existing configuration.
    """
    installer = ConfigInstaller()
    
    if installer.install(force_reinstall=force):
        click.echo(click.style("✅ vTeam configuration installed successfully!", fg="green"))
        installer.status()
    else:
        click.echo(click.style("❌ Installation failed", fg="red"))
        exit(1)


@main.command()
def uninstall():
    """Uninstall vTeam shared Claude Code configuration.
    
    Removes symlinks and restores backed up configuration if available.
    """
    installer = ConfigInstaller()
    
    if installer.uninstall():
        click.echo(click.style("✅ vTeam configuration uninstalled successfully!", fg="green"))
    else:
        click.echo(click.style("❌ Uninstallation failed", fg="red"))
        exit(1)


@main.command()
def status():
    """Show current vTeam configuration status.
    
    Displays whether configuration is active and properly linked.
    """
    installer = ConfigInstaller()
    installer.status()


@main.command()
def update():
    """Update to latest vTeam configuration.
    
    Equivalent to reinstalling with --force flag.
    """
    installer = ConfigInstaller()
    
    if installer.install(force_reinstall=True):
        click.echo(click.style("✅ vTeam configuration updated successfully!", fg="green"))
        installer.status()
    else:
        click.echo(click.style("❌ Update failed", fg="red"))
        exit(1)


if __name__ == "__main__":
    main()