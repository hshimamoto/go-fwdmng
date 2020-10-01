// MIT License Copyright (c) 2020 Hiroshi Shimamoto
// vim: set sw=4 sts=4:
package config

import (
    "bytes"
    "io/ioutil"

    "github.com/BurntSushi/toml"
)

type Fwd struct {
    Name string
    Local string
    Remote string
}

type SSHHost struct {
    Name string
    Hostname string
    User string
    Privkey string
    Proxy string
    Fwds []Fwd
}

type Config struct {
    SSHHosts []SSHHost
}

func Load(path string) (*Config, error) {
    cfg := &Config{}
    if _, err := toml.DecodeFile(path, cfg); err != nil {
	return nil, err
    }
    return cfg, nil
}

func Save(cfg *Config, path string) error {
    buf := new(bytes.Buffer)
    if err := toml.NewEncoder(buf).Encode(cfg); err != nil {
	return err
    }
    return ioutil.WriteFile(path, buf.Bytes(), 0600)
}
