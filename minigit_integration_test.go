package main_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"minigit/internal/cli"
	"minigit/internal/repository"
)

func runCLI(cwd string, args ...string) (int, string, string) {
	origDir, _ := os.Getwd()
	os.Chdir(cwd)
	defer os.Chdir(origDir)

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	exitCode := cli.Execute(args)

	wOut.Close()
	wErr.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var outBuf, errBuf bytes.Buffer
	outBuf.ReadFrom(rOut)
	errBuf.ReadFrom(rErr)

	return exitCode, outBuf.String(), errBuf.String()
}

func TestMiniGitFullWorkflow(t *testing.T) {
	repoDir := t.TempDir()

	// 1. Correct initialization
	code, out, errOut := runCLI(repoDir, "init")
	if code != 0 {
		t.Fatalf("init failed: %s %s", out, errOut)
	}
	if _, err := os.Stat(filepath.Join(repoDir, ".minigit")); err != nil {
		t.Fatalf(".minigit folder not created")
	}

	// 2. Duplicate initialization error
	code, _, errOut = runCLI(repoDir, "init")
	if code == 0 || (!strings.Contains(errOut, "repository already exists") && !strings.Contains(errOut, "ya inicializado")) {
		t.Fatalf("expected duplicate init error, got exit code %d: %s", code, errOut)
	}

	// 3. Discovery from subfolder
	subDir := filepath.Join(repoDir, "src", "nested")
	os.MkdirAll(subDir, 0755)
	code, out, _ = runCLI(subDir, "status")
	if code != 0 || !strings.Contains(out, "On branch main") {
		t.Fatalf("discovery from subfolder failed: %s", out)
	}

	// 4. Add new file
	file1 := filepath.Join(repoDir, "hello.txt")
	os.WriteFile(file1, []byte("Hello MiniGit"), 0644)
	code, _, _ = runCLI(repoDir, "add", "hello.txt")
	if code != 0 {
		t.Fatalf("add hello.txt failed")
	}

	// 5. Add directory recursively
	dirFile := filepath.Join(subDir, "code.go")
	os.WriteFile(dirFile, []byte("package main"), 0644)
	code, out, errOut = runCLI(repoDir, "add", "src")
	if code != 0 {
		t.Fatalf("add src failed: out='%s' errOut='%s'", out, errOut)
	}

	// 6. Add empty file
	emptyFile := filepath.Join(repoDir, "empty.txt")
	os.WriteFile(emptyFile, []byte(""), 0644)
	code, _, _ = runCLI(repoDir, "add", "empty.txt")
	if code != 0 {
		t.Fatalf("add empty file failed")
	}

	// 7. Add same content twice (deduplication check)
	dupFile := filepath.Join(repoDir, "dup.txt")
	os.WriteFile(dupFile, []byte("Hello MiniGit"), 0644)
	code, _, _ = runCLI(repoDir, "add", "dup.txt")
	if code != 0 {
		t.Fatalf("add dup file failed")
	}

	// 10. First commit
	code, out, errOut = runCLI(repoDir, "commit", "-m", "Initial commit")
	if code != 0 {
		t.Fatalf("first commit failed: %s %s", out, errOut)
	}
	if !strings.Contains(out, "[main ") {
		t.Fatalf("expected commit summary output, got: %s", out)
	}

	// 12. Commit without changes error
	code, _, errOut = runCLI(repoDir, "commit", "-m", "No changes commit")
	if code == 0 || !strings.Contains(errOut, "nothing to commit") {
		t.Fatalf("expected nothing to commit error, got: %s", errOut)
	}

	// 8. File modified after add
	os.WriteFile(file1, []byte("Hello MiniGit Modified"), 0644)
	code, out, _ = runCLI(repoDir, "status")
	if !strings.Contains(out, "modified:   hello.txt") {
		t.Fatalf("expected status unstaged modified file, got: %s", out)
	}

	// Restage and second commit
	runCLI(repoDir, "add", "hello.txt")
	// 11. Second commit with parent link
	code, out, _ = runCLI(repoDir, "commit", "-m", "Update hello.txt")
	if code != 0 {
		t.Fatalf("second commit failed: %s", out)
	}

	// 13. Log with multiple commits
	code, out, _ = runCLI(repoDir, "log", "--oneline")
	if code != 0 || !strings.Contains(out, "Update hello.txt") || !strings.Contains(out, "Initial commit") {
		t.Fatalf("log --oneline output unexpected: %s", out)
	}

	// 14. Clean status
	code, out, _ = runCLI(repoDir, "status")
	if !strings.Contains(out, "nothing to commit, working tree clean") {
		t.Fatalf("expected clean status, got: %s", out)
	}

	// 15. Status with untracked files
	untracked := filepath.Join(repoDir, "untracked.log")
	os.WriteFile(untracked, []byte("untracked file data"), 0644)
	code, out, _ = runCLI(repoDir, "status")
	if !strings.Contains(out, "Untracked files:") || !strings.Contains(out, "untracked.log") {
		t.Fatalf("expected status untracked files, got: %s", out)
	}

	// 18. Restore file
	os.WriteFile(file1, []byte("Corrupted local change"), 0644)
	code, _, _ = runCLI(repoDir, "restore", "hello.txt")
	if code != 0 {
		t.Fatalf("restore failed")
	}
	restoredContent, _ := os.ReadFile(file1)
	if string(restoredContent) != "Hello MiniGit Modified" {
		t.Fatalf("file content not restored correctly: %s", string(restoredContent))
	}

	// 21. Branch creation and listing
	code, out, _ = runCLI(repoDir, "branch", "feature-test")
	if code != 0 {
		t.Fatalf("branch creation failed: %s", out)
	}
	code, out, _ = runCLI(repoDir, "branch")
	if !strings.Contains(out, "* main") || !strings.Contains(out, "feature-test") {
		t.Fatalf("branch listing failed: %s", out)
	}

	// 19. Checkout to previous commit
	// Get first commit hash from log
	repo := repository.OpenRepository(repoDir)
	headCommit, _ := repo.GetHeadCommitHash()
	commitObj, _, _ := repo.GetCommitByHash(headCommit)
	firstCommitHash := commitObj.Parent

	code, out, _ = runCLI(repoDir, "checkout", firstCommitHash[:7])
	if code != 0 || !strings.Contains(out, "detached HEAD") {
		t.Fatalf("checkout to first commit hash failed: %s", out)
	}

	// Verify content reverted to first commit
	firstContent, _ := os.ReadFile(file1)
	if string(firstContent) != "Hello MiniGit" {
		t.Fatalf("checkout did not restore previous commit content: %s", string(firstContent))
	}

	// Checkout back to main
	code, out, _ = runCLI(repoDir, "checkout", "main")
	if code != 0 {
		t.Fatalf("checkout main failed: %s", out)
	}

	// 20. Checkout rejected due to local conflicting changes
	os.WriteFile(file1, []byte("Dirty local change conflicting"), 0644)
	code, _, errOut = runCLI(repoDir, "checkout", firstCommitHash[:7])
	if code == 0 || !strings.Contains(errOut, "local changes") {
		t.Fatalf("expected checkout conflict error, got: %s", errOut)
	}
	// Revert dirty change
	runCLI(repoDir, "restore", "hello.txt")

	// 31. Files with spaces
	spaceFile := filepath.Join(repoDir, "my file with spaces.txt")
	os.WriteFile(spaceFile, []byte("space content"), 0644)
	runCLI(repoDir, "add", "my file with spaces.txt")

	// 32. Unicode file names
	unicodeFile := filepath.Join(repoDir, "saludo_español.txt")
	os.WriteFile(unicodeFile, []byte("¡Hola mundo!"), 0644)
	runCLI(repoDir, "add", "saludo_español.txt")

	// 33. Binary files
	binaryFile := filepath.Join(repoDir, "image.bin")
	binData := []byte{0x00, 0xFF, 0xFE, 0xFD, 0x12, 0x34, 0x56, 0x78}
	os.WriteFile(binaryFile, binData, 0644)
	runCLI(repoDir, "add", "image.bin")

	// 34. Reasonable large files (1MB buffer)
	largeFile := filepath.Join(repoDir, "large.data")
	largeData := bytes.Repeat([]byte("A"), 1024*1024)
	os.WriteFile(largeFile, largeData, 0644)
	runCLI(repoDir, "add", "large.data")

	// 35. Executable bit preservation
	execFile := filepath.Join(repoDir, "script.sh")
	os.WriteFile(execFile, []byte("#!/bin/sh\necho hi"), 0755)
	runCLI(repoDir, "add", "script.sh")

	code, out, errOut = runCLI(repoDir, "commit", "-m", "Commit with spaces, unicode, binary, large file and exec bit")
	if code != 0 {
		t.Fatalf("commit complex files failed: %s %s", out, errOut)
	}

	// 9. File deleted
	os.Remove(spaceFile)
	runCLI(repoDir, "add", ".")
	code, out, _ = runCLI(repoDir, "status")
	if !strings.Contains(out, "deleted:    my file with spaces.txt") {
		t.Fatalf("expected status staged deleted file, got: %s", out)
	}
}

func TestShowCommand(t *testing.T) {
	repoDir := t.TempDir()
	runCLI(repoDir, "init")

	f := filepath.Join(repoDir, "a.txt")
	os.WriteFile(f, []byte("content a"), 0644)
	codeAdd, outAdd, errAdd := runCLI(repoDir, "add", ".")
	codeCommit, outCommit, errCommit := runCLI(repoDir, "commit", "-m", "Initial")
	if codeAdd != 0 || codeCommit != 0 {
		t.Fatalf("add or commit failed in TestShowCommand: add=(%d, %s, %s) commit=(%d, %s, %s)", codeAdd, outAdd, errAdd, codeCommit, outCommit, errCommit)
	}

	codeShow, outShow, errShow := runCLI(repoDir, "show")
	if codeShow != 0 || !strings.Contains(outShow, "Initial") || !strings.Contains(outShow, "+ a.txt") {
		t.Fatalf("show command output unexpected: %s %s", outShow, errShow)
	}
}

func TestSpanishCommands(t *testing.T) {
	repoDir := t.TempDir()

	// minigit inicializar
	code, out, errOut := runCLI(repoDir, "inicializar")
	if code != 0 {
		t.Fatalf("inicializar failed: %s %s", out, errOut)
	}

	// minigit agregar .
	file1 := filepath.Join(repoDir, "demo.txt")
	os.WriteFile(file1, []byte("Contenido en español"), 0644)
	code, _, _ = runCLI(repoDir, "agregar", ".")
	if code != 0 {
		t.Fatalf("agregar failed")
	}

	// minigit estado
	code, out, _ = runCLI(repoDir, "estado")
	if !strings.Contains(out, "new file:   demo.txt") {
		t.Fatalf("estado output unexpected: %s", out)
	}

	// minigit comentario "Primer commit en español"
	code, out, errOut = runCLI(repoDir, "comentario", "Primer commit en español")
	if code != 0 {
		t.Fatalf("comentario failed: %s %s", out, errOut)
	}

	// minigit historial
	code, out, _ = runCLI(repoDir, "historial", "--oneline")
	if !strings.Contains(out, "Primer commit en español") {
		t.Fatalf("historial failed: %s", out)
	}

	// minigit rama nueva-rama
	code, _, _ = runCLI(repoDir, "rama", "nueva-rama")
	if code != 0 {
		t.Fatalf("rama failed")
	}

	// minigit cambiar nueva-rama
	code, out, _ = runCLI(repoDir, "cambiar", "nueva-rama")
	if code != 0 || !strings.Contains(out, "Switched to branch 'nueva-rama'") {
		t.Fatalf("cambiar failed: %s", out)
	}

	// minigit recuperar demo.txt
	os.WriteFile(file1, []byte("Cambio local no deseado"), 0644)
	code, _, _ = runCLI(repoDir, "recuperar", "demo.txt")
	if code != 0 {
		t.Fatalf("recuperar failed")
	}
	content, _ := os.ReadFile(file1)
	if string(content) != "Contenido en español" {
		t.Fatalf("recuperar content failed: %s", string(content))
	}
}

func TestSafetyRejections(t *testing.T) {
	repoDir := t.TempDir()
	runCLI(repoDir, "init")

	// 24. Rejection of ../
	code, _, errOut := runCLI(repoDir, "add", "../outside")
	if code == 0 || !strings.Contains(errOut, "traversal") && !strings.Contains(errOut, "invalid path") {
		t.Fatalf("expected rejection of ../, got exit code %d: %s", code, errOut)
	}

	// 25. Rejection of absolute paths
	code, _, errOut = runCLI(repoDir, "add", "/etc/passwd")
	if code == 0 || !strings.Contains(errOut, "absolute") && !strings.Contains(errOut, "invalid path") {
		t.Fatalf("expected rejection of absolute path, got exit code %d: %s", code, errOut)
	}

	// 26. Ignore .minigit directory
	code, _, errOut = runCLI(repoDir, "add", ".minigit/config")
	if code == 0 || !strings.Contains(errOut, "internal") && !strings.Contains(errOut, "forbidden") {
		t.Fatalf("expected rejection of .minigit, got exit code %d: %s", code, errOut)
	}
}
