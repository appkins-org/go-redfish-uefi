package config

import (
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/spf13/viper"
)

const ipxePatchDefault = "set user-class iPXE"

type UnifiConfig struct {
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	Site     string `yaml:"site" mapstructure:"site"`
	Device   string `yaml:"device" mapstructure:"device"`
	Insecure bool   `yaml:"insecure" mapstructure:"insecure"`
}

type TftpConfig struct {
	Address       string `yaml:"address" mapstructure:"address"`
	Port          int    `yaml:"port" mapstructure:"port"`
	RootDirectory string `yaml:"root_directory" mapstructure:"root_directory"`
	IpxePatch     string `yaml:"ipxe_patch" mapstructure:"ipxe_patch"`
}

type IpxeUrl struct {
	Address string `yaml:"address" mapstructure:"address"`
	Port    int    `yaml:"port" mapstructure:"port"`
	Scheme  string `yaml:"scheme" mapstructure:"scheme"`
	Path    string `yaml:"path" mapstructure:"path"`
}

type DhcpConfig struct {
	Interface         string  `yaml:"interface" mapstructure:"interface"`
	Address           string  `yaml:"address" mapstructure:"address"`
	Port              int     `yaml:"port" mapstructure:"port"`
	IpxeBinaryUrl     IpxeUrl `yaml:"ipxe_binary_url" mapstructure:"ipxe_binary_url"`
	IpxeHttpUrl       IpxeUrl `yaml:"ipxe_http_url" mapstructure:"ipxe_http_url"`
	IpxeHttpScriptURL string  `yaml:"ipxe_http_script_url" mapstructure:"ipxe_http_script_url"`
	TftpAddress       string  `yaml:"tftp_address" mapstructure:"tftp_address"`
	TftpPort          int     `yaml:"tftp_port" mapstructure:"tftp_port"`
}

type Config struct {
	Address         string      `yaml:"address" mapstructure:"address"`
	Port            int         `yaml:"port" mapstructure:"port"`
	Unifi           UnifiConfig `yaml:"unifi" mapstructure:"unifi"`
	Tftp            TftpConfig  `yaml:"tftp" mapstructure:"tftp"`
	Dhcp            DhcpConfig  `yaml:"dhcp" mapstructure:"dhcp"`
	LogLevel        string      `yaml:"log_level" mapstructure:"log_level"`
	BackendFilePath string      `yaml:"backend_file_path" mapstructure:"backend_file_path"`
	Log             logr.Logger `yaml:"-" mapstructure:"-"`
}

func NewConfig() (conf *Config, err error) {
	conf = &Config{}

	defaultIp, defaultIface, err := GetLocalIP()
	if err != nil {
		defaultIp = "0.0.0.0"
		defaultIface = "eth0"
	}

	viper.SetConfigName("redfish")

	viper.AddConfigPath("/app/")
	viper.AddConfigPath("/config/")
	viper.AddConfigPath(".")

	viper.SetDefault("address", "0.0.0.0")
	viper.SetDefault("port", 8080)
	viper.SetDefault("backend_file_path", "backend.yaml")
	viper.SetDefault("unifi.username", "")
	viper.SetDefault("unifi.password", "")
	viper.SetDefault("unifi.endpoint", "")
	viper.SetDefault("unifi.site", "default")
	viper.SetDefault("unifi.device", "")
	viper.SetDefault("unifi.insecure", true)
	viper.SetDefault("tftp.address", "0.0.0.0")
	viper.SetDefault("tftp.port", 69)
	viper.SetDefault("tftp.root_directory", "/tftpboot")
	viper.SetDefault("tftp.ipxe_patch", ipxePatchDefault)

	viper.SetDefault("dhcp.interface", defaultIface)
	viper.SetDefault("dhcp.address", "0.0.0.0")
	viper.SetDefault("dhcp.port", 67)
	viper.SetDefault("dhcp.ipxe_http_script_url", "")
	viper.SetDefault("dhcp.ipxe_binary_url.address", defaultIp)
	viper.SetDefault("dhcp.ipxe_binary_url.port", 80)
	viper.SetDefault("dhcp.ipxe_binary_url.scheme", "http")
	viper.SetDefault("dhcp.ipxe_binary_url.path", "/ipxe.efi")
	viper.SetDefault("dhcp.ipxe_http_url.address", defaultIp)
	viper.SetDefault("dhcp.ipxe_http_url.port", 80)
	viper.SetDefault("dhcp.ipxe_http_url.scheme", "http")
	viper.SetDefault("dhcp.ipxe_http_url.path", "/ipxe.efi")
	viper.SetDefault("dhcp.tftp_address", defaultIp)
	viper.SetDefault("dhcp.tftp_port", 69)

	viper.SetDefault("log_level", "info")

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

	conf.Log = defaultLogger(conf.LogLevel)

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

func GetLocalIP() (string, string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", "", err
	}
	// handle err
	for _, i := range ifaces {
		addrs, err := i.Addrs()

		if err != nil {
			return "", "", err
		}

		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if !ip.IsLoopback() {
				if ip.To4() != nil {
					return ip.String(), i.Name, nil
				}
			}
		}
	}

	return "", "", nil
}

// defaultLogger uses the slog logr implementation.
func defaultLogger(level string) logr.Logger {
	// source file and function can be long. This makes the logs less readable.
	// truncate source file and function to last 3 parts for improved readability.
	customAttr := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.SourceKey {
			ss, ok := a.Value.Any().(*slog.Source)
			if !ok || ss == nil {
				return a
			}
			f := strings.Split(ss.Function, "/")
			if len(f) > 3 {
				ss.Function = filepath.Join(f[len(f)-3:]...)
			}
			p := strings.Split(ss.File, "/")
			if len(p) > 3 {
				ss.File = filepath.Join(p[len(p)-3:]...)
			}

			return a
		}

		return a
	}
	opts := &slog.HandlerOptions{AddSource: true, ReplaceAttr: customAttr}
	switch level {
	case "debug":
		opts.Level = slog.LevelDebug
	default:
		opts.Level = slog.LevelInfo
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, opts))

	return logr.FromSlogHandler(log.Handler())
}
