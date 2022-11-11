package worker

import (
	"context"
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	"vvService/appDB/db"
	"vvService/appDB/protoDefine"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
)

func onRoundOver(msg *mateProto.MessageMaTe) {

	var roundRecord = protoDefine.SS_RoundRecord{}
	err := json.Unmarshal(msg.Data, &roundRecord)
	if err != nil {
		glog.Warning("onRoundOver() err:=", err.Error(), ",data:=", string(msg.Data))
		return
	}

	newRoundDetail := &dbCollectionDefine.RoundDetail{
		Begin:    roundRecord.Begin,
		End:      roundRecord.End,
		CurRound: roundRecord.CurRound,
		GameStep: roundRecord.GameStep,
		Players:  roundRecord.Players}

	ctx := context.Background()
	coll := db.MongoClient.Database(db.CurDBName).Collection(dbCollectionDefine.CollRoundRecord)

	insertFunc := func() {
		newDBRoundRecord := dbCollectionDefine.DBRoundRecord{
			CrateTime:   time.Now(),
			RoundID:     roundRecord.RoundID,
			ClubID:      roundRecord.ClubID,
			TableID:     roundRecord.TableID,
			RoundDetail: []*dbCollectionDefine.RoundDetail{newRoundDetail}}

		_, err = coll.InsertOne(ctx, &newDBRoundRecord)
		if err != nil {
			glog.Warning("onRoundOver() err:=", err.Error(), ",data:=", string(msg.Data))
		}
	}

	if roundRecord.CurRound < 2 {
		insertFunc()
		return
	}

	filter := bson.M{"round_id": roundRecord.RoundID, "club_id": roundRecord.ClubID, "table_id": roundRecord.TableID}
	update := bson.M{"$push": bson.M{"detail": newRoundDetail}}

	var res *mongo.UpdateResult
	res, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		glog.Warning("onRoundOver() err:=", err.Error(), ",data:=", string(msg.Data))
		return
	}
	if res.MatchedCount < 1 {
		insertFunc()
		return
	}

	if res.ModifiedCount < 1 {
		filterText, _ := json.Marshal(filter)
		glog.Warning("onRoundOver() data:=", string(msg.Data), ",filter:=", string(filterText))
		return
	}
}
