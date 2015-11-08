package vgcfg

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

func Parse(r io.Reader) (*Group, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	p := &parser{Buffer: string(data)}
	p.Prepare()
	p.Init()

	if err := p.Parse(); err != nil {
		return nil, err
	}

	p.Execute()

	if p.internalError != nil {
		return nil, p.internalError
	}

	return p.Root(), nil
}

func Dump(vgname string) (*Group, error) {
	path, err := exec.LookPath("vgcfgbackup")
	if err != nil {
		return nil, err
	}

	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return nil, err
	}

	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("vgcfg_%s.txt", hex.EncodeToString(buf[:])))

	cmd := exec.Command(path, vgname, "-f", tmp)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	defer os.Remove(tmp)

	f, err := os.Open(tmp)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Parse(f)
}
