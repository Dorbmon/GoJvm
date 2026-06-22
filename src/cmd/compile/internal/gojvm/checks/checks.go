package checks

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gojvm-backend/src/gojvm/target"
)

type IssueKind string

const (
	IssueCGO      IssueKind = "cgo"
	IssueUnsafe   IssueKind = "unsafe"
	IssueAssembly IssueKind = "assembly"
)

type Issue struct {
	File    string
	Line    int
	Column  int
	Kind    IssueKind
	Message string
}

func (i Issue) Error() string { return i.Message }

const (
	msgCGO      = target.MsgCGOUnsupported
	msgUnsafe   = target.MsgUnsafeUnsupported
	msgAssembly = target.MsgAssemblyUnsupported
	msgLinkname = target.MsgLinknameUnsupported
)

// ValidateSources checks .go and assembly inputs for unsupported features for the JVM target.
func ValidateSources(paths []string) ([]Issue, error) {
	issues := make([]Issue, 0)
	seen := make(map[string]struct{})

	absPaths, err := expandPaths(paths, seen)
	if err != nil {
		return nil, err
	}

	for _, path := range absPaths {
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".s", ".S":
			issues = append(issues, Issue{
				File:    path,
				Kind:    IssueAssembly,
				Message: msgAssembly,
			})
		case ".go":
			goIssues, err := validateGoFile(path)
			if err != nil {
				return nil, err
			}
			issues = append(issues, goIssues...)
		}
	}

	return issues, nil
}

// ValidateBuildMode enforces the JVM-only build mode constraint.
func ValidateBuildMode(mode string) error {
	return target.ValidateBuildMode(mode)
}

func expandPaths(paths []string, seen map[string]struct{}) ([]string, error) {
	out := make([]string, 0, len(paths))
	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == ".git" || d.Name() == ".idea" || d.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if _, ok := seen[abs]; ok {
				return nil
			}
			seen[abs] = struct{}{}
			out = append(out, abs)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func validateGoFile(path string) ([]Issue, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, err
	}

	issues := make([]Issue, 0)
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		switch importPath {
		case "C":
			pos := fset.Position(imp.Path.Pos())
			issues = append(issues, Issue{
				File:    pos.Filename,
				Line:    pos.Line,
				Column:  pos.Column,
				Kind:    IssueCGO,
				Message: msgCGO,
			})
		case "unsafe":
			pos := fset.Position(imp.Path.Pos())
			issues = append(issues, Issue{
				File:    pos.Filename,
				Line:    pos.Line,
				Column:  pos.Column,
				Kind:    IssueUnsafe,
				Message: msgUnsafe,
			})
		}
	}

	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			if strings.Contains(comment.Text, "go:linkname") {
				pos := fset.Position(comment.Pos())
				issues = append(issues, Issue{
					File:    pos.Filename,
					Line:    pos.Line,
					Column:  pos.Column,
					Kind:    IssueCGO,
					Message: msgLinkname,
				})
			}
		}
	}
	return issues, nil
}
