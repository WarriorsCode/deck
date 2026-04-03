package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEnvVarUnmarshalString(t *testing.T) {
	var env Env
	require.NoError(t, yaml.Unmarshal([]byte(`FOO: bar`), &env))
	assert.Equal(t, "bar", env["FOO"].Value)
	assert.Empty(t, env["FOO"].Script)
	assert.Empty(t, env["FOO"].File)
}

func TestEnvVarUnmarshalInterpolation(t *testing.T) {
	var env Env
	require.NoError(t, yaml.Unmarshal([]byte(`PG_HOST: "$(echo localhost)"`), &env))
	assert.Equal(t, "$(echo localhost)", env["PG_HOST"].Value)
	assert.False(t, env["PG_HOST"].IsStatic())
}

func TestEnvVarUnmarshalObject(t *testing.T) {
	var env Env
	require.NoError(t, yaml.Unmarshal([]byte(`
PG_HOST:
  file: "api/etc/omsx.conf | db.host"
PG_PASS:
  script: "echo secret"
STATIC:
  value: "hello"
`), &env))

	assert.Equal(t, "api/etc/omsx.conf | db.host", env["PG_HOST"].File)
	assert.Empty(t, env["PG_HOST"].Value)

	assert.Equal(t, "echo secret", env["PG_PASS"].Script)
	assert.Empty(t, env["PG_PASS"].Value)

	assert.Equal(t, "hello", env["STATIC"].Value)
	assert.True(t, env["STATIC"].IsStatic())
}

func TestEnvVarMixedStyles(t *testing.T) {
	var env Env
	require.NoError(t, yaml.Unmarshal([]byte(`
SIMPLE: "1"
INTERP: "$(whoami)"
FROM_FILE:
  file: "config.json | db.host"
FROM_SCRIPT:
  script: "date +%Y"
`), &env))

	assert.True(t, env["SIMPLE"].IsStatic())
	assert.False(t, env["INTERP"].IsStatic())
	assert.False(t, env["FROM_FILE"].IsStatic())
	assert.False(t, env["FROM_SCRIPT"].IsStatic())
}
