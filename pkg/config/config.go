package config

import (
	"log"
	"strings"

	"github.com/appkins-org/go-redfish-uefi/api/redfish"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type UnifiConfig struct {
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	Site     string `yaml:"site" mapstructure:"site"`
	Device   string `yaml:"device" mapstructure:"device"`
}

type TftpConfig struct {
	Address       string `yaml:"address" mapstructure:"address"`
	Port          int    `yaml:"port" mapstructure:"port"`
	RootDirectory string `yaml:"root_directory" mapstructure:"root_directory"`
}

type Config struct {
	Address string                           `yaml:"address" mapstructure:"address"`
	Port    int                              `yaml:"port" mapstructure:"port"`
	Unifi   UnifiConfig                      `yaml:"unifi" mapstructure:"unifi"`
	Tftp    TftpConfig                       `yaml:"tftp" mapstructure:"tftp"`
	Systems map[string]redfish.RedfishSystem `yaml:"systems" mapstructure:"systems"`
}

func NewConfig() (conf *Config, err error) {
	conf = &Config{}

	viper.SetConfigName("redfish")

	viper.AddConfigPath("/app/")
	viper.AddConfigPath("/config/")
	viper.AddConfigPath(".")

	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("config: unable to bind env: %s", err.Error())
	}

	for _, key := range viper.AllKeys() {
		envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
		err := viper.BindEnv(key, envKey)
		if err != nil {
			log.Fatalf("config: unable to bind env: %s", err.Error())
		}
	}

	// Load the Config the first time we start the app.
	err = loadConfig(conf)
	if err != nil {
		return
	}

	// Tell viper to watch the config file.
	viper.WatchConfig()

	// Tell viper what to do when it detects the
	// config file has changed.
	viper.OnConfigChange(func(_ fsnotify.Event) {
		_ = loadConfig(conf)
	})

	return
}

func loadConfig(conf *Config) (err error) {
	// read the config file into viper and
	// handle (ignore the file) any errors
	err = viper.MergeInConfig()
	if err != nil {
		return nil
	}

	err = viper.Unmarshal(conf)
	if err != nil {
		return
	}

	return
}
