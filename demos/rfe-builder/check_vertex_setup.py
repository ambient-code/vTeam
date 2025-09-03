#!/usr/bin/env python3
"""
Vertex AI Setup Verification Script for RFE Builder

Run this script to diagnose Vertex AI configuration issues.
"""

import os
import sys
from typing import List, Tuple


def check_environment_variables() -> List[Tuple[str, bool, str]]:
    """Check required environment variables"""
    required_vars = [
        ("CLAUDE_CODE_USE_VERTEX", True, "Must be set to '1' to enable Vertex AI"),
        ("ANTHROPIC_VERTEX_PROJECT_ID", True, "Your Google Cloud project ID"),
        ("CLOUD_ML_REGION", True, "Vertex AI region (e.g., us-east5)"),
        ("ANTHROPIC_MODEL", False, "Primary model (optional)"),
        ("ANTHROPIC_SMALL_FAST_MODEL", False, "Fast model (optional)"),
        ("GOOGLE_APPLICATION_CREDENTIALS", False, "Service account key path (optional if using gcloud auth)"),
    ]
    
    results = []
    for var_name, required, description in required_vars:
        value = os.getenv(var_name)
        is_set = value is not None and value.strip() != ""
        results.append((var_name, is_set, description, required, value))
    
    return results


def check_python_imports() -> List[Tuple[str, bool, str]]:
    """Check required Python imports"""
    imports_to_check = [
        ("google.auth", "Google Cloud authentication"),
        ("google.cloud.aiplatform", "Google Cloud AI Platform"),
        ("anthropic", "Anthropic Python client"),
        ("anthropic.lib.vertex", "Anthropic Vertex AI support"),
    ]
    
    results = []
    for module_name, description in imports_to_check:
        try:
            __import__(module_name)
            results.append((module_name, True, description))
        except ImportError as e:
            results.append((module_name, False, f"{description} - Error: {e}"))
    
    return results


def check_google_auth():
    """Check Google Cloud authentication"""
    try:
        from google.auth import default
        credentials, project = default()
        return True, f"Authenticated for project: {project}"
    except Exception as e:
        return False, f"Authentication failed: {e}"


def check_vertex_client():
    """Test creating AnthropicVertex client"""
    try:
        from anthropic import AnthropicVertex
        
        project_id = os.getenv("ANTHROPIC_VERTEX_PROJECT_ID")
        region = os.getenv("CLOUD_ML_REGION")
        
        if not project_id or not region:
            return False, "Missing project_id or region environment variables"
        
        # Test client creation
        client = AnthropicVertex(project_id=project_id, region=region)
        return True, f"Client created successfully for {project_id} in {region}"
        
    except Exception as e:
        return False, f"Client creation failed: {e}"


def check_gcloud_cli():
    """Check if gcloud CLI is working"""
    try:
        import subprocess
        
        # Check if gcloud is installed
        result = subprocess.run(["gcloud", "version"], 
                              capture_output=True, text=True, timeout=10)
        
        if result.returncode != 0:
            return False, "gcloud CLI not found or not working"
        
        # Check authentication
        result = subprocess.run(["gcloud", "auth", "list", "--filter=status:ACTIVE", "--format=value(account)"],
                              capture_output=True, text=True, timeout=10)
        
        if result.returncode == 0 and result.stdout.strip():
            active_account = result.stdout.strip().split('\n')[0]
            return True, f"gcloud authenticated as: {active_account}"
        else:
            return False, "No active gcloud authentication found"
            
    except subprocess.TimeoutExpired:
        return False, "gcloud command timed out"
    except FileNotFoundError:
        return False, "gcloud CLI not installed"
    except Exception as e:
        return False, f"gcloud check failed: {e}"


def print_section(title: str):
    """Print a section header"""
    print(f"\n{'='*60}")
    print(f"  {title}")
    print('='*60)


def print_check_result(name: str, success: bool, details: str, required: bool = True):
    """Print the result of a check"""
    if success:
        icon = "✅"
        status = "PASS"
    elif required:
        icon = "❌"
        status = "FAIL"
    else:
        icon = "⚠️ "
        status = "OPTIONAL"
    
    print(f"{icon} {status:8} {name:35} - {details}")


def main():
    """Main diagnostic function"""
    print("🧪 Vertex AI Setup Verification for RFE Builder")
    print("This script will help diagnose Vertex AI configuration issues.")
    print("ℹ️  Note: This script displays project IDs and regions for diagnostic purposes.\n")
    
    all_good = True
    
    # Check environment variables
    print_section("Environment Variables")
    env_results = check_environment_variables()
    for var_name, is_set, description, required, value in env_results:
        if is_set:
            details = f"{description} ✓"
            # Show values for diagnostic purposes (low sensitivity)
            if var_name in ["ANTHROPIC_VERTEX_PROJECT_ID", "CLOUD_ML_REGION"] and value:
                details += f" (Value: {value})"
        else:
            details = f"{description} - NOT SET"
            if required:
                all_good = False
        
        print_check_result(var_name, is_set, details, required)
    
    # Check Python imports
    print_section("Python Dependencies")
    import_results = check_python_imports()
    for module_name, success, details in import_results:
        print_check_result(module_name, success, details)
        if not success:
            all_good = False
    
    # Check Google Cloud authentication
    print_section("Google Cloud Authentication")
    auth_success, auth_details = check_google_auth()
    print_check_result("Application Default Credentials", auth_success, auth_details)
    if not auth_success:
        all_good = False
    
    # Check gcloud CLI
    gcloud_success, gcloud_details = check_gcloud_cli()
    print_check_result("gcloud CLI", gcloud_success, gcloud_details, required=False)
    
    # Check Vertex AI client
    print_section("Vertex AI Client")
    client_success, client_details = check_vertex_client()
    print_check_result("AnthropicVertex Client", client_success, client_details)
    if not client_success:
        all_good = False
    
    # Summary
    print_section("Summary")
    if all_good:
        print("🎉 All checks passed! Your Vertex AI setup should work.")
        print("✅ You can now run the RFE Builder with Vertex AI support.")
        return 0
    else:
        print("❌ Some issues were found. Please address the failures above.")
        print("\n💡 Common solutions:")
        print("   • Run: gcloud auth application-default login")
        print("   • Set missing environment variables")
        print("   • Install dependencies: uv pip install -r requirements.txt")
        print("   • Check the VERTEX_AI_SETUP.md guide for detailed steps")
        return 1


if __name__ == "__main__":
    sys.exit(main())
