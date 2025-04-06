package core

type FernCredentials struct {
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Username  string `yaml:"username"`
}

type FernConfig struct {
	Region      string            `yaml:"region"`
	Credentials []FernCredentials `yaml:"credentials"`
	Keys        []KmsKey          `yaml:"keys"`
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

const (
	ZeroAccountId string = "000000000000"
)
