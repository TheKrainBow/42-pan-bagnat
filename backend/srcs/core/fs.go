package core

import (
    "errors"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
)

type FileEntry struct {
    Name  string `json:"name"`
    Path  string `json:"path"`
    IsDir bool   `json:"is_dir"`
    Size  int64  `json:"size"`
}

func moduleRepoPath(mod Module) (string, error) {
    base := os.Getenv("REPO_BASE_PATH")
    if base == "" { base = "../../repos" }
    if mod.Slug == "" {
        return "", errors.New("invalid module slug")
    }
    return filepath.Join(base, mod.Slug), nil
}

// repoRootReal resolves the module repo root path and evaluates any symlinks.
func repoRootReal(mod Module) (string, error) {
    root, err := moduleRepoPath(mod)
    if err != nil { return "", err }
    real, err := filepath.EvalSymlinks(root)
    if err != nil { return "", err }
    return real, nil
}

// ensureWithin checks that the absolute target is inside the resolved root.
func ensureWithin(rootReal, targetAbs string) error {
    rel, err := filepath.Rel(rootReal, targetAbs)
    if err != nil { return err }
    if rel == "." { return nil }
    if strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
        return fmt.Errorf("path escapes repository root")
    }
    return nil
}

func sanitizeRelPath(p string) (string, error) {
    if p == "" { return ".", nil }
    clean := filepath.Clean(p)
    if strings.Contains(clean, "..") || strings.HasPrefix(clean, string(filepath.Separator)) {
        return "", errors.New("invalid path")
    }
    return clean, nil
}

func ListModuleDir(mod Module, rel string) ([]FileEntry, error) {
    root, err := moduleRepoPath(mod)
    if err != nil { return nil, err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return nil, err }
    dir := filepath.Join(root, rel)
    // Ensure resolved target stays within repo in case of symlinks
    rootReal, err := repoRootReal(mod)
    if err != nil { return nil, err }
    dirReal, err := filepath.EvalSymlinks(dir)
    if err == nil { // directory may not exist yet; if it does, validate
        if err := ensureWithin(rootReal, dirReal); err != nil { return nil, err }
    }
    entries, err := os.ReadDir(dir)
    if err != nil { return nil, err }
    out := []FileEntry{}
    for _, e := range entries {
        info, _ := e.Info()
        out = append(out, FileEntry{
            Name: e.Name(),
            Path: filepath.ToSlash(filepath.Join(rel, e.Name())),
            IsDir: e.IsDir(),
            Size: func(i fs.FileInfo) int64 { if i!=nil { return i.Size() }; return 0 }(info),
        })
    }
    return out, nil
}

func ReadModuleFile(mod Module, rel string) ([]byte, error) {
    root, err := moduleRepoPath(mod)
    if err != nil { return nil, err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return nil, err }
    abs := filepath.Join(root, rel)
    rootReal, err := repoRootReal(mod)
    if err != nil { return nil, err }
    fileReal, err := filepath.EvalSymlinks(abs)
    if err != nil { return nil, err }
    if err := ensureWithin(rootReal, fileReal); err != nil { return nil, err }
    return os.ReadFile(abs)
}

func WriteModuleFile(mod Module, rel string, data []byte) error {
    root, err := moduleRepoPath(mod)
    if err != nil { return err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return err }
    abs := filepath.Join(root, rel)
    // Validate parent directory (resolve symlinks on existing path)
    rootReal, err := repoRootReal(mod)
    if err != nil { return err }
    parent := filepath.Dir(abs)
    parentReal, err := filepath.EvalSymlinks(parent)
    if err == nil { // parent exists
        if err := ensureWithin(rootReal, parentReal); err != nil { return err }
    } else {
        // If parent doesn't exist yet, ensure the intended parent is under root
        if err := ensureWithin(rootReal, parent); err != nil { return err }
    }
    if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil { return err }
    return os.WriteFile(abs, data, 0o644)
}

func RenameModulePath(mod Module, oldRel, newRel string) error {
    root, err := moduleRepoPath(mod)
    if err != nil { return err }
    oldRel, err = sanitizeRelPath(oldRel)
    if err != nil { return err }
    newRel, err = sanitizeRelPath(newRel)
    if err != nil { return err }
    oldAbs := filepath.Join(root, oldRel)
    newAbs := filepath.Join(root, newRel)
    rootReal, err := repoRootReal(mod)
    if err != nil { return err }
    // Old path must resolve within root if it exists
    if oldReal, err := filepath.EvalSymlinks(oldAbs); err == nil {
        if err := ensureWithin(rootReal, oldReal); err != nil { return err }
    }
    // New parent must resolve within root (avoid moving into symlink outside)
    newParent := filepath.Dir(newAbs)
    if newParentReal, err := filepath.EvalSymlinks(newParent); err == nil {
        if err := ensureWithin(rootReal, newParentReal); err != nil { return err }
    } else {
        if err := ensureWithin(rootReal, newParent); err != nil { return err }
    }
    if _, statErr := os.Stat(newAbs); statErr == nil {
        return fmt.Errorf("destination already exists")
    }
    if err := os.MkdirAll(filepath.Dir(newAbs), 0o755); err != nil { return err }
    return os.Rename(oldAbs, newAbs)
}

func DeleteModulePath(mod Module, rel string) error {
    root, err := moduleRepoPath(mod)
    if err != nil { return err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return err }
    if rel == "." || rel == "" { return errors.New("refuse to delete root") }
    abs := filepath.Join(root, rel)
    // Ensure parent resolves within root (deleting symlink removes the link itself)
    rootReal, err := repoRootReal(mod)
    if err != nil { return err }
    parent := filepath.Dir(abs)
    if parentReal, err := filepath.EvalSymlinks(parent); err == nil {
        if err := ensureWithin(rootReal, parentReal); err != nil { return err }
    } else {
        if err := ensureWithin(rootReal, parent); err != nil { return err }
    }
    return os.RemoveAll(abs)
}

func MkdirModule(mod Module, rel string) error {
    root, err := moduleRepoPath(mod)
    if err != nil { return err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return err }
    abs := filepath.Join(root, rel)
    // Validate parent under root
    rootReal, err := repoRootReal(mod)
    if err != nil { return err }
    parent := filepath.Dir(abs)
    if parentReal, err := filepath.EvalSymlinks(parent); err == nil {
        if err := ensureWithin(rootReal, parentReal); err != nil { return err }
    } else {
        if err := ensureWithin(rootReal, parent); err != nil { return err }
    }
    return os.MkdirAll(abs, 0o755)
}
