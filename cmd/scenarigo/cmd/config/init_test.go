package config

import (
	"os"
	"testing"
)

func TestInitRun(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		wd, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Chdir(wd); err != nil {
				t.Fatal(err)
			}
		})
		dir := t.TempDir()
		if err := os.Chdir(dir); err != nil {
			t.Fatal(err)
		}
		if err := initRun(nil, []string{}); err != nil {
			t.Fatalf("failed to init: %s", err)
		}
		b, err := os.ReadFile(DefaultConfigFileName)
		if err != nil {
			t.Fatalf("failed to read the created file: %s", err)
		}
		if got, expect := string(b), string(defaultConfig); got != expect {
			t.Errorf("\n=== expect ===\n%s\n=== got ===\n%s\n", expect, got)
		}
	})
	t.Run("failure", func(t *testing.T) {
		t.Run("already exist", func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.Chdir(wd); err != nil {
					t.Fatal(err)
				}
			})
			dir := t.TempDir()
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			if err := initRun(nil, []string{}); err != nil {
				t.Fatalf("failed to init: %s", err)
			}
			if err := initRun(nil, []string{}); err == nil {
				t.Fatal("should fail if already exists")
			}
		})
	})
}
