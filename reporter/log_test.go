package reporter

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLogRecorder(t *testing.T) {
	r := &logRecorder{}
	r.log("info")
	r.error("error")
	r.skip("skip")

	opts := []cmp.Option{
		cmp.AllowUnexported(logRecorder{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.Last().String() == ".m"
		}, cmp.Ignore()),
	}
	skipIdx := 2
	if diff := cmp.Diff(&logRecorder{
		strs:      []string{"info", "error", "skip"},
		infoIdxs:  []int{0},
		errorIdxs: []int{1},
		skipIdx:   &skipIdx,
	}, r, opts...); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}

	skipLog := "skip"
	if diff := cmp.Diff([]string{"info", "error", "skip"}, r.all()); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"info"}, r.infoLogs()); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"error"}, r.errorLogs()); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(&skipLog, r.skipLog()); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}

func TestLogRecorder_Append(t *testing.T) {
	s := &logRecorder{}
	s.log("INFO")
	s.error("ERROR")
	s.skip("SKIP")

	opts := []cmp.Option{
		cmp.AllowUnexported(logRecorder{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.Last().String() == ".m"
		}, cmp.Ignore()),
	}

	t.Run("skipIdx is nil", func(t *testing.T) {
		r := &logRecorder{}
		r.log("info")
		r.error("error")
		r.append(s)

		skipIdx := 4
		skipLog := "SKIP"
		if diff := cmp.Diff(&logRecorder{
			strs:      []string{"info", "error", "INFO", "ERROR", "SKIP"},
			infoIdxs:  []int{0, 2},
			errorIdxs: []int{1, 3},
			skipIdx:   &skipIdx,
		}, r, opts...); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"info", "error", "INFO", "ERROR", "SKIP"}, r.all()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"info", "INFO"}, r.infoLogs()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"error", "ERROR"}, r.errorLogs()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(&skipLog, r.skipLog()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("skipIdx is not nil", func(t *testing.T) {
		r := &logRecorder{}
		r.log("info")
		r.error("error")
		r.skip("skip")
		r.append(s)

		skipIdx := 2
		skipLog := "skip"
		if diff := cmp.Diff(&logRecorder{
			strs:      []string{"info", "error", "skip", "INFO", "ERROR", "SKIP"},
			infoIdxs:  []int{0, 3, 5},
			errorIdxs: []int{1, 4},
			skipIdx:   &skipIdx,
		}, r, opts...); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"info", "error", "skip", "INFO", "ERROR", "SKIP"}, r.all()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"info", "INFO", "SKIP"}, r.infoLogs()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff([]string{"error", "ERROR"}, r.errorLogs()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(&skipLog, r.skipLog()); diff != "" {
			t.Errorf("result mismatch (-want +got):\n%s", diff)
		}
	})
}
