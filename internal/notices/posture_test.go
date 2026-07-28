package notices

import (
	"testing"
	"time"
)

func TestPostureScore(t *testing.T) {
	cases := []struct {
		name      string
		d         Digest
		wantGrade string
		minScore  int
		maxScore  int
	}{
		{
			name:      "clean workstation",
			d:         Digest{RefreshedAt: time.Now()},
			wantGrade: "A", minScore: 100, maxScore: 100,
		},
		{
			name:      "a few updates only",
			d:         Digest{RefreshedAt: time.Now(), Updates: 4},
			wantGrade: "A", minScore: 90, maxScore: 100,
		},
		{
			name:      "one HIGH/CRITICAL CVE",
			d:         Digest{RefreshedAt: time.Now(), CVETools: 1, CVEHighOrCritical: 1},
			wantGrade: "B", minScore: 85, maxScore: 90,
		},
		{
			name:      "a leaked critical secret dominates",
			d:         Digest{RefreshedAt: time.Now(), Secrets: 1, SecretsCritical: 1},
			wantGrade: "B", minScore: 70, maxScore: 80,
		},
		{
			name:      "loaded workstation (real-world bad)",
			d:         Digest{RefreshedAt: time.Now(), CVETools: 4, CVEHighOrCritical: 4, Secrets: 6, SecretsCritical: 6, Updates: 4},
			wantGrade: "F", minScore: 0, maxScore: 0,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := c.d.PostureScore()
			if p.Grade != c.wantGrade {
				t.Errorf("grade = %q, want %q (score %d)", p.Grade, c.wantGrade, p.Score)
			}
			if p.Score < c.minScore || p.Score > c.maxScore {
				t.Errorf("score = %d, want in [%d,%d]", p.Score, c.minScore, c.maxScore)
			}
		})
	}
}
