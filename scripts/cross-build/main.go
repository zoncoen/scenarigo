package main

import (
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/Yamashou/gqlgenc/clientv2"

	"github.com/zoncoen/scenarigo/scripts/cross-build/gen"
)

var (
	go117   *semver.Version
	rootDir = os.Getenv("PJ_ROOT")
)

func init() {
	var err error
	go117, err = semver.NewVersion("1.17.0")
	if err != nil {
		panic(err)
	}
}

func main() {
	if err := release(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func release() error {
	vers, err := getVers(os.Getenv("GITHUB_TOKEN"))
	if err != nil {
		return fmt.Errorf("failed to get Go versions: %w", err)
	}
	vers = filterVers(vers)
	if err := build(vers); err != nil {
		return fmt.Errorf("failed to build: %w", err)
	}
	return nil
}

func getVers(token string) ([]string, error) {
	github := &gen.Client{
		Client: clientv2.NewClient(http.DefaultClient, "https://api.github.com/graphql", func(ctx context.Context, req *http.Request, gqlInfo *clientv2.GQLRequestInfo, res interface{}, next clientv2.RequestInterceptorFunc) error {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			return next(ctx, req, gqlInfo, res)
		}),
	}

	ctx := context.Background()
	first := 50
	getTags, err := github.GetTags(ctx, "golang", "go", &first)
	if err != nil {
		if handledError, ok := err.(*clientv2.ErrorResponse); ok {
			return nil, fmt.Errorf("handled error: %sn", handledError.Error())
		}
		return nil, fmt.Errorf("unhandled error: %s", err.Error())
	}

	vers := make([]string, 0, first)
	for _, node := range getTags.Repository.Refs.Nodes {
		ver := strings.TrimPrefix(node.Name, "go")
		v, err := semver.NewVersion(ver)
		if err != nil {
			continue
		}
		if !v.LessThan(go117) {
			vers = append(vers, ver)
		}
	}
	return vers, nil
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

func build(vers []string) error {
	if err := os.Mkdir(fmt.Sprintf("%s/assets", rootDir), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	checksum, err := os.OpenFile(fmt.Sprintf("%s/assets/checksums.txt", rootDir), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer checksum.Close()

	tmpl := template.New("config").Delims("<<", ">>")
	tmpl, err = tmpl.Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	for _, ver := range vers {
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
		sum, err := os.OpenFile(fmt.Sprintf("%s/dist/checksums.txt", rootDir), os.O_RDONLY, 0o666)
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
