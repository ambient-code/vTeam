"""
Jira Integration Module for RFE Builder
Handles Epic creation and synchronization with Jira
"""

import json
import logging
from typing import Any, Dict, Optional
from urllib.parse import urljoin

import requests
import streamlit as st
from pydantic import BaseModel, ConfigDict

from data.rfe_models import RFE


logger = logging.getLogger(__name__)


class JiraConfig(BaseModel):
    """Jira configuration model"""
    
    model_config = ConfigDict(frozen=True)
    
    base_url: str
    username: str
    api_token: str
    project_key: str
    epic_issue_type: str = "Epic"
    default_assignee: Optional[str] = None
    timeout: int = 30


class JiraError(Exception):
    """Custom exception for Jira operations"""
    pass


class JiraClient:
    """Jira API client for Epic creation and management"""
    
    def __init__(self, config: JiraConfig):
        self.config = config
        self.session = requests.Session()
        self.session.auth = (config.username, config.api_token)
        self.session.headers.update({
            "Accept": "application/json",
            "Content-Type": "application/json"
        })
        self.api_base = urljoin(config.base_url, "/rest/api/3/")
    
    def test_connection(self) -> bool:
        """Test Jira connection and permissions"""
        try:
            url = urljoin(self.api_base, "myself")
            response = self.session.get(url, timeout=self.config.timeout)
            response.raise_for_status()
            return True
        except Exception as e:
            logger.error(f"Jira connection test failed: {e}")
            return False
    
    def get_project_info(self) -> Optional[Dict[str, Any]]:
        """Get project information and validate project key"""
        try:
            url = urljoin(self.api_base, f"project/{self.config.project_key}")
            response = self.session.get(url, timeout=self.config.timeout)
            response.raise_for_status()
            return response.json()
        except Exception as e:
            logger.error(f"Failed to get project info: {e}")
            return None
    
    def create_epic(self, epic_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new Epic in Jira"""
        try:
            url = urljoin(self.api_base, "issue")
            
            # Build Jira issue payload
            payload = {
                "fields": {
                    "project": {"key": self.config.project_key},
                    "issuetype": {"name": self.config.epic_issue_type},
                    "summary": epic_data["summary"],
                    "description": {
                        "type": "doc",
                        "version": 1,
                        "content": [
                            {
                                "type": "paragraph",
                                "content": [
                                    {
                                        "text": epic_data["description"],
                                        "type": "text"
                                    }
                                ]
                            }
                        ]
                    }
                }
            }
            
            # Add Epic Name (required for Epics in many Jira configurations)
            if "epic_name" in epic_data:
                # Try common Epic Name field IDs
                for field_name in ["customfield_10011", "customfield_10004", "customfield_12345"]:
                    payload["fields"][field_name] = epic_data["epic_name"]
                    break
            
            # Add assignee if specified
            if epic_data.get("assignee"):
                payload["fields"]["assignee"] = {"name": epic_data["assignee"]}
            elif self.config.default_assignee:
                payload["fields"]["assignee"] = {"name": self.config.default_assignee}
            
            # Add priority if specified
            if epic_data.get("priority"):
                payload["fields"]["priority"] = {"name": epic_data["priority"]}
            
            # Add labels if specified
            if epic_data.get("labels"):
                payload["fields"]["labels"] = epic_data["labels"]
            
            response = self.session.post(
                url, 
                json=payload, 
                timeout=self.config.timeout
            )
            response.raise_for_status()
            
            result = response.json()
            
            # Return Epic details
            return {
                "key": result["key"],
                "id": result["id"],
                "url": f"{self.config.base_url}/browse/{result['key']}",
                "self": result["self"]
            }
            
        except requests.exceptions.RequestException as e:
            if hasattr(e.response, "text"):
                logger.error(f"Jira API error: {e.response.text}")
                raise JiraError(f"Failed to create Epic: {e.response.text}")
            else:
                logger.error(f"Jira request failed: {e}")
                raise JiraError(f"Failed to create Epic: {e}")
        except Exception as e:
            logger.error(f"Unexpected error creating Epic: {e}")
            raise JiraError(f"Unexpected error: {e}")
    
    def update_epic(self, epic_key: str, update_data: Dict[str, Any]) -> bool:
        """Update an existing Epic"""
        try:
            url = urljoin(self.api_base, f"issue/{epic_key}")
            
            payload = {"fields": {}}
            
            if "summary" in update_data:
                payload["fields"]["summary"] = update_data["summary"]
            
            if "description" in update_data:
                payload["fields"]["description"] = {
                    "type": "doc",
                    "version": 1,
                    "content": [
                        {
                            "type": "paragraph",
                            "content": [
                                {
                                    "text": update_data["description"],
                                    "type": "text"
                                }
                            ]
                        }
                    ]
                }
            
            response = self.session.put(
                url, 
                json=payload, 
                timeout=self.config.timeout
            )
            response.raise_for_status()
            return True
            
        except Exception as e:
            logger.error(f"Failed to update Epic: {e}")
            raise JiraError(f"Failed to update Epic: {e}")


class JiraEpicMapper:
    """Maps RFE data to Jira Epic format"""
    
    @staticmethod
    def rfe_to_epic_data(rfe: RFE) -> Dict[str, Any]:
        """Convert RFE to Jira Epic data structure"""
        
        # Build Epic description from RFE content
        description_parts = [
            f"**RFE ID**: {rfe.id}",
            "",
            "## Description",
            rfe.description,
            ""
        ]
        
        if rfe.business_justification:
            description_parts.extend([
                "## Business Justification",
                rfe.business_justification,
                ""
            ])
        
        if rfe.technical_requirements:
            description_parts.extend([
                "## Technical Requirements", 
                rfe.technical_requirements,
                ""
            ])
        
        if rfe.success_criteria:
            description_parts.extend([
                "## Success Criteria",
                rfe.success_criteria,
                ""
            ])
        
        description_parts.extend([
            "## Workflow Information",
            f"- Current Status: {rfe.current_status.value}",
            f"- Current Step: {rfe.current_step}/7",
            f"- Created: {rfe.created_at.strftime('%Y-%m-%d %H:%M')}",
            f"- Updated: {rfe.updated_at.strftime('%Y-%m-%d %H:%M')}"
        ])
        
        if rfe.assigned_agent:
            description_parts.append(f"- Assigned Agent: {rfe.assigned_agent.value}")
        
        epic_data = {
            "summary": f"[RFE] {rfe.title}",
            "epic_name": rfe.title,
            "description": "\n".join(description_parts),
            "labels": ["RFE", "enhancement", rfe.current_status.value]
        }
        
        # Map priority based on RFE status
        if rfe.priority:
            epic_data["priority"] = rfe.priority
        elif rfe.current_status.value in ["accepted", "ticket_created"]:
            epic_data["priority"] = "High"
        elif rfe.current_status.value in ["prioritized", "under_review"]:
            epic_data["priority"] = "Medium"
        else:
            epic_data["priority"] = "Low"
        
        return epic_data


class JiraIntegration:
    """Main Jira integration class for RFE Builder"""
    
    def __init__(self):
        self.config = self._load_config()
        self.client = JiraClient(self.config) if self.config else None
        self.mapper = JiraEpicMapper()
    
    def _load_config(self) -> Optional[JiraConfig]:
        """Load Jira configuration from Streamlit secrets or environment"""
        try:
            # Check Streamlit secrets first
            if hasattr(st, "secrets") and "jira" in st.secrets:
                jira_secrets = st.secrets["jira"]
                return JiraConfig(
                    base_url=jira_secrets["base_url"],
                    username=jira_secrets["username"], 
                    api_token=jira_secrets["api_token"],
                    project_key=jira_secrets["project_key"],
                    epic_issue_type=jira_secrets.get("epic_issue_type", "Epic"),
                    default_assignee=jira_secrets.get("default_assignee")
                )
            
            # Fallback to environment variables (for production deployment)
            import os
            if all(os.getenv(var) for var in [
                "JIRA_BASE_URL", "JIRA_USERNAME", "JIRA_API_TOKEN", "JIRA_PROJECT_KEY"
            ]):
                return JiraConfig(
                    base_url=os.getenv("JIRA_BASE_URL"),
                    username=os.getenv("JIRA_USERNAME"),
                    api_token=os.getenv("JIRA_API_TOKEN"),
                    project_key=os.getenv("JIRA_PROJECT_KEY"),
                    epic_issue_type=os.getenv("JIRA_EPIC_ISSUE_TYPE", "Epic"),
                    default_assignee=os.getenv("JIRA_DEFAULT_ASSIGNEE")
                )
            
            return None
            
        except Exception as e:
            logger.error(f"Failed to load Jira configuration: {e}")
            return None
    
    def is_configured(self) -> bool:
        """Check if Jira integration is properly configured"""
        return self.config is not None and self.client is not None
    
    def test_connection(self) -> tuple[bool, str]:
        """Test Jira connection and return status message"""
        if not self.is_configured():
            return False, "Jira configuration not found. Please check secrets.toml."
        
        try:
            if not self.client.test_connection():
                return False, "Failed to connect to Jira. Check credentials and URL."
            
            project_info = self.client.get_project_info()
            if not project_info:
                return False, f"Project '{self.config.project_key}' not found or not accessible."
            
            return True, f"Connected to Jira project: {project_info['name']}"
            
        except Exception as e:
            return False, f"Connection test failed: {str(e)}"
    
    def create_epic_for_rfe(self, rfe: RFE) -> tuple[bool, str, Optional[Dict[str, str]]]:
        """
        Create Jira Epic for RFE
        Returns: (success, message, epic_info)
        """
        if not self.is_configured():
            return False, "Jira integration not configured", None
        
        # Check if Epic already exists
        if rfe.jira_epic_key:
            return False, f"Epic already exists: {rfe.jira_epic_key}", None
        
        try:
            # Map RFE to Epic data
            epic_data = self.mapper.rfe_to_epic_data(rfe)
            
            # Create Epic in Jira
            epic_result = self.client.create_epic(epic_data)
            
            return True, f"Epic created successfully: {epic_result['key']}", {
                "key": epic_result["key"],
                "url": epic_result["url"],
                "id": epic_result["id"]
            }
            
        except JiraError as e:
            return False, f"Failed to create Epic: {str(e)}", None
        except Exception as e:
            logger.error(f"Unexpected error creating Epic: {e}")
            return False, f"Unexpected error: {str(e)}", None
    
    def update_epic_for_rfe(self, rfe: RFE) -> tuple[bool, str]:
        """Update existing Epic with RFE changes"""
        if not self.is_configured():
            return False, "Jira integration not configured"
        
        if not rfe.jira_epic_key:
            return False, "No Epic associated with this RFE"
        
        try:
            epic_data = self.mapper.rfe_to_epic_data(rfe)
            success = self.client.update_epic(rfe.jira_epic_key, epic_data)
            
            if success:
                return True, f"Epic {rfe.jira_epic_key} updated successfully"
            else:
                return False, "Failed to update Epic"
                
        except JiraError as e:
            return False, f"Failed to update Epic: {str(e)}"
        except Exception as e:
            logger.error(f"Unexpected error updating Epic: {e}")
            return False, f"Unexpected error: {str(e)}"


# Global instance for app-wide use
jira_integration = JiraIntegration()