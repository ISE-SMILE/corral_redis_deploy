package main

import (
	"github.com/ISE-SMILE/corral/api"
	"os"
	"testing"
)

func mainRunner(args ...string) (api.Plugin, error) {
	rescueStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	os.Args = append([]string{"testing"}, args...)
	go main()

	plugin := api.Plugin{}

	err := plugin.Interact(r)
	w.Close()
	os.Stdout = rescueStdout

	return plugin, err
}

func Test_PluginAPI(t *testing.T) {
	p, err := mainRunner()

	if err != nil {
		t.Error(err)
	}

	if !p.IsConnected() {
		t.Failed()
	}
}
