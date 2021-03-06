package gomplate

import (
	"context"
	"testing"
	"text/template"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

func TestNewPlugin(t *testing.T) {
	ctx := context.TODO()
	in := "foo"
	_, err := newPlugin(ctx, in)
	assert.ErrorContains(t, err, "")

	in = "foo=/bin/bar"
	out, err := newPlugin(ctx, in)
	assert.NilError(t, err)
	assert.Equal(t, "foo", out.name)
	assert.Equal(t, "/bin/bar", out.path)
}

func TestBindPlugins(t *testing.T) {
	ctx := context.TODO()
	fm := template.FuncMap{}
	in := []string{}
	err := bindPlugins(ctx, in, fm)
	assert.NilError(t, err)
	assert.DeepEqual(t, template.FuncMap{}, fm)

	in = []string{"foo=bar"}
	err = bindPlugins(ctx, in, fm)
	assert.NilError(t, err)
	assert.Check(t, cmp.Contains(fm, "foo"))

	err = bindPlugins(ctx, in, fm)
	assert.ErrorContains(t, err, "already bound")
}

func TestBuildCommand(t *testing.T) {
	ctx := context.TODO()
	data := []struct {
		plugin   string
		args     []string
		expected []string
	}{
		{"foo=foo", nil, []string{"foo"}},
		{"foo=foo", []string{"bar"}, []string{"foo", "bar"}},
		{"foo=foo.bat", nil, []string{"cmd.exe", "/c", "foo.bat"}},
		{"foo=foo.cmd", []string{"bar"}, []string{"cmd.exe", "/c", "foo.cmd", "bar"}},
		{"foo=foo.ps1", []string{"bar", "baz"}, []string{"pwsh", "-File", "foo.ps1", "bar", "baz"}},
	}
	for _, d := range data {
		p, err := newPlugin(ctx, d.plugin)
		assert.NilError(t, err)
		name, args := p.buildCommand(d.args)
		actual := append([]string{name}, args...)
		assert.DeepEqual(t, d.expected, actual)
	}
}
