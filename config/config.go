package config

type Config struct {
	Label       string   `toml:"label"`
	RootPath    string   `toml:"root_path"`
	ExecCommand []string `toml:"exec_command"`
	IgnoreList  []string `toml:"ignore_list"`
}

func ReadConfig(path string) {
	//Read a config
}
