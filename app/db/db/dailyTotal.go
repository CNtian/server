package db

import (
	"context"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

// key:游戏ID  value:玩家人数
func PutNewDay(v *dbCollectionDefine.DBDailyMengZHuPlayer) error {
	coll_ := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollDailyMengZHuPlayer)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)
	_, err := coll_.UpdateOne(nil,
		bson.M{"date": v.Date, "mzClubID": v.ClubID},
		bson.M{"$set": bson.M{"players": v.Players,
			"daily_players": v.DailyPlayers,
			"g_r_count":     v.GameRoundCount,
			"consumables":   v.Consumables,
			"g_consumables": v.GameToConsumablesArr}}, &opt)
	return err
}

func GetMengZhuAllPlayers(clubID int32) (int, error) {

	collClubInfo := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo)

	tempOtp := options.FindOneOptions{}
	tempOtp.SetProjection(bson.M{"subordinates": 1})

	s := collClubInfo.FindOne(nil, bson.M{"club_id": clubID}, &tempOtp)
	if s.Err() != nil {
		return 0, s.Err()
	}
	tempClub := collClub.DBClubData{}
	err := s.Decode(&tempClub)
	if err != nil {
		return 0, err
	}

	tempClub.Subordinates = append(tempClub.Subordinates, clubID)
	ctx := context.Background()
	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"count": bson.M{"$size": "$members"}, "_id": 0})
	cur, err := collClubInfo.Find(ctx, bson.M{"club_id": bson.M{"$in": tempClub.Subordinates}}, &opt)
	if err != nil {
		return 0, err
	}
	defer cur.Close(ctx)

	allPlayers := int32(0)
	for cur.Next(ctx) {
		v, err := cur.Current.Values()
		if err != nil {
			glog.Warning("Decode Hello .", err.Error())
			continue
		}
		if len(v) < 1 {
			glog.Warning("Decode Hello .", len(v))
			continue
		}
		allPlayers += v[0].AsInt32()
	}

	return int(allPlayers), nil
}

func RecoverMengZhuDaily(date int, f func(*dbCollectionDefine.DBDailyMengZHuPlayer)) {
	coll_ := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollDailyMengZHuPlayer)

	ctx := context.Background()
	cur, err := coll_.Find(ctx, bson.M{"date": date})
	if err != nil {
		glog.Warning("RecoverMengZhuDaily() err := ", err.Error())
		return
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		t := dbCollectionDefine.DBDailyMengZHuPlayer{}
		err = cur.Decode(&t)
		if err != nil {
			glog.Warning("RecoverMengZhuDaily  Decode() err := ", err.Error())
			continue
		}
		f(&t)
	}
}
