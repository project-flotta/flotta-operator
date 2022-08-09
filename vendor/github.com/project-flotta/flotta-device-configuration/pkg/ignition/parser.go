package ignition

import (
	"encoding/json"
	"fmt"

	// "github.com/project-flotta/flotta-device-configuration/pkg/ignition_config"

	"github.com/project-flotta/flotta-device-configuration/pkg/ignition/source/exec"
	"github.com/project-flotta/flotta-device-configuration/pkg/ignition/source/exec/stages"
	_ "github.com/project-flotta/flotta-device-configuration/pkg/ignition/source/exec/stages/files"
	"github.com/project-flotta/flotta-device-configuration/pkg/ignition/source/log"
	"github.com/project-flotta/flotta-device-configuration/pkg/ignition/source/platform"
	"github.com/project-flotta/flotta-device-configuration/pkg/ignition/source/state"

	types_exp "github.com/coreos/ignition/v2/config/v3_4_experimental/types"
)

func ParseConfig(rawConfig string) (*types_exp.Config, error) {
	var flottaConfig Config
	err := json.Unmarshal([]byte(rawConfig), &flottaConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot parse configuration: %v", err)
	}

	res := types_exp.Config{
		Ignition: types_exp.Ignition{
			Version: flottaConfig.Ignition.Version,
		},
		Passwd: flottaConfig.Passwd,
		Storage: types_exp.Storage{
			Directories: flottaConfig.Storage.Directories,
			Files:       flottaConfig.Storage.Files,
			Links:       flottaConfig.Storage.Links,
		},
		Systemd: flottaConfig.Systemd,
	}
	return &res, nil
}

func RunConfig(cfg *types_exp.Config) error {

	logger := log.New(true)
	defer logger.Close()

	state, err := state.Load("/tmp/flotta_ignition")
	if err != nil {
		return fmt.Errorf("Cannot read state: %v", err)
	}

	platformConfig := platform.MustGet("file")
	fetcher, err := platformConfig.NewFetcherFunc()(&logger)
	if err != nil {
		return fmt.Errorf("Failed to generate fetcher: %v", err)
	}

	cfgFetcher := exec.ConfigFetcher{
		Logger:  &logger,
		Fetcher: &fetcher,
		State:   &state,
	}

	finalCfg, err := cfgFetcher.RenderConfig(*cfg)
	if err != nil {
		return fmt.Errorf("Failed on config reader: %v", err)
	}

	stage := stages.Get("files").Create(&logger, "/", fetcher, &state)
	if err := stage.Apply(finalCfg, true); err != nil {
		return fmt.Errorf("Running stage files failed: %v", err)
	}
	return nil
}
