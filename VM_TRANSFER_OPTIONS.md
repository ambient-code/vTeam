# Transferring Code to VM

## Option 1: Git Clone (Recommended if you push first)

**On your Mac:**
```bash
# Commit and push your changes
cd /Users/gkrumbac/.cursor/worktrees/vTeam/syBFS
git add .
git commit -m "Add LangGraph MVP support"
git push origin lang-graph  # or your branch name
```

**On your VM:**
```bash
git clone git@github.com:Gkrumbach07/vTeam.git
cd vTeam
git checkout lang-graph  # or your branch
```

## Option 2: SCP/rsync (Fast, works with any VM)

**From your Mac:**
```bash
# Replace VM_IP with your VM's IP address
# Replace USERNAME with your VM username
scp -r /Users/gkrumbac/.cursor/worktrees/vTeam/syBFS username@VM_IP:/home/username/vTeam

# Or use rsync (better for updates):
rsync -avz --exclude '.git' \
  /Users/gkrumbac/.cursor/worktrees/vTeam/syBFS/ \
  username@VM_IP:/home/username/vTeam/
```

## Option 3: Drag and Drop (VM Client Dependent)

**VMware Fusion/Parallels:**
- Usually supports drag-and-drop if VMware Tools/Parallels Tools is installed
- Just drag the folder from Finder to the VM window

**VirtualBox:**
- Enable "Shared Clipboard" and "Drag'n'Drop" in VM settings
- May need Guest Additions installed

**VMware Workstation/Fusion:**
- Enable "Drag and Drop" in VM settings
- Drag folder from host to guest

## Option 4: Shared Folder (Best for ongoing development)

**VMware Fusion:**
1. VM Settings → Sharing → Enable "Share Folders"
2. Add folder: `/Users/gkrumbac/.cursor/worktrees/vTeam/syBFS`
3. Access in VM at: `/mnt/hgfs/syBFS` (Linux) or `/Volumes/syBFS` (macOS guest)

**VirtualBox:**
1. VM Settings → Shared Folders
2. Add shared folder pointing to your code directory
3. Access via: `/media/sf_FolderName` (Linux) or `/Volumes/FolderName` (macOS)

## Recommended: Git Clone

If you push your changes, cloning in the VM is cleanest and keeps everything in sync.

