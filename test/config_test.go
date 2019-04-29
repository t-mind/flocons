package test

import (
	"testing"

	"github.com/macq/flocons"
)

func TestSimpleConfig(t *testing.T) {
	config, err := flocons.NewConfigFromJson([]byte("{\"namespace\": \"flocons\" }"))
	if err != nil {
		t.Errorf("Could not parse config %s", err)
	} else {
		if config.Namespace != "flocons" {
			t.Errorf("Namespace %s is different than expected flocons", config.Namespace)
		}
	}
}
