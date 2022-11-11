package localConfig

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var config *LocalConfig

type RabbitMQJson struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type MongodbJson struct {
	Address string `json:"address"`
	DBName  string `json:"dbName"`
}

type RedisJson struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Password   string `json:"password"`
	LoginIndex int    `json:"loginIndex"`
	GameIndex  int    `json:"gameIndex"`
}

type LocalConfig struct {
	ID          string       `json:"ID"`
	RabbitMQ    RabbitMQJson `json:"RabbitMQ"`
	MongodbInfo MongodbJson  `json:"MongoDB"`
	Redis       RedisJson    `json:"Redis"`
}

// 读取配置
func readConfigFile(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("not find config file.%s", path)
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		return err
	}

	return nil
}

// 获取配置
func LoadConfig(configPath string) (*LocalConfig, error) {

	if config == nil {
		config = new(LocalConfig)
		err := readConfigFile(configPath)
		if err != nil {
			return config, err
		}
		if err != nil {
			return config, err
		}
	}

	return config, nil
}

func GetConfig() *LocalConfig {

	return config
}
