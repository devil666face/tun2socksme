package dns

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"tun2socksme/pkg/fs"
	"tun2socksme/pkg/shell"
)

var (
	resolvconf = `
# Generated by tun2socksme.
# Stop tun2socksme for return default config.
nameserver %s
`
	respath      = filepath.Join("/", "etc", "resolv.conf")
	reInterfaces = regexp.MustCompile(`\((\w+)\)`)
)

type manager struct {
	listen      string
	link        string
	currentconf []byte
}

func Manager(_listen string) (*manager, error) {
	_currentconf, err := fs.ReadFile(respath)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}
	_link, _ := fs.CheckSymlink(respath)
	return &manager{
		listen:      _listen,
		currentconf: _currentconf,
		link:        _link,
	}, nil
}

func (m *manager) Set() error {
	if m.link != "" {
		if err := os.RemoveAll(respath); err != nil {
			return fmt.Errorf("failed to unlink %s: %w", respath, err)
		}
	}
	if err := render(respath, resolvconf, m.listen); err != nil {
		return fmt.Errorf("resolvconf error: %w", err)
	}
	for _, i := range getInterfaces() {
		if _, err := shell.New("resolvectl", "dns", i, m.listen).Run(); err != nil {
			return fmt.Errorf("failed to set dns %s for %s: %w", m.listen, i, err)
		}
	}
	return nil
}

func (m *manager) Revert() error {
	if m.link != "" {
		if err := os.RemoveAll(respath); err != nil {
			return fmt.Errorf("failed to unlink %s: %w", respath, err)
		}
	}
	if err := os.Symlink(m.link, respath); err != nil {
		log.Fatalf("error creating symlink: %v", err)
	}
	if err := fs.WriteFile(respath, m.currentconf); err != nil {
		return fmt.Errorf("resolvconf error: %w", err)
	}
	for _, i := range getInterfaces() {
		if _, err := shell.New("resolvectl", "revert", i).Run(); err != nil {
			return fmt.Errorf("failed to revert dns for %s: %w", i, err)
		}
	}
	return nil
}

func render(path, format, a string) error {
	if err := fs.WriteFile(path, []byte(fmt.Sprintf(format, a))); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	return nil
}

func getInterfaces() (interfaces []string) {
	out, err := shell.New("resolvectl", "dns").Run()
	if err != nil {
		log.Printf("failed to get interfaces: %v", err)
		return
	}
	for _, match := range reInterfaces.FindAllStringSubmatch(out, -1) {
		interfaces = append(interfaces, match[1])
	}
	return
}
