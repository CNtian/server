package localConfig

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
)

var (
	config     *LocalConfig
	configPath string
)

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

type TableNumberRange struct {
	Begin int `json:"begin"`
	End   int `json:"end"`
}

type LocalConfig struct {
	ID             string           `json:"ID"`
	PprofPort      string           `json:"pprofPort"`
	RabbitMQ       RabbitMQJson     `json:"RabbitMQ"`
	Redis          RedisJson        `json:"Redis"`
	MongodbInfo    MongodbJson      `json:"MongoDB"`
	SupportPlaying []PlayingInfo    `json:"SupportPlaying"`
	TableNumRange  TableNumberRange `json:"TableNumberRange"`
	IsTestPai      bool             `json:"TestPai"`
}

type PlayingInfo struct {
	PlayingID int32  `json:"playingID"`
	Name      string `json:"name"`
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

	playInfo := map[int32]string{}
	for _, v := range config.SupportPlaying {
		if _, ok := playInfo[v.PlayingID]; ok == true {
			return fmt.Errorf("find repeat playing ID.id:=%d", v.PlayingID)
		}
	}

	configPath = path
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
	}

	return config, nil
}

func ReloadConfig() {
	config = nil
	if _, err := LoadConfig(configPath); err != nil {
		glog.Warning("ReloadConfig() failed. err:=", err.Error())
	}
}

func GetConfig() *LocalConfig {
	return config
}
