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
    return os.ReadFile(filepath.Join(root, rel))
}

func WriteModuleFile(mod Module, rel string, data []byte) error {
    root, err := moduleRepoPath(mod)
    if err != nil { return err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return err }
    abs := filepath.Join(root, rel)
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
    return os.RemoveAll(abs)
}

func MkdirModule(mod Module, rel string) error {
    root, err := moduleRepoPath(mod)
    if err != nil { return err }
    rel, err = sanitizeRelPath(rel)
    if err != nil { return err }
    abs := filepath.Join(root, rel)
    return os.MkdirAll(abs, 0o755)
}
