package db

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"strconv"
	"time"
	"vvService/commonPackge/errorCodeDef"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func UpdatePlayerFrozen(clubID int32, playerID, operID int64, status, isMengZhu bool) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "members.uid": playerID}
	update := bson.M{"$set": bson.M{
		"members.$.frozen.operation_time": time.Now(),
		"members.$.frozen.status":         status,
		"members.$.frozen.operID":         operID,
		"members.$.frozen.is_MZ":          isMengZhu,
	}}
	updateResult, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", playerID, clubID)
	}
	return 0, nil
}

func UpdatePlayerStop3(clubID int32, playerID int64, value bool, players int) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "members.uid": playerID}
	update := bson.M{}
	if players == 3 {
		update = bson.M{"$set": bson.M{"members.$.isNo3": value}}
	} else if players == 4 {
		update = bson.M{"$set": bson.M{"members.$.isNo4": value}}
	} else {
		return 0, errors.New("not match players")
	}

	updateResult, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", playerID, clubID)
	}
	return 0, nil
}

func UpdateMemberRemark(clubID int32, playerID int64, v string) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "members.uid": playerID}
	update := bson.M{"$set": bson.M{"members.$.remark": v}}

	updateResult, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", playerID, clubID)
	}
	return 0, nil
}

func UpdateDirectSubordinateRemark(clubID, dSubClubID int32, v string) (int32, error) {
	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "d_subordinate.club_id": dSubClubID}
	update := bson.M{"$set": bson.M{"d_subordinate.$.remark": v}}

	updateResult, err := coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match dSubClubID:=%d,clubID:=%d", dSubClubID, clubID)
	}
	return 0, nil
}

func UpdatePlayerAuthority(clubID int32, playerID int64, isAdmin bool) (int32, error) {

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	filter := bson.M{"club_id": clubID, "members.uid": playerID}
	updateResult, err := coll.UpdateOne(ctx, filter,
		bson.M{"$set": bson.M{"members.$.is_admin": isAdmin}})
	if err != nil {
		return 0, err
	}
	if updateResult.MatchedCount < 1 && updateResult.ModifiedCount < 1 {
		return errorCodeDef.Err_Not_Find_Item, fmt.Errorf("not match uid:=%d,clubID:=%d", playerID, clubID)
	}
	return 0, nil
}

type OR int32

const (
	OR__QuanZhu_To_ChengYun      = 1 // 圈主(管理员) -> 本圈 成员
	OR__MengZhu_To_ChengYun      = 2 // 盟主 -> 本圈成员
	OR__MengZhu_To_XiaJiChengYun = 3 // 盟主 -> 下级 成员
	OR__MengZhu_To_MengZhu       = 4 // 盟主 -> 盟主自己
	OR__ShangJi_To_XiaJi         = 5 // 上级 圈主(管理员) -> 下级 成员
)

type UpdateMemberScoreParam struct {
	// 被减成员
	A_UID          int64
	A_ClubName     string
	A_ClubID       int32
	A_Nick         string
	Retrun_A_Score int64
	Before_A_Score int64

	// 减数
	B_ClubID       int32
	B_ClubName     string
	B_UID          int64
	B_Nick         string
	Retrun_B_Score int64
	Before_B_Score int64

	// 操作分
	Value int64

	OperationRelation OR // 操作关系 0:圈主->成员 1:盟主->圈主  2:盟主->下级圈子成员 3：盟主->盟主
}

func UpdateMemberScore(mzClubID int32, param *UpdateMemberScoreParam) (int32, error) {

	judgeLog := collClub.DBClubJudgeLog{
		A_ClubID:   param.A_ClubID,
		A_ClubName: param.A_ClubName,
		A_UID:      param.A_UID,
		A_Nick:     param.A_Nick,
		B_ClubID:   param.B_ClubID,
		B_ClubName: param.B_ClubName,
		B_UID:      param.B_UID,
		B_Nick:     param.B_Nick,
		Value:      param.Value,
	}

	nowTT := time.Now()
	year, month, day := nowTT.Date()
	date__, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	AClubLog := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date__)
	AcollClubScoreLog := mongoDBClient.Database(databaseName).Collection(AClubLog, wcMajorityCollectionOpts)

	BClubLog := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date__)
	BcollClubScoreLog := mongoDBClient.Database(databaseName).Collection(BClubLog, wcMajorityCollectionOpts)

	//collIndex.FindAndCreate_club_score(AcollClubScoreLog,AClubLog)
	//collIndex.FindAndCreate_club_score(BcollClubScoreLog,BClubLog)

	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayerTotal := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal, wcMajorityCollectionOpts)
	ctx := context.Background()

	session, outSideErr := mongoDBClient.StartSession()
	if outSideErr != nil {
		return 0, outSideErr
	}
	defer session.EndSession(ctx)

	//insertLog := make([]mongo.WriteModel, 0, 7)
	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error
		// 修改 俱乐部 总分
		updateClubScoreFunc := func(clubID int32, value int64, insertLogID primitive.ObjectID) error {
			var (
				tempOtp options.FindOneAndUpdateOptions
			)
			tempOtp.SetProjection(bson.M{"score_count": 1})
			tempOtp.SetReturnDocument(options.Before)
			single := collClubInfo.FindOneAndUpdate(sctx,
				bson.M{"club_id": clubID},
				bson.M{"$inc": bson.M{"score_count": value}}, &tempOtp)

			if single.Err() != nil {
				return single.Err()
			}
			tempClub := collClub.DBClubData{}
			errSession = single.Decode(&tempClub)
			if errSession != nil {
				return errSession
			}

			//lg := mongo.NewInsertOneModel().SetDocument(dbCollectionDefine.DBClubScoreLog{ClubID: clubID,
			//	PreValue:     tempClub.ClubScoreCount,
			//	ChangedValue: param.Value,
			//	ID:           insertLogID,
			//	Category:     int32(collClub.LogCaiPan)})
			//insertLog = append(insertLog, lg)
			return nil
		}

		// 更新 个人 俱乐部分
		updatePlayerClubScoreFunc := func(clubID int32, uid, value int64) (int64, int64, error) {
			eleMatch := bson.M{}
			if value < 0 {
				eleMatch = bson.M{"clubID": clubID, "score": bson.M{"$gte": -value}}
			} else {
				eleMatch = bson.M{"clubID": clubID}
			}

			filter := bson.M{"uid": uid, "club_info": bson.M{"$elemMatch": eleMatch}}
			update := bson.M{"$inc": bson.M{"club_info.$.score": value}}
			opt := options.FindOneAndUpdateOptions{}
			opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
			opt.SetReturnDocument(options.Before)

			var (
				playerInfo collPlayer.PlayerInfo
			)
			errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
			if errSession != nil {
				return 0, 0, errSession
			}
			for _, v := range playerInfo.ClubData {
				if v.ClubID == clubID {
					return v.Score, v.Score + value, nil
				}
			}
			return 0, 0, fmt.Errorf("updatePlayerClubScoreFunc() not match clubID")
		}

		// 写入 日志
		writeClubLogFunc := func(c *mongo.Collection, cID int32, pID int64) (insertID primitive.ObjectID) {
			var res *mongo.InsertOneResult
			res, errSession = c.InsertOne(sctx, &collClub.DBClubScoreLog{
				CreateTime: nowTT,
				ClubID:     cID,
				PlayerID:   pID,
				Category:   collClub.LogCaiPan,
				Data:       &judgeLog,
			})
			if errSession != nil {
				return primitive.NilObjectID
			}

			return res.InsertedID.(primitive.ObjectID)
		}
		// 更新统计
		updatePlayerTotalFunc := func(clubID int32, uid int64, value int64) error {
			opt := options.UpdateOptions{}
			opt.SetUpsert(true)
			_, err := collPlayerTotal.UpdateOne(sctx, bson.M{"date": date__, "uid": uid, "player_clubID": clubID},
				bson.M{"$inc": bson.M{"zsS": value}}, &opt)
			return err
		}

		switch param.OperationRelation {
		case OR__MengZhu_To_MengZhu:
			param.Before_A_Score, param.Retrun_A_Score, errSession = updatePlayerClubScoreFunc(param.A_ClubID, param.A_UID, param.Value)
			if errSession != nil {
				break
			}

			errSession = updatePlayerTotalFunc(param.A_ClubID, param.A_UID, param.Value)
			if errSession != nil {
				break
			}

			judgeLog.CurAScore = param.Retrun_A_Score
			//insertID := writeClubLogFunc([]*collClub.PayReceiptInfo{{ClubID: param.A_ClubID, PlayerID: param.A_UID}})
			insertID := writeClubLogFunc(AcollClubScoreLog, param.A_ClubID, param.A_UID)
			if errSession != nil {
				break
			}

			errSession = updateClubScoreFunc(param.A_ClubID, param.Value, insertID)
			if errSession != nil {
				break
			}

		case OR__MengZhu_To_ChengYun:
			fallthrough
		case OR__QuanZhu_To_ChengYun:
			param.Before_A_Score, param.Retrun_A_Score, errSession = updatePlayerClubScoreFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			param.Before_B_Score, param.Retrun_B_Score, errSession = updatePlayerClubScoreFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}

			judgeLog.CurAScore = param.Retrun_A_Score
			judgeLog.CurBScore = param.Retrun_B_Score

			writeClubLogFunc(AcollClubScoreLog, param.A_ClubID, param.A_UID)
			writeClubLogFunc(BcollClubScoreLog, param.B_ClubID, param.B_UID)

			//writeClubLogFunc([]*collClub.PayReceiptInfo{
			//	{ClubID: param.A_ClubID, PlayerID: param.A_UID},
			//	{ClubID: param.A_ClubID, PlayerID: param.B_UID}})

		case OR__MengZhu_To_XiaJiChengYun:
			fallthrough
		case OR__ShangJi_To_XiaJi:
			param.Before_A_Score, param.Retrun_A_Score, errSession = updatePlayerClubScoreFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			param.Before_B_Score, param.Retrun_B_Score, errSession = updatePlayerClubScoreFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}

			judgeLog.CurAScore = param.Retrun_A_Score
			judgeLog.CurBScore = param.Retrun_B_Score

			//insertID := writeClubLogFunc([]*collClub.PayReceiptInfo{
			//	{ClubID: param.A_ClubID, PlayerID: param.A_UID},
			//	{ClubID: param.B_ClubID, PlayerID: param.B_UID}})
			insertID := writeClubLogFunc(AcollClubScoreLog, param.A_ClubID, param.A_UID)
			writeClubLogFunc(BcollClubScoreLog, param.B_ClubID, param.B_UID)

			if errSession != nil {
				break
			}

			var (
				tempClubID   = param.B_ClubID
				tempClubData *collClub.DBClubData
			)

			for i := 0; i < 100; i++ {
				tempClubData, errSession = LoadClub(tempClubID)
				if errSession != nil {
					break
				}

				// 更新 俱乐部 总分
				errSession = updateClubScoreFunc(tempClubData.ClubID, param.Value, insertID)
				if errSession != nil {
					break
				}

				tempClubID = tempClubData.DirectSupervisor.ClubID
				if tempClubID == param.A_ClubID {
					break
				}
			}
		default:
			errSession = fmt.Errorf("not find OperationRelation.OperationRelation:=%d", param.OperationRelation)
		}
		return nil, errSession
	})

	//if outSideErr == nil {
	//	PutClubScoreLog(insertLog)
	//}

	return 0, outSideErr
}

//func PutClubScoreLog(data []mongo.WriteModel) {
//	if len(data) < 1 {
//		return
//	}
//
//	y, m, d := time.Now().Date()
//	collName := fmt.Sprintf("%s_%d_%02d_%02d", dbCollectionDefine.CollClubScoreLogPre, y, m, d)
//	coll := mongoDBClient.Database(databaseName).Collection(collName)
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

func ActivityAward(param *UpdateMemberScoreParam, category int32, attach interface{}) (int32, error) {

	awardLog := collClub.DBClubActivityLog{
		A_ClubID:   param.A_ClubID,
		A_ClubName: param.A_ClubName,
		A_UID:      param.A_UID,
		A_Nick:     param.A_Nick,
		B_ClubID:   param.B_ClubID,
		B_ClubName: param.B_ClubName,
		B_UID:      param.B_UID,
		B_Nick:     param.B_Nick,
		Value:      param.Value,
		Additional: attach,
	}

	nowTT := time.Now()
	year, month, day := nowTT.Date()
	date__, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	AClubLog := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, param.A_ClubID, date__)
	AcollClubScoreLog := mongoDBClient.Database(databaseName).Collection(AClubLog, wcMajorityCollectionOpts)

	BClubLog := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, param.B_ClubID, date__)
	BcollClubScoreLog := mongoDBClient.Database(databaseName).Collection(BClubLog, wcMajorityCollectionOpts)

	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	collPlayerTotal := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal, wcMajorityCollectionOpts)
	collActivityList := mongoDBClient.Database(databaseName).Collection(collClub.CollActivityAwardList, wcMajorityCollectionOpts)
	ctx := context.Background()

	session, outSideErr := mongoDBClient.StartSession()
	if outSideErr != nil {
		return 0, outSideErr
	}
	defer session.EndSession(ctx)

	//insertLog := make([]mongo.WriteModel, 0, 7)
	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error
		// 修改 俱乐部 总分
		updateClubScoreFunc := func(clubID int32, value int64, insertLogID primitive.ObjectID) error {
			var (
				tempOtp options.FindOneAndUpdateOptions
			)
			tempOtp.SetProjection(bson.M{"score_count": 1})
			tempOtp.SetReturnDocument(options.Before)
			single := collClubInfo.FindOneAndUpdate(sctx,
				bson.M{"club_id": clubID},
				bson.M{"$inc": bson.M{"score_count": value}}, &tempOtp)

			if single.Err() != nil {
				return single.Err()
			}
			tempClub := collClub.DBClubData{}
			errSession = single.Decode(&tempClub)
			if errSession != nil {
				return errSession
			}

			//lg := mongo.NewInsertOneModel().SetDocument(dbCollectionDefine.DBClubScoreLog{ClubID: clubID,
			//	PreValue:     tempClub.ClubScoreCount,
			//	ChangedValue: param.Value,
			//	ID:           insertLogID,
			//	Category:     int32(collClub.LogActivityAward)})
			//insertLog = append(insertLog, lg)
			return nil
		}

		// 更新 个人 俱乐部分
		updatePlayerClubScoreFunc := func(clubID int32, uid, value int64) (int64, int64, error) {
			eleMatch := bson.M{}
			if value < 0 {
				eleMatch = bson.M{"clubID": clubID, "score": bson.M{"$gte": -value}}
			} else {
				eleMatch = bson.M{"clubID": clubID}
			}

			filter := bson.M{"uid": uid, "club_info": bson.M{"$elemMatch": eleMatch}}
			update := bson.M{"$inc": bson.M{"club_info.$.score": value}}
			opt := options.FindOneAndUpdateOptions{}
			opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
			opt.SetReturnDocument(options.Before)

			var (
				playerInfo collPlayer.PlayerInfo
			)
			errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
			if errSession != nil {
				return 0, 0, errSession
			}
			for _, v := range playerInfo.ClubData {
				if v.ClubID == clubID {
					return v.Score, v.Score + value, nil
				}
			}
			return 0, 0, fmt.Errorf("updatePlayerClubScoreFunc() not match clubID")
		}

		// 写入 日志
		writeClubLogFunc := func(c *mongo.Collection, cID int32, pID int64) (insertID primitive.ObjectID) {
			var res *mongo.InsertOneResult
			res, errSession = c.InsertOne(sctx, &collClub.DBClubScoreLog{
				CreateTime: nowTT,
				ClubID:     cID,
				PlayerID:   pID,
				Category:   collClub.LogActivityAward,
				Data:       &awardLog,
			})
			if errSession != nil {
				return primitive.NilObjectID
			}

			return res.InsertedID.(primitive.ObjectID)
		}
		// 更新统计
		updatePlayerTotalFunc := func(clubID int32, uid int64, value int64) error {
			opt := options.UpdateOptions{}
			opt.SetUpsert(true)
			_, err := collPlayerTotal.UpdateOne(sctx, bson.M{"date": date__, "uid": uid, "player_clubID": clubID},
				bson.M{"$inc": bson.M{"zsS": value}}, &opt)
			return err
		}

		switch param.OperationRelation {
		case OR__MengZhu_To_MengZhu:
			param.Before_A_Score, param.Retrun_A_Score, errSession = updatePlayerClubScoreFunc(param.A_ClubID, param.A_UID, param.Value)
			if errSession != nil {
				break
			}

			errSession = updatePlayerTotalFunc(param.A_ClubID, param.A_UID, param.Value)
			if errSession != nil {
				break
			}

			awardLog.CurAScore = param.Retrun_A_Score
			//insertID := writeClubLogFunc([]*collClub.PayReceiptInfo{{ClubID: param.A_ClubID, PlayerID: param.A_UID}})
			insertID := writeClubLogFunc(AcollClubScoreLog, param.A_ClubID, param.A_UID)
			if errSession != nil {
				break
			}

			errSession = updateClubScoreFunc(param.A_ClubID, param.Value, insertID)
			if errSession != nil {
				break
			}

		case OR__MengZhu_To_ChengYun:
			fallthrough
		case OR__QuanZhu_To_ChengYun:
			param.Before_A_Score, param.Retrun_A_Score, errSession = updatePlayerClubScoreFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			param.Before_B_Score, param.Retrun_B_Score, errSession = updatePlayerClubScoreFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}

			awardLog.CurAScore = param.Retrun_A_Score
			awardLog.CurBScore = param.Retrun_B_Score

			writeClubLogFunc(AcollClubScoreLog, param.A_ClubID, param.A_UID)
			writeClubLogFunc(BcollClubScoreLog, param.B_ClubID, param.B_UID)

			//writeClubLogFunc([]*collClub.PayReceiptInfo{
			//	{ClubID: param.A_ClubID, PlayerID: param.A_UID},
			//	{ClubID: param.A_ClubID, PlayerID: param.B_UID}})

		case OR__MengZhu_To_XiaJiChengYun:
			fallthrough
		case OR__ShangJi_To_XiaJi:
			param.Before_A_Score, param.Retrun_A_Score, errSession = updatePlayerClubScoreFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			param.Before_B_Score, param.Retrun_B_Score, errSession = updatePlayerClubScoreFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.A_ClubID, param.A_UID, -param.Value)
			if errSession != nil {
				break
			}
			errSession = updatePlayerTotalFunc(param.B_ClubID, param.B_UID, param.Value)
			if errSession != nil {
				break
			}

			awardLog.CurAScore = param.Retrun_A_Score
			awardLog.CurBScore = param.Retrun_B_Score

			//insertID := writeClubLogFunc([]*collClub.PayReceiptInfo{
			//	{ClubID: param.A_ClubID, PlayerID: param.A_UID},
			//	{ClubID: param.B_ClubID, PlayerID: param.B_UID}})
			insertID := writeClubLogFunc(AcollClubScoreLog, param.A_ClubID, param.A_UID)
			writeClubLogFunc(BcollClubScoreLog, param.B_ClubID, param.B_UID)

			if errSession != nil {
				break
			}

			var (
				tempClubID   = param.B_ClubID
				tempClubData *collClub.DBClubData
			)

			for i := 0; i < 100; i++ {
				tempClubData, errSession = LoadClub(tempClubID)
				if errSession != nil {
					break
				}

				// 更新 俱乐部 总分
				errSession = updateClubScoreFunc(tempClubData.ClubID, param.Value, insertID)
				if errSession != nil {
					break
				}

				tempClubID = tempClubData.DirectSupervisor.ClubID
				if tempClubID == param.A_ClubID {
					break
				}
			}
		default:
			errSession = fmt.Errorf("not find OperationRelation.OperationRelation:=%d", param.OperationRelation)
		}

		if errSession == nil {
			var upRest *mongo.UpdateResult
			upRest, errSession = collActivityList.UpdateOne(sctx,
				bson.M{"clubID": awardLog.A_ClubID, "uid": awardLog.B_UID, "category": category, "isGet": false},
				bson.M{"$set": bson.M{"isGet": true}})
			if upRest.ModifiedCount != 1 {
				errSession = fmt.Errorf("collActivityList.UpdateOne()")
			}
		}

		return nil, errSession
	})

	//if outSideErr == nil {
	//	PutClubScoreLog(insertLog)
	//}

	return 0, outSideErr
}
