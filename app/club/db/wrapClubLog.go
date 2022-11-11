package db

import (
	"context"
	"time"
	collClub "vvService/dbCollectionDefine/club"
)

// 写入俱乐部操作日志
func PutClubOperationLog(clubID int32, logType int32, uid int64, nick string, value interface{}) error {

	operationLog := collClub.DBClubOperationLog{
		ClubID:   clubID,
		Date:     time.Now(),
		OperID:   uid,
		OperName: nick,
		Category: logType,
		Data:     value,
	}

	collClubOperLog := mongoDBClient.Database(databaseName).Collection(collClub.CollClubOperationLog)
	ctx := context.Background()

	_, err := collClubOperLog.InsertOne(ctx, &operationLog)
	return err
}
