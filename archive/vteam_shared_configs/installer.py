"""Configuration installer for vTeam shared Claude Code settings."""

import os
import shutil
import json
from pathlib import Path
from datetime import datetime
from importlib import resources
import click


class ConfigInstaller:
    """Manages installation and configuration of vTeam shared Claude settings."""
    
    def __init__(self):
        self.claude_dir = Path.home() / ".claude"
        self.global_config_link = self.claude_dir / "CLAUDE.md"
        self.templates_link = self.claude_dir / "project-templates"
        self.settings_file = self.claude_dir / "settings.json"
        
    def _get_package_data_path(self, filename):
        """Get path to package data file."""
        try:
            # Python 3.9+
            with resources.files("vteam_shared_configs.data") as data_path:
                return data_path / filename
        except AttributeError:
            # Python 3.8 fallback
            with resources.path("vteam_shared_configs.data", filename) as data_path:
                return data_path
    
    def _create_backup(self, file_path):
        """Create timestamped backup of existing file."""
        if not file_path.exists():
            return None
            
        timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
        backup_name = f"{file_path.name}.backup-{timestamp}"
        backup_path = file_path.parent / backup_name
        
        if file_path.is_symlink():
            # Just remove symlinks, no backup needed
            return None
        elif file_path.is_file():
            shutil.copy2(file_path, backup_path)
            click.echo(f"📦 Backed up {file_path.name} to {backup_name}")
            return backup_path
        elif file_path.is_dir():
            shutil.copytree(file_path, backup_path)
            click.echo(f"📦 Backed up {file_path.name}/ to {backup_name}/")
            return backup_path
            
        return None
    
    def _check_symlink(self, link_path, target_path):
        """Check if symlink exists and points to correct target."""
        if not link_path.is_symlink():
            return False
            
        try:
            current_target = link_path.resolve()
            expected_target = target_path.resolve()
            return current_target == expected_target
        except (OSError, RuntimeError):
            return False
    
    def _create_symlink(self, target_path, link_path, description):
        """Create symlink with proper error handling."""
        try:
            # Backup existing file/directory
            if link_path.exists():
                self._create_backup(link_path)
                if link_path.is_symlink():
                    link_path.unlink()
                elif link_path.is_file():
                    link_path.unlink()
                elif link_path.is_dir():
                    shutil.rmtree(link_path)
            
            # Create symlink
            link_path.symlink_to(target_path)
            click.echo(f"🔗 Linked {description}")
            return True
            
        except (OSError, RuntimeError) as e:
            click.echo(f"❌ Failed to link {description}: {e}")
            return False
    
    def install(self, force_reinstall=False):
        """Install vTeam shared configuration."""
        click.echo("🚀 Installing vTeam shared Claude Code configuration...")
        
        # Create .claude directory if needed
        if not self.claude_dir.exists():
            self.claude_dir.mkdir(parents=True)
            click.echo(f"📁 Created {self.claude_dir}")
        
        success = True
        
        # Install global configuration
        try:
            global_config_source = self._get_package_data_path("claude/global-CLAUDE.md")
            if not self._check_symlink(self.global_config_link, global_config_source) or force_reinstall:
                if not self._create_symlink(global_config_source, self.global_config_link, "global configuration"):
                    success = False
            else:
                click.echo("✅ Global configuration already linked")
        except Exception as e:
            click.echo(f"❌ Failed to install global configuration: {e}")
            success = False
        
        # Install project templates
        try:
            templates_source = self._get_package_data_path("claude/project-templates")
            if not self._check_symlink(self.templates_link, templates_source) or force_reinstall:
                if not self._create_symlink(templates_source, self.templates_link, "project templates"):
                    success = False
            else:
                click.echo("✅ Project templates already linked")
        except Exception as e:
            click.echo(f"❌ Failed to install project templates: {e}")
            success = False
        
        # Install team hooks (copy, not symlink, so user can modify)
        try:
            hooks_source = self._get_package_data_path(".claude/settings.json")
            if not self.settings_file.exists() or force_reinstall:
                self._create_backup(self.settings_file)
                shutil.copy2(hooks_source, self.settings_file)
                click.echo("⚙️ Installed team hooks configuration")
            else:
                click.echo("✅ Team hooks configuration already exists")
        except Exception as e:
            click.echo(f"❌ Failed to install team hooks: {e}")
            success = False
        
        return success
    
    def uninstall(self):
        """Uninstall vTeam shared configuration."""
        click.echo("🗑️ Uninstalling vTeam shared Claude Code configuration...")
        
        success = True
        
        # Remove symlinks
        for link_path, description in [
            (self.global_config_link, "global configuration"),
            (self.templates_link, "project templates")
        ]:
            if link_path.is_symlink():
                try:
                    link_path.unlink()
                    click.echo(f"🔓 Removed {description} symlink")
                except OSError as e:
                    click.echo(f"❌ Failed to remove {description}: {e}")
                    success = False
            elif link_path.exists():
                click.echo(f"⚠️ {description} exists but is not a symlink - leaving unchanged")
        
        # Find and offer to restore backups
        backup_pattern = "*.backup-*"
        backups = list(self.claude_dir.glob(backup_pattern))
        
        if backups:
            click.echo(f"📦 Found {len(backups)} backup(s)")
            for backup in sorted(backups, reverse=True):  # Most recent first
                original_name = backup.name.split('.backup-')[0]
                original_path = self.claude_dir / original_name
                
                if not original_path.exists():
                    if click.confirm(f"Restore {original_name} from {backup.name}?"):
                        try:
                            if backup.is_file():
                                shutil.copy2(backup, original_path)
                            else:
                                shutil.copytree(backup, original_path)
                            click.echo(f"✅ Restored {original_name}")
                            
                            if click.confirm(f"Remove backup {backup.name}?"):
                                if backup.is_file():
                                    backup.unlink()
                                else:
                                    shutil.rmtree(backup)
                        except Exception as e:
                            click.echo(f"❌ Failed to restore {original_name}: {e}")
                            success = False
        
        return success
    
    def status(self):
        """Display current configuration status."""
        click.echo("\n📊 vTeam Configuration Status")
        click.echo("=" * 35)
        
        # Check global configuration
        if self.global_config_link.is_symlink():
            try:
                target = self.global_config_link.resolve()
                if "vteam_shared_configs" in str(target):
                    click.echo("✅ Global configuration: Active (vTeam)")
                else:
                    click.echo(f"⚠️ Global configuration: Linked to different source")
            except (OSError, RuntimeError):
                click.echo("❌ Global configuration: Broken symlink")
        elif self.global_config_link.exists():
            click.echo("⚠️ Global configuration: File exists (not vTeam managed)")
        else:
            click.echo("❌ Global configuration: Not found")
        
        # Check project templates
        if self.templates_link.is_symlink():
            try:
                target = self.templates_link.resolve()
                if "vteam_shared_configs" in str(target):
                    click.echo("✅ Project templates: Active (vTeam)")
                else:
                    click.echo("⚠️ Project templates: Linked to different source")
            except (OSError, RuntimeError):
                click.echo("❌ Project templates: Broken symlink")
        elif self.templates_link.exists():
            click.echo("⚠️ Project templates: Directory exists (not vTeam managed)")
        else:
            click.echo("❌ Project templates: Not found")
        
        # Check team hooks
        if self.settings_file.exists():
            try:
                with open(self.settings_file) as f:
                    settings = json.load(f)
                    if any("vteam" in str(hook).lower() for hook in settings.get("hooks", {}).values() if isinstance(hook, list)):
                        click.echo("✅ Team hooks: Active")
                    else:
                        click.echo("⚠️ Team hooks: File exists (may not be vTeam)")
            except (json.JSONDecodeError, OSError):
                click.echo("❌ Team hooks: File exists but invalid JSON")
        else:
            click.echo("❌ Team hooks: Not found")
        
        # Check for local overrides
        for project_root in [Path.cwd(), Path.cwd().parent]:
            local_settings = project_root / ".claude" / "settings.local.json"
            if local_settings.exists():
                click.echo(f"ℹ️ Local overrides: Found in {project_root.name}")
                break
        else:
            click.echo("ℹ️ Local overrides: None found")
        
        click.echo("\n💡 Use 'vteam-config install' to set up configuration")
        click.echo("💡 Create '.claude/settings.local.json' for personal overrides")