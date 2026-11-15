package snapshot

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Create takes a virtually instantaneous snapshot of the workspace
// using Git's low-level plumbing commands, ensuring zero pollution to
// the user's branch history or staging area.
func Create(workspace string) error {
	if err := ensureGitRepository(workspace); err != nil {
		return err
	}

	index, err := os.CreateTemp("", "stuntdouble-index-*")
	if err != nil {
		return err
	}
	indexFile := index.Name()
	index.Close()
	defer os.Remove(indexFile)

	// Create a temporary index loaded with HEAD (if present)
	cmd1 := exec.Command("git", "read-tree", "HEAD")
	cmd1.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexFile)
	cmd1.Dir = workspace
	_ = cmd1.Run()

	// Add all current working directory files to the temporary index
	cmd2 := exec.Command("git", "add", ".")
	cmd2.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexFile)
	cmd2.Dir = workspace
	if err := cmd2.Run(); err != nil {
		return err
	}

	// Write the index to a tree object in the git database
	cmd3 := exec.Command("git", "write-tree")
	cmd3.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexFile)
	cmd3.Dir = workspace
	out, err := cmd3.Output()
	if err != nil {
		return err
	}

	treeHash := strings.TrimSpace(string(out))

	// Save the tree hash
	stuntDir := getStorageDir(workspace)
	os.MkdirAll(stuntDir, 0755)
	snapshotFile := filepath.Join(stuntDir, "latest_snapshot.txt")
	if err := os.WriteFile(snapshotFile, []byte(treeHash), 0644); err != nil {
		return err
	}

	fmt.Printf("📸 [agentsh] Zero-copy workspace snapshot captured (Tree: %s)\n", treeHash[:8])
	return nil
}

func getStorageDir(workspace string) string {
	if _, err := os.Stat(filepath.Join(workspace, ".agentsh")); err == nil {
		return filepath.Join(workspace, ".agentsh")
	}
	return filepath.Join(workspace, ".stuntdouble")
}

// SaveCheckpoint creates a named workspace snapshot saved in .agentsh/checkpoints/<name>.txt
func SaveCheckpoint(workspace, name string) error {
	if name == "" {
		return fmt.Errorf("checkpoint name cannot be empty")
	}

	treeHash, err := captureTreeHash(workspace)
	if err != nil {
		return fmt.Errorf("failed to capture workspace tree: %w", err)
	}

	checkpointDir := filepath.Join(getStorageDir(workspace), "checkpoints")
	if err := os.MkdirAll(checkpointDir, 0755); err != nil {
		return err
	}

	checkpointFile := filepath.Join(checkpointDir, name+".txt")
	if err := os.WriteFile(checkpointFile, []byte(treeHash), 0644); err != nil {
		return err
	}

	fmt.Printf("🔖 [agentsh] Checkpoint %q saved (Tree: %s)\n", name, treeHash[:8])
	return nil
}

// RestoreCheckpoint rewinds workspace to a previously saved named checkpoint
func RestoreCheckpoint(workspace, name string) error {
	checkpointFile := filepath.Join(getStorageDir(workspace), "checkpoints", name+".txt")
	data, err := os.ReadFile(checkpointFile)
	if err != nil {
		// Fallback check in .stuntdouble if getStorageDir returned .agentsh
		fallbackFile := filepath.Join(workspace, ".stuntdouble", "checkpoints", name+".txt")
		var fallbackErr error
		data, fallbackErr = os.ReadFile(fallbackFile)
		if fallbackErr != nil {
			return fmt.Errorf("checkpoint %q not found: %w", name, err)
		}
	}

	treeHash := strings.TrimSpace(string(data))
	if err := restoreTreeHash(workspace, treeHash); err != nil {
		return fmt.Errorf("failed to restore checkpoint %q: %w", name, err)
	}

	fmt.Printf("⏪ [agentsh] Workspace rewound to checkpoint %q (Tree: %s)\n", name, treeHash[:8])
	return nil
}

// ListCheckpoints returns all saved checkpoint names
func ListCheckpoints(workspace string) ([]string, error) {
	checkpointDir := filepath.Join(getStorageDir(workspace), "checkpoints")
	entries, err := os.ReadDir(checkpointDir)
	if err != nil && os.IsNotExist(err) {
		// Try fallback
		checkpointDir = filepath.Join(workspace, ".stuntdouble", "checkpoints")
		entries, err = os.ReadDir(checkpointDir)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			names = append(names, strings.TrimSuffix(entry.Name(), ".txt"))
		}
	}
	return names, nil
}

func captureTreeHash(workspace string) (string, error) {
	if err := ensureGitRepository(workspace); err != nil {
		return "", err
	}

	index, err := os.CreateTemp("", "stuntdouble-index-*")
	if err != nil {
		return "", err
	}
	indexFile := index.Name()
	index.Close()
	defer os.Remove(indexFile)

	// Try loading HEAD if it exists, but ignore error if HEAD is not created yet (new repo)
	cmd1 := exec.Command("git", "read-tree", "HEAD")
	cmd1.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexFile)
	cmd1.Dir = workspace
	_ = cmd1.Run()

	cmd2 := exec.Command("git", "add", ".")
	cmd2.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexFile)
	cmd2.Dir = workspace
	if err := cmd2.Run(); err != nil {
		return "", err
	}

	cmd3 := exec.Command("git", "write-tree")
	cmd3.Env = append(os.Environ(), "GIT_INDEX_FILE="+indexFile)
	cmd3.Dir = workspace
	out, err := cmd3.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func restoreTreeHash(workspace, treeHash string) error {
	cmd1 := exec.Command("git", "restore", "--source", treeHash, "--worktree", ".")
	cmd1.Dir = workspace
	if err := cmd1.Run(); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	return removeUntrackedAbsentFromSnapshot(workspace, treeHash)
}

// Restore rewinds the workspace to the latest captured snapshot
func Restore(workspace string) error {
	snapshotFile := filepath.Join(getStorageDir(workspace), "latest_snapshot.txt")
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		fallbackFile := filepath.Join(workspace, ".agentsh", "latest_snapshot.txt")
		var fallbackErr error
		data, fallbackErr = os.ReadFile(fallbackFile)
		if fallbackErr != nil {
			return fmt.Errorf("no StuntDouble snapshot found: %w", err)
		}
	}

	treeHash := strings.TrimSpace(string(data))
	return restoreTreeHash(workspace, treeHash)
}

func removeUntrackedAbsentFromSnapshot(workspace, treeHash string) error {
	snapshotFiles, err := filesInTree(workspace, treeHash)
	if err != nil {
		return err
	}

	untracked, err := untrackedFiles(workspace)
	if err != nil {
		return err
	}

	for _, rel := range untracked {
		if _, ok := snapshotFiles[rel]; ok {
			continue
		}

		cleanRel := filepath.Clean(rel)
		if cleanRel == "." || filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, ".."+string(os.PathSeparator)) || cleanRel == ".." {
			return fmt.Errorf("refusing to remove unsafe path %q", rel)
		}
		if err := os.RemoveAll(filepath.Join(workspace, cleanRel)); err != nil {
			return err
		}
	}
	return nil
}

func filesInTree(workspace, treeHash string) (map[string]struct{}, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "-z", "--name-only", treeHash)
	cmd.Dir = workspace
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	files := make(map[string]struct{})
	for _, part := range bytes.Split(out, []byte{0}) {
		if len(part) == 0 {
			continue
		}
		files[string(part)] = struct{}{}
	}
	return files, nil
}

func untrackedFiles(workspace string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "--others", "--exclude-standard", "-z")
	cmd.Dir = workspace
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var files []string
	for _, part := range bytes.Split(out, []byte{0}) {
		if len(part) == 0 {
			continue
		}
		files = append(files, string(part))
	}
	return files, nil
}

func ensureGitRepository(workspace string) error {
	if _, err := os.Stat(filepath.Join(workspace, ".git")); os.IsNotExist(err) {
		fmt.Println("⚡ [StuntDouble] Initializing zero-copy git index in non-git workspace...")
		cmd := exec.Command("git", "init")
		cmd.Dir = workspace
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("initializing git repository: %w", err)
		}

		// Create initial empty commit anchor so HEAD exists for git plumbing
		cmdCommit := exec.Command("git", "-c", "user.name=StuntDouble", "-c", "user.email=stuntdouble@internal", "commit", "--allow-empty", "-m", "initial zero-copy snapshot anchor")
		cmdCommit.Dir = workspace
		_ = cmdCommit.Run()

		gitignore := filepath.Join(workspace, ".gitignore")
		f, err := os.OpenFile(gitignore, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString("\n.stuntdouble/\n.stuntdouble.telemetry.json\n")
			f.Close()
		}
	}
	return nil
}
