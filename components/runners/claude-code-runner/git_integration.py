#!/usr/bin/env python3

import asyncio
import json
import logging
import os
import subprocess
from pathlib import Path
from typing import Dict, List, Optional

logger = logging.getLogger(__name__)


class GitIntegration:
    """Git integration for claude-runner with authentication and repository management"""

    def __init__(self):
        self.user_name = os.getenv("GIT_USER_NAME", "")
        self.user_email = os.getenv("GIT_USER_EMAIL", "")
        self.repositories = self._parse_repositories()
        self.ssh_configured = False
        self.credentials_configured = False

    def _parse_repositories(self) -> List[Dict]:
        """Parse Git repositories from environment variable"""
        repos_json = os.getenv("GIT_REPOSITORIES", "[]")
        try:
            return json.loads(repos_json)
        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse GIT_REPOSITORIES: {e}")
            return []

    async def setup_git_config(self) -> bool:
        """Set up Git configuration and authentication"""
        try:
            logger.info("Setting up Git configuration...")

            # Configure Git user if provided
            if self.user_name:
                await self._run_git_command(["config", "--global", "user.name", self.user_name])
                logger.info(f"Set Git user.name: {self.user_name}")

            if self.user_email:
                await self._run_git_command(["config", "--global", "user.email", self.user_email])
                logger.info(f"Set Git user.email: {self.user_email}")

            # Set up SSH authentication if available
            await self._setup_ssh_auth()

            # Set up token authentication if available
            await self._setup_token_auth()

            # Configure Git for container environment
            await self._configure_git_environment()

            logger.info("Git configuration completed successfully")
            return True

        except Exception as e:
            logger.error(f"Failed to setup Git configuration: {e}")
            return False

    async def _setup_ssh_auth(self):
        """Set up SSH authentication for Git"""
        ssh_dir = Path("/tmp/.ssh")
        if ssh_dir.exists():
            logger.info("Setting up SSH authentication...")

            # Set up SSH directory permissions (try to chmod, but don't fail if read-only)
            try:
                ssh_dir.chmod(0o700)
                logger.info("Set SSH directory permissions")
            except (OSError, PermissionError) as e:
                logger.warning(f"Could not set SSH directory permissions: {e}")

            # Find SSH key files
            key_files = list(ssh_dir.glob("*"))
            for key_file in key_files:
                if key_file.name != "known_hosts":
                    try:
                        key_file.chmod(0o600)
                        logger.info(f"Set permissions for SSH key: {key_file}")
                    except (OSError, PermissionError) as e:
                        logger.warning(f"Could not set key permissions for {key_file}: {e}")

            # Set up SSH config to use the key
            ssh_config = ssh_dir / "config"
            if not ssh_config.exists():
                ssh_config_content = """Host *
    StrictHostKeyChecking no
    UserKnownHostsFile /tmp/.ssh/known_hosts
    IdentitiesOnly yes
"""
                # Add identity files for any private keys found
                for key_file in key_files:
                    if key_file.name not in ["config", "known_hosts"] and not key_file.name.endswith(".pub"):
                        ssh_config_content += f"    IdentityFile {key_file}\n"

                try:
                    ssh_config.write_text(ssh_config_content)
                    ssh_config.chmod(0o600)
                    logger.info("Created SSH config file")
                except (OSError, PermissionError) as e:
                    logger.warning(f"Could not create SSH config: {e}")
                    # Fall back to using GIT_SSH_COMMAND
                    await self._setup_git_ssh_command(key_files)

            self.ssh_configured = True
        else:
            logger.info("No SSH directory found, skipping SSH setup")

    async def _setup_git_ssh_command(self, key_files):
        """Set up Git SSH command as fallback for OpenShift read-only filesystem"""
        for key_file in key_files:
            if key_file.name not in ["config", "known_hosts"] and not key_file.name.endswith(".pub"):
                # Set GIT_SSH_COMMAND environment variable
                ssh_command = f"ssh -i {key_file} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"
                os.environ["GIT_SSH_COMMAND"] = ssh_command
                logger.info(f"Set GIT_SSH_COMMAND for OpenShift: {ssh_command}")
                break

    async def _setup_token_auth(self):
        """Set up token-based authentication for Git"""
        credentials_dir = Path("/tmp/.git-credentials")
        if credentials_dir.exists():
            logger.info("Setting up Git token authentication...")

            # Read token files and set up credential helper
            for cred_file in credentials_dir.iterdir():
                if cred_file.is_file():
                    try:
                        token = cred_file.read_text().strip()
                        if token:
                            # Configure credential helper for HTTPS
                            await self._run_git_command([
                                "config", "--global", "credential.helper",
                                f"store --file=/tmp/.git-credentials/{cred_file.name}"
                            ])
                            logger.info(f"Configured Git credentials from {cred_file.name}")
                    except Exception as e:
                        logger.warning(f"Failed to read credential file {cred_file}: {e}")

            self.credentials_configured = True

    async def _configure_git_environment(self):
        """Configure Git for container environment"""
        # Set safe directory for Git (OpenShift compatibility)
        await self._run_git_command(["config", "--global", "safe.directory", "*"])

        # Configure Git to use HTTPS instead of SSH for GitHub (fallback)
        await self._run_git_command([
            "config", "--global", "url.https://github.com/.insteadOf", "git@github.com:"
        ])

        # Set default branch name
        await self._run_git_command(["config", "--global", "init.defaultBranch", "main"])

        # Configure merge behavior
        await self._run_git_command(["config", "--global", "pull.rebase", "false"])

    async def clone_repositories(self, workspace_dir: Path) -> Dict[str, Path]:
        """Clone configured repositories to workspace"""
        cloned_repos = {}

        for repo in self.repositories:
            try:
                url = repo.get("url")
                branch = repo.get("branch", "main")
                clone_path = repo.get("clonePath", "")

                if not url:
                    logger.warning("Repository URL not provided, skipping")
                    continue

                # Determine clone destination
                if clone_path:
                    dest_dir = workspace_dir / clone_path
                else:
                    # Extract repository name from URL
                    repo_name = url.split("/")[-1].replace(".git", "")
                    dest_dir = workspace_dir / repo_name

                logger.info(f"Cloning repository: {url} -> {dest_dir}")

                # Clone the repository
                clone_result = await asyncio.create_subprocess_exec(
                    "git", "clone", "--branch", branch, url, str(dest_dir),
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE
                )
                stdout, stderr = await clone_result.communicate()

                if clone_result.returncode == 0:
                    logger.info(f"Successfully cloned {url} to {dest_dir}")
                    cloned_repos[url] = dest_dir
                else:
                    logger.error(f"Failed to clone {url}: {stderr.decode()}")

            except Exception as e:
                logger.error(f"Error cloning repository {repo}: {e}")

        return cloned_repos

    async def create_and_push_branch(self, repo_path: Path, branch_name: str, commit_message: str) -> bool:
        """Create a new branch, commit changes, and push to remote"""
        try:
            original_dir = Path.cwd()
            os.chdir(repo_path)

            logger.info(f"Creating and pushing branch: {branch_name}")

            # Create and checkout new branch
            await self._run_git_command(["checkout", "-b", branch_name])

            # Add all changes
            await self._run_git_command(["add", "."])

            # Check if there are changes to commit
            status_result = await asyncio.create_subprocess_exec(
                "git", "status", "--porcelain",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, _ = await status_result.communicate()

            if not stdout.strip():
                logger.info("No changes to commit")
                return True

            # Commit changes
            await self._run_git_command(["commit", "-m", commit_message])

            # Push branch to remote
            await self._run_git_command(["push", "-u", "origin", branch_name])

            logger.info(f"Successfully pushed branch: {branch_name}")
            return True

        except Exception as e:
            logger.error(f"Failed to create and push branch: {e}")
            return False
        finally:
            os.chdir(original_dir)

    async def _run_git_command(self, args: List[str]) -> bool:
        """Run a Git command and return success status"""
        try:
            result = await asyncio.create_subprocess_exec(
                "git", *args,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await result.communicate()

            if result.returncode == 0:
                return True
            else:
                logger.error(f"Git command failed: git {' '.join(args)}")
                logger.error(f"Error: {stderr.decode()}")
                return False

        except Exception as e:
            logger.error(f"Error running git command: {e}")
            return False

    def get_auth_status(self) -> Dict[str, bool]:
        """Get authentication status"""
        return {
            "ssh_configured": self.ssh_configured,
            "credentials_configured": self.credentials_configured,
            "user_configured": bool(self.user_name and self.user_email),
            "repositories_configured": len(self.repositories) > 0
        }