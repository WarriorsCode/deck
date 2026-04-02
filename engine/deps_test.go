package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/warriorscode/deck/config"
)

func TestDepAlreadyRunning(t *testing.T) {
	deps := config.MapOf[config.Dep](
		"myservice", config.Dep{Check: "true", Start: config.StringOrList{"false"}},
	)
	err := EnsureDeps(context.Background(), ".", deps, nil)
	require.NoError(t, err)
}

func TestDepFirstStrategyWorks(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	deps := config.MapOf[config.Dep](
		"myservice", config.Dep{Check: "test -f " + marker, Start: config.StringOrList{"touch " + marker}},
	)
	err := EnsureDeps(context.Background(), ".", deps, nil)
	require.NoError(t, err)
	_, err = os.Stat(marker)
	require.NoError(t, err)
}

func TestDepFallbackStrategy(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	deps := config.MapOf[config.Dep](
		"myservice", config.Dep{
			Check: "test -f " + marker,
			Start: config.StringOrList{"false", "touch " + marker},
		},
	)
	err := EnsureDeps(context.Background(), ".", deps, nil)
	require.NoError(t, err)
}

func TestDepAllStrategiesFail(t *testing.T) {
	deps := config.MapOf[config.Dep](
		"myservice", config.Dep{Check: "false", Start: config.StringOrList{"true"}},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := EnsureDeps(ctx, ".", deps, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "myservice")
}
