package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
	"time"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

const (
	_15Days = 60 * 60 * 24 * 15 // 最近 15 天
	_1Days  = 60 * 60 * 24 * 1  // 最近 1 天
	_3Days  = 60 * 60 * 24 * 3  // 最近 3 天
	_7Days  = 60 * 60 * 24 * 7  // 最近 7 天
	_10Days = 60 * 60 * 24 * 10 // 最近 10 天
)

func totalClubOperationLog() error {
	//coll := MongoClient.Database(CurDBName).Collection(collClub.CollClubOperationLog)
	//ctx := context.Background()
	//
	//cur, err := coll.Aggregate(ctx, bson.A{
	//	bson.M{"$group": bson.M{"_id": bson.M{"clubID": "$club_id"}, "count": bson.M{"$sum": 1}}},
	//	bson.M{"match": bson.M{"count": bson.M{"$gt": _30ItemCount}}}})
	//if err != nil {
	//	return err
	//}
	//defer cur.Close(ctx)
	//
	//type AggregateResult struct {
	//	ID struct {
	//		ClubID int32 `bson:"clubID"`
	//	} `bson:"_id"`
	//	Count int `bson:"count"`
	//}
	//var res AggregateResult
	//
	//tt := time.Unix(time.Now().Unix()-_7Days, 0)
	//
	//for cur.Next(ctx) {
	//	err = cur.Decode(&res)
	//	if err != nil {
	//		glog.Warning("totalClubOperationLog(). err:=", err.Error())
	//	}
	//
	//	err = deleteClubOperationLog(res.ID.ClubID, tt)
	//	if err != nil {
	//		glog.Warning("totalClubOperationLog(). err:=", err.Error())
	//	}
	//}
	return nil
}

func DeleteExpiredData() {

	glog.Warning("DeleteExpiredData starting....")
	{
		t7 := time.Unix(time.Now().Unix()-_7Days, 0)
		y, m, d := t7.Date()
		date7, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))

		DeleteLog(collClub.CollClubMail, bson.M{"create_time": bson.M{"$lt": t7}})

		DeleteLog(collClub.CollClubOperationLog, bson.M{"date": bson.M{"$lt": t7}})

		DeleteLog(dbCollectionDefine.CollClubPlayerTotal, bson.M{"date": bson.M{"$lt": date7}})

		DeleteLog(collPlayer.CollPlayerPropertyLog, bson.M{"operator_time": bson.M{"$lt": t7}})

		DeleteLog(dbCollectionDefine.CollClubTotal, bson.M{"date": bson.M{"$lt": date7}})

		DeleteLog(dbCollectionDefine.CollDailyMengZHuPlayer, bson.M{"date": bson.M{"$lt": date7}})

		DeleteLog(collPlayer.CollPlayerEmail, bson.M{"create_time": bson.M{"$lt": t7}})

		DeleteLog(collClub.CollRoomCardDealLog, bson.M{"create_time": bson.M{"$lt": t7}})

		DeleteLog(dbCollectionDefine.CollRoundRecord, bson.M{"create_time": bson.M{"$lt": t7}})

		DeleteLog(collClub.CollSafeBox, bson.M{"date": bson.M{"$lt": date7}})
	}

	{
		t1 := time.Unix(time.Now().Unix()-_1Days, 0)
		y, m, d := t1.Date()
		date1, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", y, m, d))
		RedisDeleteTongZhuoKey(date1)

		coll := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo)

		deleteResult, err := coll.UpdateMany(nil, bson.M{},
			bson.M{"$pull": bson.M{"plays": bson.M{"del": true, "delete_time": bson.M{"$lt": t1}}}})
		if err != nil {
			glog.Warning("CollClubInfo UpdateMany() err. err:=", err.Error())
		} else {
			glog.Warning("DeleteMany clubPlays ", deleteResult.ModifiedCount)
		}

		coll = MongoClient.Database(CurDBName).Collection(collClub.CollClubPlayPercentage)
		del, err := coll.DeleteMany(nil, bson.M{"is_del": true, "del_time": bson.M{"$lt": t1}})
		if err != nil {
			glog.Warning("CollClubPlayPercentage DeleteMany() err. err:=", err.Error())
		} else {
			glog.Warning("DeleteMany ClubPlayPercentage ", del.DeletedCount)
		}
	}

	{
		tt := time.Now().Unix()
		for i := int64(7); i < 10; i++ {
			year, month, day := time.Unix(tt-60*60*24*i, 0).Date()
			date__, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
			_name := fmt.Sprintf("%s_%d", dbCollectionDefine.CollGameRecord, date__)
			_coll := MongoClient.Database(CurDBName).Collection(_name)
			err := _coll.Drop(nil)
			if err != nil {
				glog.Warning("DeleteGameRound.", _name, ",err:=", err.Error())
			}

			_name = fmt.Sprintf("index_%s", _name)
			_coll.Indexes().DropOne(nil, _name)

			//_name = fmt.Sprintf("%s_%d_%02d_%02d", dbCollectionDefine.CollClubScoreLogPre, year, month, day)
			//_coll = MongoClient.Database(CurDBName).Collection(_name)
			//err = _coll.Drop(nil)
			//if err != nil {
			//	glog.Warning("CollClubScoreLog.", _name, ",err:=", err.Error())
			//}
		}
	}

	DeleteClubScoreLog()

}

func DeleteClubScoreLog() {
	mengZhuClub := []collClub.DBClubData{}
	{
		coll_ := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo)
		ctx := context.Background()
		opt := options.FindOptions{}
		opt.SetProjection(bson.M{"club_id": 1, "creator_id": 1})
		cur, err := coll_.Find(ctx, bson.M{"d_superior.club_id": 0})
		if err != nil {
			glog.Warning(err.Error())
			return
		}
		defer cur.Close(ctx)

		err = cur.All(ctx, &mengZhuClub)
		if err != nil {
			glog.Warning(err.Error())
			return
		}
	}

	droupCount := 0
	for i := int64(0); i < 20; i++ {
		t3 := time.Unix(time.Now().Unix()-(_7Days+(i*60*60*24)), 0)
		year, month, day := t3.Date()
		date_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
		isFind := false
		for i, _ := range mengZhuClub {
			collName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mengZhuClub[i].ClubID, date_)
			coll_ := MongoClient.Database(CurDBName).Collection(collName)
			coll_.Indexes().DropAll(nil)
			err := coll_.Drop(nil)
			if err != nil {
				glog.Error(err.Error())
			}
			isFind = true

			collName = fmt.Sprintf("%s_%d_%d", dbCollectionDefine.CollTongZhuo, mengZhuClub[i].ClubID, date_)
			coll_ = MongoClient.Database(CurDBName).Collection(collName)
			coll_.Indexes().DropAll(nil)
			err = coll_.Drop(nil)
			if err != nil {
				glog.Warning("tong zhuo.", collName, ",err:=", err.Error())
			}
		}
		if !isFind {
			break
		}
	}

	glog.Warning("DeleteMany CollClubScoreLog ", droupCount)
}

func DeleteLog(collName string, filter bson.M) {
	coll := MongoClient.Database(CurDBName).Collection(collName)

	delCount := int64(0)
	for i := 0; i < 100; i++ {
		ctx := context.Background()
		opt := options.FindOptions{}
		opt.SetLimit(5000)
		opt.SetProjection(bson.M{"_id": 1})
		cur, err := coll.Find(ctx, filter, &opt)
		if err != nil {
			glog.Warning("DeleteLog err.", collName, "    err:=", err.Error())
			return
		}

		_idArr := make([]primitive.ObjectID, 0, 5000)
		for cur.Next(ctx) {
			id := cur.Current.Lookup("_id").ObjectID()
			_idArr = append(_idArr, id)
		}
		cur.Close(ctx)

		if len(_idArr) < 1 {
			break
		}

		r, err := coll.DeleteMany(nil, bson.M{"_id": bson.M{"$in": _idArr}})
		if err != nil {
			glog.Warning("DeleteLog err.", collName, "    err:=", err.Error())
		} else {
			delCount += r.DeletedCount
		}
	}
	glog.Warning("DeleteMany ", collName, "  ", delCount)
}
