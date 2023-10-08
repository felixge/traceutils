package pprof

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPPROFWall(t *testing.T) {
	exampleTrace, err := os.ReadFile(filepath.Join("..", "..", "testdata", "1.21", "fgprof.trace"))
	require.NoError(t, err)

	var out bytes.Buffer
	err = Convert(bytes.NewReader(exampleTrace), &out, Options{})
	require.NoError(t, err)

	p, err := profile.Parse(&out)
	require.NoError(t, err)

	assert.Equal(t, 1784*time.Millisecond, round(samplesDuration(samplesWithFunc(p, "main.slowNetworkRequest")), time.Millisecond))
	assert.Equal(t, 836*time.Millisecond, round(samplesDuration(samplesWithFunc(p, "main.cpuIntensiveTask")), time.Millisecond))
	assert.Equal(t, 315*time.Millisecond, round(samplesDuration(samplesWithFunc(p, "main.weirdFunction")), time.Millisecond))
}

func samplesWithFunc(p *profile.Profile, fn string) (samples []*profile.Sample) {
outer:
	for _, s := range p.Sample {
		for _, l := range s.Location {
			for _, ln := range l.Line {
				if ln.Function.Name == fn {
					samples = append(samples, s)
					continue outer
				}
			}
		}
	}
	return
}

func samplesDuration(samples []*profile.Sample) (d time.Duration) {
	for _, s := range samples {
		d += time.Duration(s.Value[0])
	}
	return
}

func round(d, precision time.Duration) time.Duration {
	return d / precision * precision
}
