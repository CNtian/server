package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	"vvService/appDB/collIndex"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

var (
	MongoClient *mongo.Client
	CurDBName   string
)

func SetMongo(client *mongo.Client, database string) {
	MongoClient = client
	CurDBName = database
}

func LoadClub(clubID int32) (*collClub.DBClubData, error) {
	coll := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	var opt options.FindOneOptions
	opt.SetProjection(bson.M{
		"plays": 1, "superior_percentage": 1, "d_superior": 1,
		"creator_id": 1, "club_id": 1, "name": 1, "manageFee": 1, "level": 1})

	clubData := collClub.DBClubData{}
	return &clubData, coll.FindOne(ctx, bson.M{"club_id": clubID}, &opt).Decode(&clubData)
}

// 更新 奖励分
func UpdateClubJiangLiScore(nowTT time.Time, date_ int, clubID int32, param *collClub.DBClubJiangLiScoreLog) error {
	if param.GongXian == 0 || param.JiangLiScore == 0 {
		return nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, clubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	collPlayerInfo := MongoClient.Database(CurDBName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	collPlayerTotal := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerTotal, wcMajorityCollectionOpts)
	ctx := context.Background()

	playerInfo := collPlayer.PlayerInfo{}

	session, outSideErr := MongoClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error

		// 更新 个人 俱乐部分
		eleMatch := bson.M{"clubID": clubID}

		filter := bson.M{"uid": param.GotID, "club_info": bson.M{"$elemMatch": eleMatch}}
		update := bson.M{"$inc": bson.M{"club_info.$.score": param.JiangLiScore}}
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
		opt.SetReturnDocument(options.After)

		errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
		if errSession != nil {
			return nil, errSession
		}

		for _, v := range playerInfo.ClubData {
			if v.ClubID == clubID {
				param.CurClubScore = v.Score
				break
			}
		}

		filter = bson.M{"date": date_, "uid": param.GotID, "player_clubID": clubID}
		updateOpt := options.UpdateOptions{}
		updateOpt.SetUpsert(true)
		_, errSession = collPlayerTotal.UpdateOne(sctx, filter,
			bson.M{"$inc": bson.M{"jlS": param.JiangLiScore}}, &updateOpt)
		if errSession != nil {
			return nil, errSession
		}

		//readClubIDArr :=
		//readPlayerArr := []int64{param.GotID}
		//
		//if readClubIDArr[0] != param.GotClubID {
		//	readClubIDArr = append(readClubIDArr, param.GotClubID)
		//}
		//if readPlayerArr[0] != param.GotID {
		//	readPlayerArr = append(readPlayerArr, param.GotID)
		//}

		// 写入 日志
		//_, errSession = collClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
		//	CreateTime: nowTT,
		//	ClubID:     param.GotClubID,
		//	PlayerID:   param.GotID,
		//	Category:   collClub.LogJiangLi,
		//	Data:       param,
		//})

		return nil, nil
	})

	return outSideErr
}

func PutClubJiangLiLog(nowTT time.Time, date_ int, clubID int32, param *collClub.DBClubJiangLiScoreLog) error {
	if param.GongXian == 0 || param.JiangLiScore == 0 {
		return nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, clubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	// 写入 日志
	_, err := collClubScoreLog.InsertOne(nil, &collClub.DBClubScoreLog{
		CreateTime: nowTT,
		ClubID:     param.GotClubID,
		PlayerID:   param.GotID,
		Category:   collClub.LogJiangLi,
		Data:       param,
	})

	return err
}

func PutClubBaoDiLog(nowTT time.Time, date_ int, clubID int32, param *collClub.DBClubBaoDiScoreLog) error {
	if param.GongXian == 0 || param.JiangLiScore == 0 {
		return nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, clubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	// 写入 日志
	_, err := collClubScoreLog.InsertOne(nil, &collClub.DBClubScoreLog{
		CreateTime: nowTT,
		ClubID:     param.GotClubID,
		PlayerID:   param.GotID,
		Category:   collClub.LogBaoDi,
		Data:       param,
	})

	return err
}

// 更新 保底分
func UpdateClubBaoDiScore(nowTT time.Time, date_ int, clubID int32, param *collClub.DBClubBaoDiScoreLog) error {
	if param.GongXian == 0 || param.JiangLiScore == 0 {
		return nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, clubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	collPlayerInfo := MongoClient.Database(CurDBName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	collPlayerTotal := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerTotal, wcMajorityCollectionOpts)
	ctx := context.Background()

	playerInfo := collPlayer.PlayerInfo{}

	session, outSideErr := MongoClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error

		// 更新 个人 俱乐部分
		eleMatch := bson.M{"clubID": clubID}

		filter := bson.M{"uid": param.GotID, "club_info": bson.M{"$elemMatch": eleMatch}}
		update := bson.M{"$inc": bson.M{"club_info.$.score": param.JiangLiScore}}
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
		opt.SetReturnDocument(options.After)

		errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
		if errSession != nil {
			return nil, errSession
		}

		for _, v := range playerInfo.ClubData {
			if v.ClubID == clubID {
				param.CurClubScore = v.Score
				break
			}
		}

		filter = bson.M{"date": date_, "uid": param.GotID, "player_clubID": clubID}
		updateOpt := options.UpdateOptions{}
		updateOpt.SetUpsert(true)
		_, errSession = collPlayerTotal.UpdateOne(sctx, filter,
			bson.M{"$inc": bson.M{"bao_di": param.JiangLiScore}}, &updateOpt)
		if errSession != nil {
			return nil, errSession
		}

		// 写入 日志
		_, errSession = collClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: nowTT,
			ClubID:     param.GotClubID,
			PlayerID:   param.GotID,
			Category:   collClub.LogBaoDi,
			Data:       param,
		})

		return nil, nil
	})

	return outSideErr
}

// 记录玩家俱乐部 游戏分
func UpdateClubGameScore(nowTT time.Time, date_ int, mzClubID, clubID int32, param *collClub.DBClubGameScoreLog) error {

	if param.GameScore == 0 {
		return nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	collPlayerInfo := MongoClient.Database(CurDBName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	ctx := context.Background()

	playerInfo := collPlayer.PlayerInfo{}

	session, outSideErr := MongoClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error

		gameScore := param.GameScore
		// 提前离场的,已经扣除了
		if param.IsLeave != 0 {
			gameScore = 0
		}
		// 更新 个人 俱乐部分
		filter := bson.M{"uid": param.UID, "club_info": bson.M{"$elemMatch": bson.M{"clubID": clubID}}}
		update := bson.M{"$inc": bson.M{"club_info.$.score": gameScore}}
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
		opt.SetReturnDocument(options.After)

		errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
		if errSession != nil {
			return nil, errSession
		}

		for _, v := range playerInfo.ClubData {
			if v.ClubID == clubID {
				param.CurClubScore = v.Score
				break
			}
		}

		// 写入 日志
		_, errSession = collClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: nowTT,
			ClubID:     clubID,
			PlayerID:   param.UID,
			Category:   collClub.LogGame,
			Data:       param,
		})

		return nil, nil
	})

	return outSideErr
}

// 更新 消耗分
func UpdateClubXiaoHaoScore(nowTT time.Time, date_ int, mzClubID, clubID int32, param *collClub.DBClubXiaoHaoScoreLog) error {
	if param.XiaoHao == 0 {
		return nil
	}
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	collPlayerInfo := MongoClient.Database(CurDBName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	ctx := context.Background()

	playerInfo := collPlayer.PlayerInfo{}

	session, outSideErr := MongoClient.StartSession()
	if outSideErr != nil {
		return outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error

		// 更新 个人 俱乐部分
		filter := bson.M{"uid": param.UID, "club_info": bson.M{"$elemMatch": bson.M{"clubID": clubID}}}
		update := bson.M{"$inc": bson.M{"club_info.$.score": -param.XiaoHao}}
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
		opt.SetReturnDocument(options.After)

		errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
		if errSession != nil {
			return nil, errSession
		}

		for _, v := range playerInfo.ClubData {
			if v.ClubID == clubID {
				param.CurClubScore = v.Score
				break
			}
		}

		// 写入 日志
		_, errSession = collClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: nowTT,
			ClubID:     clubID,
			PlayerID:   param.UID,
			Category:   collClub.LogXiaoHaoValue,
			Data:       param,
		})
		return nil, nil
	})

	return outSideErr
}

// 保底分
func PutBaoDiScore(nowTT time.Time, date_ int, clubID int32, param *collClub.DBClubBaoDiScoreLog) error {
	if param.JiangLiScore == 0 {
		return nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, clubID, date_)
	collClubScoreLog := MongoClient.Database(CurDBName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

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
		var errSession error

		_, errSession = collClubInfo.UpdateOne(sctx, bson.M{"club_id": clubID}, bson.M{"$inc": bson.M{"score_count": param.JiangLiScore}})
		if errSession != nil {
			return nil, errSession
		}

		// 更新 个人 俱乐部分
		filter := bson.M{"uid": param.UID, "club_info": bson.M{"$elemMatch": bson.M{"clubID": clubID}}}
		update := bson.M{"$inc": bson.M{"club_info.$.score": param.JiangLiScore}}
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
		opt.SetReturnDocument(options.After)

		errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
		if errSession != nil {
			return nil, errSession
		}

		for _, v := range playerInfo.ClubData {
			if v.ClubID == clubID {
				param.CurClubScore = v.Score
				break
			}
		}

		// 写入 日志
		_, errSession = collClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: nowTT,
			ClubID:     clubID,
			PlayerID:   param.UID,
			Category:   collClub.LogBaoDi,
			Data:       param,
		})
		return nil, nil
	})

	return outSideErr
}

// 俱乐部总分
func UpdateClubCountScore(clubID int32, score int64) (int64, error) {
	if score == 0 {
		return 0, nil
	}
	collClubInfo := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo)

	tempOtp := options.FindOneAndUpdateOptions{}
	// 俱乐部总分
	//updateRes, err := collClubInfo.UpdateOne(ctx, bson.M{"club_id": clubID}, bson.M{"$inc": bson.M{"score_count": score}})
	//if err != nil {
	//	return 0, err
	//}
	//if updateRes.ModifiedCount < 1 {
	//	return fmt.Errorf("ModifiedCount < 1 ")
	//}
	tempOtp.SetProjection(bson.M{"score_count": 1})
	tempOtp.SetReturnDocument(options.Before)
	single := collClubInfo.FindOneAndUpdate(nil,
		bson.M{"club_id": clubID},
		bson.M{"$inc": bson.M{"score_count": score}}, &tempOtp)

	if single.Err() != nil {
		return 0, single.Err()
	}
	tempClub := collClub.DBClubData{}
	errSession := single.Decode(&tempClub)
	if errSession != nil {
		return 0, errSession
	}

	return tempClub.ClubScoreCount, nil
}

// 俱乐部统计更新
func UpdateClubTotal(param *dbCollectionDefine.DBClubTotal, players int64) error {
	collClubTotal := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubTotal)
	ctx := context.Background()

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)
	_, err := collClubTotal.UpdateOne(ctx,
		bson.M{"date": param.Date, "club_id": param.ClubID, "club_CID": param.ClubCreator},
		bson.M{"$inc": bson.M{"game_score": param.GameScore,
			"game_count": players,
			"haoKa":      param.HaoKa,
			"gongXian":   param.GongXian,
			"jiangLi":    param.JiangLi,
			"baoDi":      param.BaoDi,
			"xiao_hao":   param.XiaoHaoCount}}, &opt)
	return err
}

// 俱乐部玩家统计
func ReadClubPlayerTotal(date int, funcCallBack func(*dbCollectionDefine.DBClubPlayerTotal)) error {
	coll := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerTotal)

	ctx := context.Background()
	cur, err := coll.Find(ctx, bson.M{"date": date})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		temp := new(dbCollectionDefine.DBClubPlayerTotal)
		err = cur.Decode(temp)
		if err != nil {
			glog.Warning("ReadClubPlayerTotal().", err.Error())
			continue
		}
		funcCallBack(temp)
	}

	return err
}

// 俱乐部玩家统计
func GetClubPlayerTotal(date int, clubID int32, uid int64, readValue *dbCollectionDefine.DBClubPlayerTotal) error {
	coll := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerTotal)

	single := coll.FindOne(nil, bson.D{{"date", date}, {"player_clubID", clubID}, {"uid", uid}})

	return single.Decode(readValue)
}

func UpdateClubPlayerTotal(date int, writeArr []mongo.WriteModel) error {
	updateCount := len(writeArr)

	col := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerTotal)
	ctx := context.Background()

	result, err := col.BulkWrite(ctx, writeArr)
	if err != nil {
		return err
	}
	if result.ModifiedCount+result.InsertedCount+result.UpsertedCount != int64(updateCount) {
		glog.Warning("CollClubPlayerTotal() err. date:=", date,
			" ,result.ModifiedCount:=", result.ModifiedCount, " ,int64(len(updateMap):=", updateCount)
	}
	return err
}

func DeleteClubPlayerUnusableScore(uid int64, clubID int32) (int64, error) {

	col := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerUnusableScore)

	rsp := col.FindOneAndDelete(context.Background(), bson.M{"uid": uid, "clubID": clubID})

	temp := dbCollectionDefine.DBClubPlayerUnusableScore{}
	err := rsp.Decode(&temp)

	return temp.GameScore, err
}

func UpdateClubPlayerUnusableScore(uid, newAddScore int64, clubID int32) error {

	col := MongoClient.Database(CurDBName).Collection(dbCollectionDefine.CollClubPlayerUnusableScore)

	opt := options.UpdateOptions{}
	opt.SetUpsert(true)
	_, err := col.UpdateOne(context.Background(), bson.M{"uid": uid, "clubID": clubID}, bson.M{"$inc": bson.M{"game_score": newAddScore}}, &opt)

	return err
}

func UpdateClubUnusableScore(writeArr []mongo.WriteModel) error {
	updateCount := len(writeArr)

	col := MongoClient.Database(CurDBName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	result, err := col.BulkWrite(ctx, writeArr)
	if err != nil {
		return err
	}
	if result.ModifiedCount+result.InsertedCount+result.UpsertedCount != int64(updateCount) {
		glog.Warning("UpdateClubUnusableScore() err. result.ModifiedCount:=", result.ModifiedCount, " ,int64(len(updateMap):=", updateCount)
	}
	return err
}

//func PutClubScoreLog(data []mongo.WriteModel) {
//	if len(data) < 1 {
//		return
//	}
//
//	y, m, d := time.Now().Date()
//	collName := fmt.Sprintf("%s_%d_%02d_%02d", dbCollectionDefine.CollClubScoreLogPre, y, m, d)
//	coll := MongoClient.Database(CurDBName).Collection(collName)
//
//	re, err := coll.BulkWrite(nil, data)
//	if err != nil {
//		glog.Warning("PutClubScoreLog", err.Error())
//		return
//	}
//	if re.InsertedCount != int64(len(data)) {
//		glog.Warning("PutClubScoreLog ready write len:=", len(data), ",reality:=", re.InsertedCount)
//	}
//}
