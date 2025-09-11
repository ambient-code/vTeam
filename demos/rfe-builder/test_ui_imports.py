"""Test UI component imports to catch restricted import issues."""

import pytest
import ast
import os
from pathlib import Path


class TestUIComponentImports:
    """Test suite to prevent UI component import issues in LlamaIndex environment."""
    
    def test_ui_components_use_allowed_imports_only(self):
        """Test that UI components only use allowed import libraries."""
        allowed_imports = {
            # React core
            "react",
            # Shadcn UI components
            "@/components/ui/card", "@/components/ui/button", "@/components/ui/textarea",
            "@/components/ui/alert", "@/components/ui/tabs", "@/components/ui/checkbox",
            "@/components/ui/label", "@/components/ui/badge", "@/components/ui/tooltip",
            "@/components/ui/select", "@/components/ui/input", "@/components/ui/dialog",
            "@/components/ui/dropdown-menu", "@/components/ui/popover", "@/components/ui/sheet",
            "@/components/ui/scroll-area", "@/components/ui/separator", "@/components/ui/progress",
            # Shadcn utilities
            "@/lib/utils",
            # LlamaIndex chat-ui (allowed)
            "llamaindex/chat-ui",
            # Lucide React icons (allowed)
            "lucide-react",
            # Zod validation (allowed)
            "zod"
        }
        
        ui_components_dir = Path("ui/components")
        if not ui_components_dir.exists():
            pytest.skip("UI components directory not found")
        
        jsx_files = list(ui_components_dir.glob("*.jsx"))
        js_files = list(ui_components_dir.glob("*.js"))
        
        for file_path in jsx_files + js_files:
            with open(file_path, 'r') as f:
                content = f.read()
            
            # Extract import statements using regex
            import re
            import_pattern = r'import\s+.*?\s+from\s+["\']([^"\']+)["\']'
            imports = re.findall(import_pattern, content)
            
            for imported_module in imports:
                # Skip relative imports to other UI components (these cause the main issue)
                # Exception: index.js files are allowed to have relative imports for re-exports
                if (imported_module.startswith('./') or imported_module.startswith('../')) and file_path.name != 'index.js':
                    pytest.fail(
                        f"File {file_path} contains relative import '{imported_module}' "
                        f"which is not allowed in LlamaIndex environment. "
                        f"Use absolute imports or inline the component."
                    )
                
                # Check if import is in allowed list (skip relative imports since they're handled above)
                if not (imported_module.startswith('./') or imported_module.startswith('../')) and not any(allowed in imported_module for allowed in allowed_imports):
                    pytest.fail(
                        f"File {file_path} imports '{imported_module}' which is not allowed. "
                        f"Only these imports are supported: {', '.join(sorted(allowed_imports))}"
                    )
    
    def test_no_custom_component_exports_in_index(self):
        """Test that index.js doesn't export components that use restricted imports."""
        index_file = Path("ui/components/index.js")
        if not index_file.exists():
            pytest.skip("UI components index.js not found")
        
        # List of components known to have import issues
        problematic_components = [
            "FrameworkSelector",
            "EnhancedRFEBuilder", 
            "MultiFrameworkProgress"
        ]
        
        with open(index_file, 'r') as f:
            content = f.read()
        
        for component in problematic_components:
            if component in content:
                pytest.fail(
                    f"index.js exports '{component}' which has restricted import issues. "
                    f"This component should be refactored to use only allowed imports "
                    f"or moved to a different environment."
                )
    
    def test_jsx_files_have_valid_syntax(self):
        """Test that all JSX files have valid syntax."""
        ui_components_dir = Path("ui/components")
        if not ui_components_dir.exists():
            pytest.skip("UI components directory not found")
        
        jsx_files = list(ui_components_dir.glob("*.jsx"))
        
        for jsx_file in jsx_files:
            with open(jsx_file, 'r') as f:
                content = f.read()
            
            # Basic syntax check - look for common JSX issues
            if 'export' not in content:
                pytest.fail(f"File {jsx_file} may have syntax issues - no export found")
            
            # Check for proper export syntax
            if content.count('export') > 0:
                # Look for either 'export function' or 'export default'
                valid_exports = ['export function', 'export default', 'export {', 'export const']
                if not any(export_type in content for export_type in valid_exports):
                    pytest.fail(f"File {jsx_file} has invalid export syntax")


class TestUIComponentDependencies:
    """Test UI component file dependencies and structure."""
    
    def test_ui_components_directory_structure(self):
        """Test that UI components follow expected directory structure."""
        ui_dir = Path("ui")
        components_dir = Path("ui/components")
        
        if ui_dir.exists():
            assert components_dir.exists(), "ui/components directory should exist if ui/ exists"
            
            # Check for index file
            index_files = list(components_dir.glob("index.*"))
            assert len(index_files) > 0, "UI components should have an index file"
    
    def test_no_circular_imports_in_components(self):
        """Test that components don't have circular import dependencies."""
        ui_components_dir = Path("ui/components")
        if not ui_components_dir.exists():
            pytest.skip("UI components directory not found")
        
        js_files = list(ui_components_dir.glob("*.js")) + list(ui_components_dir.glob("*.jsx"))
        
        for file_path in js_files:
            with open(file_path, 'r') as f:
                content = f.read()
            
            # Check for imports that could create cycles
            if './index' in content and file_path.name != 'index.js':
                pytest.fail(f"Component {file_path} imports from index, which could create circular dependency")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])