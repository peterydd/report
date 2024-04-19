package config

import (
	"github.com/davecgh/go-spew/spew"
	"testing"
)

func TestNewConfig(t *testing.T) {
	conf, err := NewConfig()
	if err != nil {
		spew.Dump(err)
	}
	spew.Dump(conf)
}
