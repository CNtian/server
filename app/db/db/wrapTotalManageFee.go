package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"strconv"
	"time"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func GetClubManageFeeClubID(date int) (map[int32]*dbCollectionDefine.DBClubTotal, error) {
	collSelect := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubTotal)
	ctxSelect := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"haoKa": 1})
	cur, err := collSelect.Find(ctxSelect, bson.M{"date": date})
	if err != nil {
		glog.Warning("GetClubManageFeeClubID() err. date:=", date, " ,err:=", err.Error())
		return nil, err
	}
	defer cur.Close(ctxSelect)

	// key:俱乐部ID
	clubIDMap := make(map[int32]*dbCollectionDefine.DBClubTotal)
	for cur.Next(ctxSelect) {
		temp := dbCollectionDefine.DBClubTotal{}
		err = cur.Decode(&temp)
		if err != nil {
			glog.Warning("GetClubManageFeeClubID() err. date:=", date, " ,err:=", err.Error())
			continue
		}
		temp.TempManageFeeCount = temp.ManageFee
		clubIDMap[temp.ClubID] = &temp
	}

	return clubIDMap, nil
}

func GetClubHaoKaCount(date int, clubID int32) (int64, error) {
	collSelect := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubTotal)
	ctxSelect := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"haoKa": 1})
	cur, err := collSelect.Find(ctxSelect, bson.M{"date": date, "club_id": clubID})
	if err != nil {
		glog.Warning("GetClubHaoKaCount() err. date:=", date, " ,err:=", err.Error())
		return 0, err
	}
	defer cur.Close(ctxSelect)

	// key:俱乐部ID
	for cur.Next(ctxSelect) {
		temp := dbCollectionDefine.DBClubTotal{}
		err = cur.Decode(&temp)
		if err != nil {
			glog.Warning("GetClubHaoKaCount() err. date:=", date, " ,err:=", err.Error())
			continue
		}
		return temp.HaoKa, nil
	}

	return 0, nil
}

func UpdateManageFeeLog(log *collClub.DBClubManageFeeLog) error {
	if log.ManageFeeScore == 0 {
		return nil
	}
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	nowTT := time.Now()
	year, month, day := nowTT.Date()
	date_, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	collName := fmt.Sprintf("%s_%d_%d",collClub.CollClubScoreLog,log.GotClubID,date_)
	gotClubcollClubScoreLog := MongoClient.Database(CurDBName).Collection(collName, wcMajorityCollectionOpts)

	collName = fmt.Sprintf("%s_%d_%d",collClub.CollClubScoreLog,log.PayClubID,date_)
	payClubcollClubScoreLog := MongoClient.Database(CurDBName).Collection(collName, wcMajorityCollectionOpts)

	collPlayerInfo := MongoClient.Database(CurDBName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	collClubInfo := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)

	ctx := context.Background()

	playerInfo := collPlayer.PlayerInfo{}

	session, outSideErr := MongoClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var (
			errSession   error
			updateResult *mongo.UpdateResult
		)

		// 更新 个人 俱乐部分
		{
			filter := bson.M{"uid": log.PayUID, "club_info": bson.M{"$elemMatch": bson.M{"clubID": log.PayClubID}}}
			update := bson.M{"$inc": bson.M{"club_info.$.score": -log.ManageFeeScore}}
			opt := options.FindOneAndUpdateOptions{}
			opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
			opt.SetReturnDocument(options.Before)

			errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
			if errSession != nil {
				return nil, errSession
			}

			for _, v := range playerInfo.ClubData {
				if v.ClubID == log.PayClubID {
					log.PayBeforeClubScore = v.Score
					log.PayCurClubScore = v.Score - log.ManageFeeScore
				}
			}
			// 修改 俱乐部 总分
			updateResult, errSession = collClubInfo.UpdateOne(sctx,
				bson.M{"club_id": log.PayClubID},
				bson.M{"$inc": bson.M{"score_count": -log.ManageFeeScore}})
			if errSession != nil {
				return nil, errSession
			}
			if updateResult.ModifiedCount != 1 {
				return nil, fmt.Errorf("updateClubScoreFunc() modify count:=%d", updateResult.ModifiedCount)
			}
		}

		{
			filter := bson.M{"uid": log.GotUID, "club_info": bson.M{"$elemMatch": bson.M{"clubID": log.GotClubID}}}
			update := bson.M{"$inc": bson.M{"club_info.$.score": log.ManageFeeScore}}
			opt := options.FindOneAndUpdateOptions{}
			opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
			opt.SetReturnDocument(options.Before)

			errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
			if errSession != nil {
				return nil, errSession
			}

			for _, v := range playerInfo.ClubData {
				if v.ClubID == log.GotClubID {
					log.GotBeforeClubScore = v.Score
					log.GotCurClubScore = v.Score + log.ManageFeeScore
				}
			}
		}

		// 写入 日志
		_, errSession = payClubcollClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: nowTT,
			ClubID:log.PayClubID,
			PlayerID:log.PayUID,
			Category: collClub.LogMangeFei,
			Data:     log,
		})

		_, errSession = gotClubcollClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: nowTT,
			ClubID:log.GotClubID,
			PlayerID:log.GotUID,
			Category: collClub.LogMangeFei,
			Data:     log,
		})

		return nil, errSession
	})

	return outSideErr
}

func UpdateMangeFee(date int, updateMap map[int32]*dbCollectionDefine.DBClubTotal) error {

	updateCount := 0
	update := make([]mongo.WriteModel, 0, len(updateMap))

	for k, v := range updateMap {
		if v.TempManageFeeCount < 1 {
			//glog.Warning("UpdateMangeFee() err. date:=", date, " ,clubID:=", k)
			continue
		}
		updateCount += 1
		update = append(update,
			mongo.NewUpdateOneModel().SetFilter(bson.M{"date": date, "club_id": k}).SetUpdate(bson.M{"$set": bson.M{"manageFee": v.TempManageFeeCount}}))
	}

	if len(update) < 1 {
		return nil
	}

	collClubManageFee := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubTotal)
	ctx := context.Background()

	result, err := collClubManageFee.BulkWrite(ctx, update)
	if err != nil {
		return err
	}
	if result.ModifiedCount != int64(updateCount) {
		glog.Warning("UpdateMangeFee() err. date:=", date,
			" ,result.ModifiedCount:=", result.ModifiedCount, " ,int64(len(updateMap):=", updateCount)
	}
	return nil
}

func UpdateHaoKa(date int, clubIDArr []int32, count int64, resetClubIDArr []int32) {
	update := make([]mongo.WriteModel, 0, len(clubIDArr)+len(resetClubIDArr)+1)

	for _, v := range clubIDArr {
		update = append(update,
			mongo.NewUpdateOneModel().SetFilter(bson.M{"date": date, "club_id": v}).SetUpdate(bson.M{"$inc": bson.M{"haoKa": -count}}))
	}

	for _, v := range resetClubIDArr {
		update = append(update,
			mongo.NewUpdateOneModel().SetFilter(bson.M{"date": date, "club_id": v}).SetUpdate(bson.M{"$set": bson.M{"haoKa": 0}}))
	}

	collClubManageFee := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubTotal)
	ctx := context.Background()

	_, err := collClubManageFee.BulkWrite(ctx, update)
	if err != nil {
		glog.Warning("UpdateHaoKa() err. err:=", err.Error())
		return
	}
}
