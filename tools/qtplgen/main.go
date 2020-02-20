// Command qtplgen generates the quick template files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	qtcparser "github.com/valyala/quicktemplate/parser"
	"github.com/yookoala/realpath"
)

func main() {
	path := flag.String("path", "", "path")
	tplSuffix := flag.String("tpl-suffix", ".qtpl", "template suffix")
	genSuffix := flag.String("gen-suffix", ".qtpl.go", "generated suffix")
	flag.Parse()
	if err := run(*path, *tplSuffix, *genSuffix); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(dir, tplSuffix, genSuffix string) error {
	// save current working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	// resolve directory
	if dir == "" {
		dir = wd
	}
	dir, err = realpath.Realpath(dir)
	if err != nil {
		return fmt.Errorf("unable to determine realpath for %s: %v", dir, err)
	}
	// generate
	if err = filepath.Walk(dir, generate(tplSuffix, genSuffix)); err != nil {
		return err
	}
	return os.Chdir(wd)
}

// generate returns a filepath walker that generates and formats the go source
// for quicktemplate (*.qtpl) files in the specified directory.
func generate(tplSuffix, genSuffix string) filepath.WalkFunc {
	return func(name string, fi os.FileInfo, err error) error {
		switch {
		case err != nil:
			return err
		case fi.IsDir() || !strings.HasSuffix(name, tplSuffix):
			return nil
		}
		dir, base := filepath.Dir(name), filepath.Base(name)
		// change to dir (necessary for qtcparser to work)
		if err = os.Chdir(dir); err != nil {
			return fmt.Errorf("unable to change directory to %s: %v", dir, err)
		}
		// load, parse, generate
		src, err := ioutil.ReadFile(name)
		if err != nil {
			return fmt.Errorf("unable to load %s: %v", name, err)
		}
		buf := new(bytes.Buffer)
		if err = qtcparser.Parse(buf, bytes.NewReader(src), base, filepath.Base(dir)); err != nil {
			return fmt.Errorf("could not parse %s: %v", name, err)
		}
		// format and write to disk
		out, err := format.Source(buf.Bytes())
		if err != nil {
			return fmt.Errorf("could not format %s: %v", name, err)
		}
		outpath := filepath.Join(dir, strings.TrimSuffix(base, tplSuffix)+genSuffix)
		return ioutil.WriteFile(outpath, out, 0644)
	}
}
