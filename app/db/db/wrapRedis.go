package db

import (
	"fmt"
	"github.com/go-redis/redis/v7"
	"strconv"
	commonDB "vvService/commonPackge/db"
)

var (
	GameRedisClient     *redis.Client
	PersonalRedisClient *redis.Client
)

// 连接Redis
func ConnectRedis(ip, pwd string, port int, index int) (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     ip + ":" + strconv.Itoa(port),
		Password: pwd,
		DB:       index,
	})

	_, err := c.Ping().Result()
	return c, err
}

func RedisPutMaxTongZhuoPlayer(date int, uidArr []int64) error {

	key := fmt.Sprintf("%s%d", commonDB.HKEY_MaxTZCount, date)

	pipe := GameRedisClient.Pipeline()

	for i, v := range uidArr {
		for j := i + 1; j < len(uidArr); j++ {
			filed := fmt.Sprintf("%d_%d", v, uidArr[j])
			pipe.HIncrBy(key, filed, 1)
		}
	}
	_, err := pipe.Exec()
	if err != nil {
		return err
	}
	return err
}

func RedisDeleteTongZhuoKey(date int) {
	key := fmt.Sprintf("%s%d", commonDB.HKEY_MaxTZCount, date)
	GameRedisClient.Del(key)
}

// ():段位,经验,晋级W,晋级L
//func GetPlayerDuanWei(uid int64) (int, int, int, int, error) {

//var duanWei, exp, finalsW, finalsL int
//key := fmt.Sprintf("%s%d", commonDB.HKeyPlayerInfo, uid)
//cmdRes := PersonalRedisClient.HMGet(key,
//	commonDB.FieldPlayerInfoDuanWei, commonDB.FieldPlayerInfoEXP, commonDB.FieldPlayerInfoFinalsW, commonDB.FieldPlayerInfoFinalsL)
//if cmdRes.Err() == nil {
//	if len(cmdRes.Val()) > 3 {
//		if cmdRes.Val()[0] != nil {
//			duanWei, _ = strconv.Atoi(cmdRes.Val()[0].(string))
//		}
//		if cmdRes.Val()[1] != nil {
//			exp, _ = strconv.Atoi(cmdRes.Val()[1].(string))
//		}
//		if cmdRes.Val()[2] != nil {
//			finalsW, _ = strconv.Atoi(cmdRes.Val()[2].(string))
//		}
//		if cmdRes.Val()[3] != nil {
//			finalsL, _ = strconv.Atoi(cmdRes.Val()[3].(string))
//		}
//	}
//}
//return duanWei, exp, finalsW, finalsL, cmdRes.Err()
//}

func SetPlayerDuanWei(uid int64, duanWei, exp, finalW, finalL int) error {

	//key := fmt.Sprintf("%s%d", commonDB.HKeyPlayerInfo, uid)
	//cmdRes := PersonalRedisClient.HMSet(key,
	//	commonDB.FieldPlayerInfoDuanWei, duanWei,
	//	commonDB.FieldPlayerInfoEXP, exp,
	//	commonDB.FieldPlayerInfoFinalsW, finalW,
	//	commonDB.FieldPlayerInfoFinalsL, finalL)
	//
	//return cmdRes.Err()
	return nil
}
