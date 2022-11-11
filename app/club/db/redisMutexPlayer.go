package db

import (
	"fmt"
	commonDB "vvService/commonPackge/db"
)

// 存入 互斥
func PutMutexGroupToRedis(playerID []int64) error {

	pipe := PersonalRedisClient.Pipeline()

	for _, v := range playerID {
		key := fmt.Sprintf("%s%d", commonDB.HKeyMutex, v)
		arr := make([]interface{}, 0, 10)
		for _, v1 := range playerID {
			if v == v1 {
				continue
			}
			arr = append(arr, v1)
		}
		pipe.SAdd(key, arr...)
	}
	_, err := pipe.Exec()
	if err != nil {
		return err
	}

	return nil
}

// 删除 互斥
func RemoveMutexGroupFromRedis(delPlayerID []int64) error {

	tempArr := make([]interface{}, 0, len(delPlayerID))
	for _, v1 := range delPlayerID {
		tempArr = append(tempArr, v1)
	}

	pipe := PersonalRedisClient.Pipeline()
	for _, v := range delPlayerID {
		key := fmt.Sprintf("%s%d", commonDB.HKeyMutex, v)

		pipe.SRem(key, tempArr[:]...)
	}
	_, err := pipe.Exec()
	return err
}
