package gitlab

import (
	"ambient-code-backend/types"
)

// MapGitLabBranchToCommon converts a GitLabBranch to a common Branch type
func MapGitLabBranchToCommon(gitlabBranch types.GitLabBranch) types.Branch {
	return types.Branch{
		Name:      gitlabBranch.Name,
		Protected: gitlabBranch.Protected,
		Default:   gitlabBranch.Default,
		Commit: types.CommitInfo{
			SHA:       gitlabBranch.Commit.ID,
			Message:   gitlabBranch.Commit.Title,
			Author:    gitlabBranch.Commit.AuthorName,
			Timestamp: gitlabBranch.Commit.CommittedDate.Format("2006-01-02T15:04:05Z07:00"),
		},
	}
}

// MapGitLabBranchesToCommon converts multiple GitLab branches to common format
func MapGitLabBranchesToCommon(gitlabBranches []types.GitLabBranch) []types.Branch {
	branches := make([]types.Branch, len(gitlabBranches))
	for i, gb := range gitlabBranches {
		branches[i] = MapGitLabBranchToCommon(gb)
	}
	return branches
}

// MapGitLabTreeEntryToCommon converts a GitLabTreeEntry to a common TreeEntry type
func MapGitLabTreeEntryToCommon(gitlabEntry types.GitLabTreeEntry) types.TreeEntry {
	return types.TreeEntry{
		Name: gitlabEntry.Name,
		Path: gitlabEntry.Path,
		Type: gitlabEntry.Type,
		Mode: gitlabEntry.Mode,
		SHA:  gitlabEntry.ID,
	}
}

// MapGitLabTreeEntriesToCommon converts multiple GitLab tree entries to common format
func MapGitLabTreeEntriesToCommon(gitlabEntries []types.GitLabTreeEntry) []types.TreeEntry {
	entries := make([]types.TreeEntry, len(gitlabEntries))
	for i, ge := range gitlabEntries {
		entries[i] = MapGitLabTreeEntryToCommon(ge)
	}
	return entries
}

// MapGitLabFileContentToCommon converts GitLab file content to common format
func MapGitLabFileContentToCommon(gitlabFile *GitLabFileContent) types.FileContent {
	return types.FileContent{
		Name:     gitlabFile.FileName,
		Path:     gitlabFile.FilePath,
		Content:  gitlabFile.Content,
		Encoding: gitlabFile.Encoding,
		Size:     gitlabFile.Size,
		SHA:      gitlabFile.BlobID,
	}
}
