package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	releaseVer = os.Getenv("RELEASE_VERSION")
	ver        = os.Getenv("GO_VERSION")
	rootDir    = os.Getenv("PJ_ROOT")
)

func main() {
	if err := release(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func release() error {
	if err := exec.Command("docker", "pull", fmt.Sprintf("ghcr.io/gythialy/golang-cross:v%s", ver)).Run(); err != nil {
		fmt.Printf("failed to pull image: %s\n", err)
		return nil
	}
	if err := build(ver); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	return nil
}

func filterVers(candidates []string) []string {
	vers := make([]string, 0, len(candidates))
	for _, ver := range candidates {
		if err := exec.Command("docker", "pull", fmt.Sprintf("ghcr.io/gythialy/golang-cross:v%s", ver)).Run(); err == nil {
			vers = append(vers, ver)
		}
	}
	return vers
}

//go:embed templates/goreleaser.yml.tmpl
var tmplBytes []byte

func build(ver string) error {
	if err := os.Mkdir(fmt.Sprintf("%s/assets", rootDir), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmpl := template.New("config").Delims("<<", ">>")
	tmpl, err = tmpl.Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/.goreleaser.yml", rootDir), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open .goreleaser.yml: %w", err)
	}
	defer f.Close()
	if err := tmpl.Execute(f, map[string]string{
		"GoVersion": ver,
	}); err != nil {
		return fmt.Errorf("failed to create .goreleaser.yml: %w", err)
	}
	if err := goreleaser(ver); err != nil {
		return fmt.Errorf("goreleaser failed (go%s): %w", ver, err)
	}

	// move results
	checksumFilename := fmt.Sprintf("%s/assets/scenarigo_%s_go%s_checksums.txt", rootDir, releaseVer, ver)
	checksum, err := os.OpenFile(filepath.Join(rootDir, "assets", checksumFilename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer checksum.Close()
	sum, err := os.OpenFile(filepath.Join(rootDir, "dist", checksumFilename), os.O_RDONLY, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer sum.Close()
	if _, err := io.Copy(checksum, sum); err != nil {
		return fmt.Errorf("failed to copy checksums.txt: %w", err)
	}
	archives, err := filepath.Glob(fmt.Sprintf("%s/dist/*.tar.gz", rootDir))
	if err != nil {
		return fmt.Errorf("failed to find archives: %w", err)
	}
	for _, archive := range archives {
		dst, err := os.OpenFile(fmt.Sprintf("%s/assets/%s", rootDir, filepath.Base(archive)), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer dst.Close()
		src, err := os.OpenFile(archive, os.O_RDONLY, 0o666)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer src.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return fmt.Errorf("failed to copy archive: %w", err)
		}
	}

	return nil
}

func goreleaser(ver string) error {
	out, err := exec.Command(
		"docker", "run", "--rm", "--privileged",
		"-v", fmt.Sprintf("%s:/scenarigo", rootDir),
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-w", "/scenarigo",
		fmt.Sprintf("ghcr.io/gythialy/golang-cross:v%s", ver),
		"--skip-publish", "--rm-dist", "--release-notes", ".CHANGELOG.md",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s:\n%s", err, out)
	}
	return nil
}
