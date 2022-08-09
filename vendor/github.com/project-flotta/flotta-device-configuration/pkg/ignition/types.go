package ignition

import (
	// "github.com/coreos/ignition/v2/config/v3_2/types"
	"github.com/coreos/ignition/v2/config/v3_4_experimental/types"
)

type Config struct {
	Ignition Ignition      `json:"ignition"`
	Storage  Storage       `json:"storage,omitempty"`
	Systemd  types.Systemd `json:"systemd,omitempty"`
	Passwd   types.Passwd  `json:"passwd,omitempty"`
}

type Storage struct {
	Directories []types.Directory `json:"directories,omitempty"`
	Files       []types.File      `json:"files,omitempty"`
	Links       []types.Link      `json:"links,omitempty"`
}

type Ignition struct {
	Version string `json:"version"`
}
