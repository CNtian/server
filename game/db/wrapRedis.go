package db

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"qpGame/localConfig"
	"strconv"
	"time"
)

const (
	HKeyPlayerInfo       = "p"
	FieldPlayerInfoG     = "game" // {"g":游戏服务名称,"t":tableNumber,"p":123}
	FieldPlayerInfoToken = "token"
	//FieldPlayerInfoHead   = "head"
	FieldPlayerInfoHeadDomain = "headD"
	FieldPlayerInfoHeadPath   = "headP"
	FieldPlayerInfoNick       = "nick"
	FieldPlayerInfoSex        = "sex"
	FieldPlayerPowerLevel     = "power"
)

const (
	HKeyTable         = "t"
	FieldTableSource  = "s"
	FieldMZClubID     = "mz"
	FieldClubIDPlayID = "p"
	FieldMaxPlayers   = "max"
	FieldGameID       = "gID"
	FieldPlayer       = "pr"
	FieldCreatTime    = "ct"
	FieldGameRule     = "gr"
	FieldPlayRule     = "playr"
)

const HKeyMutex = "mutex"

type PlayerGameIntro struct {
	GID    string `json:"gID"`
	Table  int32  `json:"table"`
	PlayID int32  `json:"playID"`
	Time   int64  `json:"time"`
}

var (
	GameRedisClient     *redis.Client
	PersonalRedisClient *redis.Client
)

// 连接Redis
func ConnectRedis(ip, pwd string, port int, index int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     ip + ":" + strconv.Itoa(port),
		Password: pwd,
		DB:       index,
	})

	_, err := client.Ping().Result()
	return client, err
}

func StorePlayerGameIntro(playerID int64, tableNumber, playID int32) (bool, error) {

	key := fmt.Sprintf("%s%d", HKeyPlayerInfo, playerID)
	valueData, _ := json.Marshal(&PlayerGameIntro{
		GID:    localConfig.GetConfig().ID,
		Table:  tableNumber,
		PlayID: playID,
		Time:   time.Now().Unix()},
	)
	cmdRes := PersonalRedisClient.HSetNX(key, FieldPlayerInfoG, string(valueData))
	return cmdRes.Val(), cmdRes.Err()
}

func LoadPlayerGameIntro(playerID int64) (string, error) {
	key := fmt.Sprintf("%s%d", HKeyPlayerInfo, playerID)

	cmdRes := PersonalRedisClient.HGet(key, FieldPlayerInfoG)
	return cmdRes.Val(), cmdRes.Err()
}

// ():是否删除成功,错误
func RemovePlayerGameIntro(playerID int64) (bool, error) {

	value, err := LoadPlayerGameIntro(playerID)
	if err != nil && err != redis.Nil {
		return false, err
	}
	if len(value) < 1 {
		return true, nil
	}

	gameIntro := PlayerGameIntro{}
	err = json.Unmarshal([]byte(value), &gameIntro)
	if err != nil {
		return false, err
	}

	// 小心同一个游戏退出- 进入
	if gameIntro.GID != localConfig.GetConfig().ID {
		return false, fmt.Errorf("GID not same. G:%s CG%s", gameIntro.GID, localConfig.GetConfig().ID)
	}

	key := fmt.Sprintf("%s%d", HKeyPlayerInfo, playerID)

	cmdRes := PersonalRedisClient.HDel(key, FieldPlayerInfoG)
	return true, cmdRes.Err()
}

// ():head,nick,sex
func GetPlayerIntro(playerID int64) (string, string, int32, error) {
	key := fmt.Sprintf("%s%d", HKeyPlayerInfo, playerID)

	cmdRes := PersonalRedisClient.HMGet(key, FieldPlayerInfoHeadDomain, FieldPlayerInfoHeadPath, FieldPlayerInfoNick, FieldPlayerInfoSex)

	if cmdRes.Err() != nil {
		return "", "", 0, cmdRes.Err()
	}
	if len(cmdRes.Val()) > 3 {
		var (
			head, nick string
			sex        int32
			//powerMap   map[int32]int32
		)
		if cmdRes.Val()[0] != nil {
			head, _ = cmdRes.Val()[0].(string)
		}
		if cmdRes.Val()[1] != nil {
			temp, _ := cmdRes.Val()[1].(string)
			head += temp
		}
		if cmdRes.Val()[2] != nil {
			nick, _ = cmdRes.Val()[2].(string)
		}
		if cmdRes.Val()[3] != nil {
			temp, _ := strconv.Atoi(cmdRes.Val()[3].(string))
			sex = int32(temp)
		}

		return head, nick, sex, nil
	}
	return "", "", 0, redis.Nil
}

func GetPlayerPower(playerID int64) (map[string]int32, error) {
	key := fmt.Sprintf("%s%d", HKeyPlayerInfo, playerID)

	cmdRes := PersonalRedisClient.HGet(key, FieldPlayerPowerLevel)
	if cmdRes.Err() != nil {
		return nil, cmdRes.Err()
	}

	powerMap := make(map[string]int32)
	err := json.Unmarshal([]byte(cmdRes.Val()), &powerMap)
	if err != nil {
		fmt.Println(err.Error())
	}

	return powerMap, nil
}

func StoreTableInfo(tableNum, MZClubID int32, clubPlayID int64, maxPlayers, gameID int32, playRule, gameRule string) error {
	key := fmt.Sprintf("%s%d", HKeyTable, tableNum)

	cmdRes := GameRedisClient.HMSet(key,
		FieldTableSource, localConfig.GetConfig().ID,
		FieldMZClubID, MZClubID,
		FieldClubIDPlayID, clubPlayID,
		FieldMaxPlayers, maxPlayers,
		FieldGameID, gameID,
		FieldCreatTime, time.Now().Unix(),
		FieldGameRule, gameRule,
		FieldPlayRule, playRule)
	return cmdRes.Err()
}

func UpdateTablePlayer(tableID int32, seatArr []int64) error {
	js, _ := json.Marshal(seatArr)
	key := fmt.Sprintf("%s%d", HKeyTable, tableID)

	cmdRes := GameRedisClient.HSet(key, FieldPlayer, js)
	return cmdRes.Err()
}

func RemoveTableInfo(tableNum int32) error {
	key := fmt.Sprintf("%s%d", HKeyTable, tableNum)

	cmdRes := GameRedisClient.Del(key)
	return cmdRes.Err()
}

// 获取玩家互斥
func GetPlayerMutex(playerID int64) (map[int64]bool, error) {

	members := make(map[int64]bool)

	// SMEMBERS
	key := fmt.Sprintf("%s%d", HKeyMutex, playerID)
	cmdRes := PersonalRedisClient.SMembers(key)
	if cmdRes.Err() != nil {
		return members, cmdRes.Err()
	}
	for _, v := range cmdRes.Val() {
		tempInt64, _ := strconv.Atoi(v)
		members[int64(tempInt64)] = false
	}
	return members, nil
}

const (
	// 最大同桌数
	HKEY_MaxTZCount = "h_date_"
)

type PlayerTongZhuoCount struct {
	UID   int64
	Value int
}

func GetMaxTongZhuo(dateKey string, uid int64, playerArr *[]PlayerTongZhuoCount) error {

	hKey := make([]string, 0, len(*playerArr)*2)
	for _, v := range *playerArr {
		hKey = append(hKey, fmt.Sprintf("%d_%d", uid, v.UID))
		hKey = append(hKey, fmt.Sprintf("%d_%d", v.UID, uid))
	}
	if len(hKey) < 1 {
		return nil
	}
	cmdRes := GameRedisClient.HMGet(dateKey, hKey...)
	if cmdRes.Err() != nil {
		return cmdRes.Err()
	}

	val := cmdRes.Val()
	for i, j := 0, 0; i < len(val) && j < len(*playerArr); j++ {
		str1, ok1 := val[i].(string)
		str2, ok2 := val[i+1].(string)
		if ok1 {
			temp, _ := strconv.Atoi(str1)
			(*playerArr)[j].Value += temp
		}
		if ok2 {
			temp, _ := strconv.Atoi(str2)
			(*playerArr)[j].Value += temp
		}
		i += 2
	}
	return nil
}
