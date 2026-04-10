package config

type config struct {
	Limits  limits  `json:"limits"`
	Globals globals `json:"globals"`
}

func (c *config) load() error {
	if err := c.Globals.load(); err != nil {
		return err
	}
	if err := c.Limits.load(); err != nil {
		return err
	}
	return nil
}
