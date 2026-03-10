package config

var (
	Config config
)

func init() {
	err := Config.load()
	if err != nil {
		panic(err)
	}
}
