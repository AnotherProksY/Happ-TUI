package appconfig

// AppConfig is the part of the config the TUI needs. The core is the only
// process that reads the config file; the TUI gets these values from it with
// get-config, so a system service and the TUI never disagree.
type AppConfig struct {
	SocksPort    int  `json:"socks-port"`
	HttpPort     int  `json:"http-port"`
	NoBackground bool `json:"no-background"`
	Root         bool `json:"root"`
	Service      bool `json:"service"`
}

var is_config_loaded bool = false

var application_configuration AppConfig

func GetConfig() AppConfig {
	if !is_config_loaded {
		panic("attempted to access config before loading")
	}
	return application_configuration
}

func SetConfig(config AppConfig) {
	application_configuration = config
	is_config_loaded = true
}
