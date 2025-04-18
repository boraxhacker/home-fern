package core

import "io"

type FernCredentials struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Username  string `yaml:"username"`
}

type FernConfig struct {
	Region      string            `yaml:"region"`
	Credentials []FernCredentials `yaml:"credentials"`
	Keys        []KmsKey          `yaml:"kms"`
	DnsDefaults DnsDefaults       `yaml:"dns"`
}

type DnsDefaults struct {
	Soa         string   `yaml:"soa"`
	NameServers []string `yaml:"nameServers"`
}

type KmsKey struct {
	KeyId string `yaml:"id"`
	Alias string `yaml:"alias"`
	Key   string `yaml:"key"`
}

type ResourceTag struct {
	Key   string
	Value string
}

type DatabaseDumper interface {
	LogKeys(writer io.Writer) error
}

const (
	ZeroAccountId string = "000000000000"
)
