package db

import (
	"context"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func UpdatePlayerLastClubTime(clubID int32, uid int64) {
	coll := MongoClient.Database(CurDBName).Collection(collPlayer.CollPlayerInfo)
	ctx := context.Background()

	_, err := coll.UpdateOne(ctx,
		bson.M{"uid": uid, "club_info.clubID": clubID},
		bson.M{"$set": bson.M{"club_info.$.last_play_time": time.Now()}})
	if err != nil {
		glog.Warning("UpdatePlayerLastClubTime() err. uid:=", uid, ",clubID:=", clubID, ",err:=", err.Error())
	}
}

func GetClubPlayPercent(mzClubID, clubID int32, playID int64, v *collClub.DBClubPlayPercentage) error {
	coll := MongoClient.Database(CurDBName).Collection(collClub.CollClubPlayPercentage)
	rsp := coll.FindOne(nil, bson.D{{"mz_club_id", mzClubID}, {"play_id", playID}, {"club_id", clubID}})

	err := rsp.Decode(v)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}

	return nil
}
