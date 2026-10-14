package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var skipDirs = map[string]struct{}{".git": {}, ".idea": {}, "bin": {}, "testdata": {}, "vendor": {}, "node_modules": {}}

var (
	generated  = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)
	directives = []string{"//go:", "// +build", "//export ", "//line ", "//nolint", "//lint:"}
)

func main() {
	strip := flag.Bool("strip-comments", false, "remove all comments (except directives)")
	dry := flag.Bool("n", false, "only list the files that would change")
	flag.Parse()

	roots := flag.Args()

	if len(roots) == 0 {
		roots = []string{"."}
	}

	changed, failed := 0, 0

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				if _, skip := skipDirs[d.Name()]; skip && path != root {
					return filepath.SkipDir
				}

				return nil
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			ok, err := tidyFile(path, *strip, *dry)

			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
				failed++
				return nil
			}

			if ok {
				fmt.Println(path)
				changed++
			}

			return nil
		})

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr, "%d files changed, %d with errors\n", changed, failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func tidyFile(path string, strip, dry bool) (bool, error) {
	src, err := os.ReadFile(path)

	if err != nil {
		return false, err
	}

	if generated.Match(src) {
		return false, nil
	}

	out := src

	if strip {
		out, err = stripComments(out)

		if err != nil {
			return false, err
		}
	}

	out, err = groupImports(out, modulePath(path))

	if err != nil {
		return false, err
	}

	out, err = addSpacing(out)

	if err != nil {
		return false, err
	}

	if bytes.Equal(out, src) {
		return false, nil
	}

	if dry {
		return true, nil
	}

	return true, os.WriteFile(path, out, 0o644)
}

type edit struct {
	start, end int
	text       string
}

func apply(src []byte, edits []edit) []byte {
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	out := append([]byte(nil), src...)

	for _, e := range edits {
		out = append(out[:e.start], append([]byte(e.text), out[e.end:]...)...)
	}

	return out
}

func parse(src []byte) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	return fset, f, err
}

func isDirective(text string) bool {
	for _, d := range directives {
		if strings.HasPrefix(text, d) {
			return true
		}
	}

	return false
}

func stripComments(src []byte) ([]byte, error) {
	fset, f, err := parse(src)

	if err != nil {
		return nil, err
	}

	var edits []edit

	for _, g := range f.Comments {
		for _, c := range g.List {
			if isDirective(c.Text) {
				continue
			}

			start := fset.Position(c.Pos()).Offset
			end := fset.Position(c.End()).Offset
			lineStart := bytes.LastIndexByte(src[:start], '\n') + 1
			lineEnd := len(src)

			if i := bytes.IndexByte(src[end:], '\n'); i >= 0 {
				lineEnd = end + i
			}

			before := bytes.TrimSpace(src[lineStart:start])
			after := bytes.TrimSpace(src[end:lineEnd])

			switch {
			case len(before) == 0 && len(after) == 0:
				stop := lineEnd

				if stop < len(src) {
					stop++
				}

				edits = append(edits, edit{lineStart, stop, ""})
			case len(after) == 0:
				from := start

				for from > lineStart && (src[from-1] == ' ' || src[from-1] == '\t') {
					from--
				}

				edits = append(edits, edit{from, end, ""})
			default:
				edits = append(edits, edit{start, end, " "})
			}
		}
	}

	return format.Source(apply(src, edits))
}

func modulePath(path string) string {
	dir, _ := filepath.Abs(filepath.Dir(path))

	for {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))

		if err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				if mod, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
					return strings.TrimSpace(mod)
				}
			}

			return ""
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			return ""
		}

		dir = parent
	}
}

func importGroup(path, module string) int {
	switch {
	case module != "" && (path == module || strings.HasPrefix(path, module+"/")):
		return 2
	case !strings.Contains(strings.SplitN(path, "/", 2)[0], "."):
		return 0
	default:
		return 1
	}
}

func groupImports(src []byte, module string) ([]byte, error) {
	fset, f, err := parse(src)

	if err != nil {
		return nil, err
	}

	var edits []edit

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)

		if !ok || gd.Tok != token.IMPORT || !gd.Lparen.IsValid() || len(gd.Specs) < 2 {
			continue
		}

		start := fset.Position(gd.Lparen).Offset
		end := fset.Position(gd.Rparen).Offset

		if hasComment(f, gd.Lparen, gd.Rparen) {
			continue
		}

		var groups [3][]string

		for _, s := range gd.Specs {
			is := s.(*ast.ImportSpec)
			p, _ := strconv.Unquote(is.Path.Value)
			line := is.Path.Value

			if is.Name != nil {
				line = is.Name.Name + " " + line
			}

			g := importGroup(p, module)
			groups[g] = append(groups[g], line)
		}

		var b strings.Builder

		b.WriteString("(\n")

		first := true

		for _, g := range groups {
			if len(g) == 0 {
				continue
			}

			sort.Slice(g, func(i, j int) bool { return importPath(g[i]) < importPath(g[j]) })

			if !first {
				b.WriteString("\n")
			}

			first = false

			for _, line := range g {
				b.WriteString("\t" + line + "\n")
			}
		}

		edits = append(edits, edit{start, end, b.String()})
	}

	if len(edits) == 0 {
		return src, nil
	}

	return format.Source(apply(src, edits))
}

func importPath(line string) string {
	return line[strings.IndexByte(line, '"'):]
}

func hasComment(f *ast.File, from, to token.Pos) bool {
	for _, g := range f.Comments {
		if g.Pos() > from && g.End() < to {
			return true
		}
	}

	return false
}

func isBlock(s ast.Stmt) bool {
	switch s.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	}

	return false
}

func addSpacing(src []byte) ([]byte, error) {
	fset, f, err := parse(src)

	if err != nil {
		return nil, err
	}

	lines := bytes.Split(src, []byte("\n"))
	before := map[int]bool{}

	need := func(pos token.Pos) {
		before[leadLine(lines, fset.Position(pos).Line)] = true
	}

	stmts := func(list []ast.Stmt) {
		for i := 1; i < len(list); i++ {
			if isBlock(list[i]) || isBlock(list[i-1]) {
				need(list[i].Pos())
			}
		}
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BlockStmt:
			stmts(n.List)
		case *ast.CaseClause:
			stmts(n.Body)
		case *ast.CommClause:
			stmts(n.Body)
		}

		return true
	})

	for i := 1; i < len(f.Decls); i++ {
		prev, cur := f.Decls[i-1], f.Decls[i]

		if oneLiner(fset, prev) && oneLiner(fset, cur) {
			continue
		}

		need(declStart(cur))
	}

	var out bytes.Buffer

	for i, line := range lines {
		n := i + 1

		if before[n] && i > 0 && len(bytes.TrimSpace(lines[i-1])) > 0 && !opensBlock(lines[i-1]) {
			out.WriteByte('\n')
		}

		out.Write(line)

		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}

	return format.Source(out.Bytes())
}

func leadLine(lines [][]byte, line int) int {
	for line > 1 {
		prev := bytes.TrimSpace(lines[line-2])

		if !bytes.HasPrefix(prev, []byte("//")) {
			break
		}

		line--
	}

	return line
}

func opensBlock(line []byte) bool {
	t := bytes.TrimSpace(line)
	return bytes.HasSuffix(t, []byte("{")) || bytes.HasSuffix(t, []byte("(")) || bytes.HasSuffix(t, []byte(":"))
}

func declStart(d ast.Decl) token.Pos {
	switch d := d.(type) {
	case *ast.FuncDecl:
		if d.Doc != nil {
			return d.Doc.Pos()
		}
	case *ast.GenDecl:
		if d.Doc != nil {
			return d.Doc.Pos()
		}
	}

	return d.Pos()
}

func oneLiner(fset *token.FileSet, d ast.Decl) bool {
	fd, ok := d.(*ast.FuncDecl)
	return ok && fset.Position(fd.Pos()).Line == fset.Position(fd.End()).Line
}
