package db

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	collClub "vvService/dbCollectionDefine/club"
)

//func Testa() {
//
//	log := collClub.DBClubBaoDiScoreLog{JiangLiScore: 100}
//	PushBaoDiItem(time.Now(), 20220822, 1, 2, &log)
//
//	jlLog := collClub.DBClubJiangLiScoreLog{JiangLiScore: 50}
//	PushJiangLiItem(time.Now(), 20220822, 1, 2, &jlLog)
//}

func PushBaoDiItem(curNow time.Time, date int, clubID int32, uid int64, bdLog *collClub.DBClubBaoDiScoreLog) error {
	if bdLog.JiangLiScore == 0 {
		return nil
	}

	coll_ := MongoClient.Database(CurDBName).Collection(collClub.CollSafeBox)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)

	log := &collClub.DBClubScoreLog{CreateTime: curNow, Category: collClub.LogBaoDi, Data: bdLog}

	_, err := coll_.UpdateOne(nil,
		bson.D{{"date", date}, {"uid", uid}, {"club_id", clubID}},
		bson.M{"$push": bson.M{"bd_item": log}, "$inc": bson.M{"bd_c": bdLog.JiangLiScore}}, &opt)

	return err

}

func PushJiangLiItem(curNow time.Time, date int, clubID int32, uid int64, jlLog []collClub.DBClubJiangLiScoreLog, jlCount int64) error {

	coll_ := MongoClient.Database(CurDBName).Collection(collClub.CollSafeBox)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)

	log := &collClub.DBClubScoreLog{CreateTime: curNow, Category: collClub.LogJiangLi, Data: jlLog}

	_, err := coll_.UpdateOne(nil,
		bson.D{{"date", date}, {"club_id", clubID}, {"uid", uid}},
		bson.M{"$push": bson.M{"jl_item": log}, "$inc": bson.M{"jl_c": jlCount}}, &opt)

	return err

}
