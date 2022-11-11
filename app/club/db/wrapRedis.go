package db

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"strconv"
	"time"
	commonDB "vvService/commonPackge/db"
)

var (
	GameRedisClient     *redis.Client
	PersonalRedisClient *redis.Client
)

// 连接Redis
func ConnectGameRedis(ip, pwd string, port int, index int) error {
	GameRedisClient = redis.NewClient(&redis.Options{
		Addr:     ip + ":" + strconv.Itoa(port),
		Password: pwd,
		DB:       index,
	})

	_, err := GameRedisClient.Ping().Result()
	return err
}

// 连接Redis
func ConnectPersonalRedis(ip, pwd string, port int, index int) error {
	PersonalRedisClient = redis.NewClient(&redis.Options{
		Addr:     ip + ":" + strconv.Itoa(port),
		Password: pwd,
		DB:       index,
	})

	_, err := PersonalRedisClient.Ping().Result()
	return err
}

func LoadPlayerHead(playerID int64) (headDomain, headPath string, nick string, err error) {
	key := fmt.Sprintf("%s%d", commonDB.HKeyPlayerInfo, playerID)

	cmdRes := PersonalRedisClient.HMGet(key,
		commonDB.FieldPlayerInfoHeadDomain, commonDB.FieldPlayerInfoHeadPath, commonDB.FieldPlayerInfoNick)

	if len(cmdRes.Val()) > 0 && cmdRes.Val()[0] != nil {
		headDomain = cmdRes.Val()[0].(string)
	}
	if len(cmdRes.Val()) > 1 && cmdRes.Val()[1] != nil {
		headPath = cmdRes.Val()[1].(string)
	}
	if len(cmdRes.Val()) > 2 && cmdRes.Val()[2] != nil {
		nick = cmdRes.Val()[2].(string)
	}
	err = cmdRes.Err()
	return
}

func WriteLastStocktaking(clubID int32, time_ int64) error {

	key := fmt.Sprintf("%s%d", commonDB.HKeyClub, clubID)

	cmdRes := PersonalRedisClient.HSet(key, commonDB.LastStocktaking, time_)
	return cmdRes.Err()
}

func GetLastStocktaking(clubID int32) (int64, error) {

	key := fmt.Sprintf("%s%d", commonDB.HKeyClub, clubID)

	cmdRes := PersonalRedisClient.HGet(key, commonDB.LastStocktaking)
	return cmdRes.Int64()
}

func WriteClubMengZhuID(clubID, mzID int32) error {

	key := fmt.Sprintf("%s%d", commonDB.HKeyClub, clubID)

	cmdRes := PersonalRedisClient.HSet(key, commonDB.FieldMZID, mzID)
	return cmdRes.Err()
}

func GetClubMengZhuID(clubID int32) (string, error) {

	key := fmt.Sprintf("%s%d", commonDB.HKeyClub, clubID)

	cmdRes := PersonalRedisClient.HGet(key, commonDB.FieldMZID)
	return cmdRes.Val(), cmdRes.Err()
}

func WriteOrDelClubActivityPlayer(clubID int32, data string) error {

	key := fmt.Sprintf("%s%d", commonDB.KeyClubActivityPlayer, clubID)

	if len(data) > 0 {
		cmdRes := PersonalRedisClient.Set(key, data, time.Hour*24*7)
		return cmdRes.Err()
	}
	cmdRes := PersonalRedisClient.Del(key)
	return cmdRes.Err()
}

func WriteLastClubActivityRule(clubID int32, data string) error {
	key := fmt.Sprintf("%s%d", commonDB.KeyLastClubActivity, clubID)

	cmdRes := PersonalRedisClient.Set(key, data, time.Hour*24*7)
	return cmdRes.Err()
}

func GetLastClubActivityRule(clubID int32) ([]byte, error) {
	key := fmt.Sprintf("%s%d", commonDB.KeyLastClubActivity, clubID)

	cmdRes := PersonalRedisClient.Get(key)
	return cmdRes.Bytes()
}

func GetClubActivity(clubID int32) ([]byte, error) {

	key := fmt.Sprintf("%s%d", commonDB.KeyClubActivityPlayer, clubID)

	cmdRes := PersonalRedisClient.Get(key)
	return cmdRes.Bytes()
}
