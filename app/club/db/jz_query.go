package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"time"
	"vvService/appDB/collIndex"
	commonDef "vvService/commonPackge"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func GetGameScoreRank(pageSize, curPage, date int, mzCID int32, get func(cursor *mongo.Cursor) error, selfID int64) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal)
	ctx := context.Background()

	match := bson.M{"$match": bson.D{{"date", date}, {"mzCID", mzCID}}}
	project := bson.M{"$project": bson.M{"date": 1, "uid": 1, "game_score": 1}}
	sort := bson.M{"$sort": bson.M{"game_score": -1}}

	selfValue := bson.A{bson.M{"$match": bson.M{"uid": selfID}}}

	page := bson.A{bson.M{"$skip": pageSize * curPage}, bson.M{"$limit": pageSize}}

	selfIndex := bson.A{bson.M{"$group": bson.M{"_id": nil, "all": bson.M{"$push": bson.M{"uid": "$uid"}}}},
		bson.M{"$project": bson.M{"_id": 0, "index": bson.M{"$indexOfArray": bson.A{"$all.uid", selfID}}}}}

	facet := bson.M{"$facet": bson.M{"value": selfValue, "page": page, "rank": selfIndex}}

	cur, err := coll_.Aggregate(ctx, bson.A{match, project, sort, facet})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return get(cur)
}

func GetGameRoundRank(pageSize, curPage, date int, mzCID int32, get func(cursor *mongo.Cursor) error, selfID int64) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal)
	ctx := context.Background()

	match := bson.M{"$match": bson.D{{"date", date}, {"mzCID", mzCID}}}
	project := bson.M{"$project": bson.M{"date": 1, "uid": 1, "game_count": 1}}
	sort := bson.M{"$sort": bson.M{"game_count": -1}}

	selfValue := bson.A{bson.M{"$match": bson.M{"uid": selfID}}}

	page := bson.A{bson.M{"$skip": pageSize * curPage}, bson.M{"$limit": pageSize}}

	selfIndex := bson.A{bson.M{"$group": bson.M{"_id": nil, "all": bson.M{"$push": bson.M{"uid": "$uid"}}}},
		bson.M{"$project": bson.M{"_id": 0, "index": bson.M{"$indexOfArray": bson.A{"$all.uid", selfID}}}}}

	facet := bson.M{"$facet": bson.M{"value": selfValue, "page": page, "rank": selfIndex}}

	cur, err := coll_.Aggregate(ctx, bson.A{match, project, sort, facet})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return get(cur)
}

func GetGameRecord(pageSize, curPage, date int, filter bson.D, get *[]dbCollectionDefine.DBGameOverRecord) error {

	collName := fmt.Sprintf("%s_%d", dbCollectionDefine.CollGameRecord, date)
	coll_ := mongoDBClient.Database(databaseName).Collection(collName)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetSort(bson.M{"create_time": -1})
	opt.SetLimit(int64(pageSize))
	opt.SetSkip(int64(pageSize * curPage))

	cur, err := coll_.Find(ctx, filter, &opt)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return cur.All(ctx, get)
}

func JZGetClubTotal(today, yesterday int, clubID int32, get *[]dbCollectionDefine.DBClubTotal) error {

	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubTotal)
	ctx := context.Background()

	cur, err := coll_.Find(ctx, bson.D{{"date", bson.M{"$in": []int{today, yesterday}}}, {"club_id", clubID}})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return cur.All(ctx, get)
}

func GetPlayerTotalForGame(pageSize, curPage int, date int, clubID int32, uid int64, get *[]interface{}) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal)
	ctx := context.Background()

	match := bson.M{"$match": bson.D{{"date", date}, {"player_clubID", clubID}, {"uid", uid}}}
	project := bson.M{"$project": bson.D{{"game_total", 1}, {"date", 1}}}
	unwind := bson.M{"$unwind": "$game_total"}
	//limit := bson.A{}

	cur, err := coll_.Aggregate(ctx, bson.A{match, project, unwind,
		bson.M{"$sort": bson.M{"date": -1}}, bson.M{"$limit": pageSize}, bson.M{"$skip": pageSize * curPage}})
	if err != nil {
		glog.Warning(err.Error())
		return nil
	}
	defer cur.Close(ctx)

	type gameTotalItem struct {
		Date      int `json:"date" bson:"date"`
		GameTotal struct {
			PlayName     string `json:"playName" bson:"play_name"`
			RoundCount   int    `json:"round_C" bson:"roundC"`
			GameScoreInt int64  `json:"-" bson:"tScore"`
			GameScoreStr string `json:"score" bson:"-"`

			XiaoHaoScoreInt int64  `json:"-" bson:"xiaoHao_score"`
			XiaoHaoScoreStr string `json:"xhScore" bson:"-"`
		} `json:"item" bson:"game_total"`
	}

	getArr := make([]interface{}, 0, pageSize)
	for cur.Next(ctx) {
		item_ := &gameTotalItem{}
		err = cur.Decode(item_)
		if err != nil {
			glog.Warning(err.Error())
		} else {
			item_.GameTotal.GameScoreStr = commonDef.ScoreToClient(item_.GameTotal.GameScoreInt)
			item_.GameTotal.XiaoHaoScoreStr = commonDef.ScoreToClient(item_.GameTotal.XiaoHaoScoreInt)
			getArr = append(getArr, item_)
		}
	}

	*get = getArr
	return nil
}

func GetPlayerTotalForClub(pageSize, curPage int, clubID int32, uid int64, cbFun func(*dbCollectionDefine.DBClubPlayerTotal)) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetSort(bson.M{"date": -1})
	opt.SetSkip(int64(pageSize * curPage))
	opt.SetLimit(int64(pageSize))

	cur, err := coll_.Find(ctx, bson.D{{"player_clubID", clubID}, {"uid", uid}}, &opt)
	if err != nil {
		glog.Warning(err.Error())
		return nil
	}
	defer cur.Close(ctx)

	item_ := dbCollectionDefine.DBClubPlayerTotal{}
	for cur.Next(ctx) {
		err = cur.Decode(&item_)
		if err != nil {
			glog.Warning(err.Error())
		} else {
			cbFun(&item_)
		}
	}

	return nil
}

// 保险箱
func ReceivedBaoDi(id primitive.ObjectID, safeBox *collClub.DBSafeBox, nowInt int) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(collClub.CollSafeBox)

	opt := options.FindOneAndUpdateOptions{}
	opt.SetProjection(bson.M{"club_id": 1, "uid": 1, "bd_c": 1})

	sing := coll_.FindOneAndUpdate(nil, bson.D{{"_id", id}, {"date", bson.M{"$ne": nowInt}}, {"bd_received", bson.M{"$exists": false}}},
		bson.M{"$set": bson.M{"bd_received": true}}, &opt)

	return sing.Decode(safeBox)
}

func ReceivedJiangLi(id primitive.ObjectID, safeBox *collClub.DBSafeBox, nowInt int) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(collClub.CollSafeBox)

	opt := options.FindOneAndUpdateOptions{}
	opt.SetProjection(bson.M{"club_id": 1, "uid": 1, "jl_c": 1})

	sing := coll_.FindOneAndUpdate(nil, bson.D{{"_id", id}, {"date", bson.M{"$ne": nowInt}}, {"jl_received", bson.M{"$exists": false}}},
		bson.M{"$set": bson.M{"jl_received": true}}, &opt)

	return sing.Decode(safeBox)
}

func GetSafeBoxItemList(clubID int32, uid int64, arr *[]collClub.DBSafeBox) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(collClub.CollSafeBox)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"bd_item": 0, "jl_item": 0})
	opt.SetSort(bson.M{"date": -1})

	cur, err := coll_.Find(ctx, bson.D{{"uid", uid}, {"club_id", clubID}}, &opt)
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return cur.All(ctx, arr)
}

func GetSafeBoxJiangLiItemDetail(pageSize, curPage int, id primitive.ObjectID) (interface{}, error) {
	coll_ := mongoDBClient.Database(databaseName).Collection(collClub.CollSafeBox)
	ctx := context.Background()

	//match := bson.M{"$match": bson.D{{"_id", id}}}
	//project := bson.M{"$project": bson.D{{"jl_item", 1}}}
	//unwind := bson.M{"$unwind": "$jl_item"}
	//
	//cur, err := coll_.Aggregate(ctx, bson.A{match, project, unwind,
	//	bson.M{"$sort": bson.M{"_id": -1}}, bson.M{"$limit": pageSize}, bson.M{"$skip": pageSize * curPage}})

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"jl_item": bson.M{"$slice": bson.A{bson.M{"$reverseArray": "$jl_item"}, pageSize * curPage, pageSize}}})

	cur, err := coll_.Find(ctx, bson.D{{"_id", id}}, &opt)
	if err != nil {
		glog.Warning(err.Error())
		return nil, err
	}
	defer cur.Close(ctx)

	type TempJLLogItem struct {
		CreateTime time.Time                        `json:"createTime" bson:"create_time"` // 记录时间
		Data       []collClub.DBClubJiangLiScoreLog `json:"data" bson:"data"`
	}

	type TempDBSafeBox struct {
		ID primitive.ObjectID `json:"logID" bson:"_id,omitempty"`

		BaoDi   []interface{}   `json:"baoD" bson:"bd_item"`
		JiangLi []TempJLLogItem `json:"jiangL" bson:"jl_item"`
	}

	for cur.Next(ctx) {
		item_ := &TempDBSafeBox{}
		err = cur.Decode(item_)
		if err != nil {
			glog.Warning(err.Error())
		} else {
			if len(item_.BaoDi) > 0 {
				glog.Warning("len(item_.BaoDi)>0")
			}
			for i, _ := range item_.JiangLi {
				for j, _ := range item_.JiangLi[i].Data {
					item_.JiangLi[i].Data[j].JiangLiScoreText = commonDef.ScoreToClient(item_.JiangLi[i].Data[j].JiangLiScore)
				}
			}
			return item_.JiangLi, nil
		}
	}

	return nil, nil
}

func GetSafeBoxBaoDiItemDetail(pageSize, curPage int, id primitive.ObjectID) (interface{}, error) {
	coll_ := mongoDBClient.Database(databaseName).Collection(collClub.CollSafeBox)
	ctx := context.Background()

	//match := bson.M{"$match": bson.D{{"_id", id}}}
	//project := bson.M{"$project": bson.D{{"bd_item", 1}}}
	//unwind := bson.M{"$unwind": "$bd_item"}
	//
	//cur, err := coll_.Aggregate(ctx, bson.A{match, project, unwind,
	//	bson.M{"$sort": bson.M{"_id": -1}}, bson.M{"$limit": pageSize}, bson.M{"$skip": pageSize * curPage}})
	//if err != nil {
	//	glog.Warning(err.Error())
	//	return nil
	//}
	//defer cur.Close(ctx)
	//
	//type safeBoxItem struct {
	//	Item struct {
	//	} `json:"item" bson:"bd_item"`
	//}
	//
	//getArr := make([]interface{}, 0, pageSize)
	//for cur.Next(ctx) {
	//	item_ := &safeBoxItem{}
	//	err = cur.Decode(item_)
	//	if err != nil {
	//		glog.Warning(err.Error())
	//	} else {
	//		//item_.GameTotal.GameScoreStr = commonDef.ScoreToClient(item_.GameTotal.GameScoreInt)
	//		getArr = append(getArr, item_)
	//	}
	//}
	//
	//*get = getArr
	//
	//return nil

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"jl_item": 0, "bd_item": bson.M{"$slice": bson.A{pageSize * curPage, pageSize}}})

	cur, err := coll_.Find(ctx, bson.D{{"_id", id}}, &opt)
	if err != nil {
		glog.Warning(err.Error())
		return nil, err
	}
	defer cur.Close(ctx)

	type TempBDLogItem struct {
		CreateTime time.Time                    `json:"createTime" bson:"create_time"` // 记录时间
		Data       collClub.DBClubBaoDiScoreLog `json:"data" bson:"data"`
	}

	type TempDBSafeBox struct {
		ID primitive.ObjectID `json:"logID" bson:"_id,omitempty"`

		BaoDi []TempBDLogItem `json:"bdItem" bson:"bd_item"`
	}

	for cur.Next(ctx) {
		item_ := &TempDBSafeBox{}
		err = cur.Decode(item_)
		if err != nil {
			glog.Warning(err.Error())
		} else {
			for i, _ := range item_.BaoDi {
				item_.BaoDi[i].Data.JiangLiScoreText = commonDef.ScoreToClient(item_.BaoDi[i].Data.JiangLiScore)
			}
			return item_.BaoDi, nil
		}
	}

	return nil, nil
}

func WriteReceivedSafeBoxLog(date int, mzClubID int32, dirClubID []int32, type_ collClub.ClubScoreLogType, uid, score int64) (int64, error) {
	if score == 0 {
		return 0, nil
	}

	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(3*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)

	clubScoreLogName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date)
	collClubScoreLog := mongoDBClient.Database(databaseName).Collection(clubScoreLogName, wcMajorityCollectionOpts)
	collIndex.FindAndCreate_club_score(collClubScoreLog, clubScoreLogName)

	collPlayerInfo := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo, wcMajorityCollectionOpts)
	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo, wcMajorityCollectionOpts)
	ctx := context.Background()

	playerInfo := collPlayer.PlayerInfo{}
	curScore := int64(0)

	session, outSideErr := mongoDBClient.StartSession()
	if outSideErr != nil {
		return 0, outSideErr
	}
	defer session.EndSession(ctx)

	_, outSideErr = session.WithTransaction(ctx, func(sctx mongo.SessionContext) (interface{}, error) {
		var errSession error

		// 直属上级(包括自己) 俱乐部的总分
		_, errSession = collClubInfo.UpdateMany(sctx, bson.M{"club_id": bson.M{"$in": dirClubID}}, bson.M{"$inc": bson.M{"score_count": score}})
		if errSession != nil {
			return nil, errSession
		}

		// 更新 个人 俱乐部分
		filter := bson.M{"uid": uid, "club_info": bson.M{"$elemMatch": bson.M{"clubID": dirClubID[0]}}}
		update := bson.M{"$inc": bson.M{"club_info.$.score": score}}
		opt := options.FindOneAndUpdateOptions{}
		opt.SetProjection(bson.M{"club_info": 1, "_id": 0})
		opt.SetReturnDocument(options.After)

		errSession = collPlayerInfo.FindOneAndUpdate(sctx, filter, update, &opt).Decode(&playerInfo)
		if errSession != nil {
			return nil, errSession
		}

		for _, v := range playerInfo.ClubData {
			if v.ClubID == dirClubID[0] {
				curScore = v.Score
				break
			}
		}

		// 写入 日志
		_, errSession = collClubScoreLog.InsertOne(sctx, &collClub.DBClubScoreLog{
			CreateTime: time.Now(),
			ClubID:     dirClubID[0],
			PlayerID:   uid,
			Category:   type_,
			Data:       &collClub.DBReceivedLog{ScoreInt: score, CurClubScore: curScore},
		})
		return nil, nil
	})

	return curScore, outSideErr
}

// 同桌
func GetTongZhuo(pageSize, curPage, date int, mzClubID int32, uid int64, get *[]interface{}) error {
	collName := fmt.Sprintf("%s_%d_%d", dbCollectionDefine.CollTongZhuo, mzClubID, date)
	coll_ := mongoDBClient.Database(databaseName).Collection(collName)

	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"uid": 1, "b_id": 1, "rCount": bson.M{"$size": "$r_id"}})
	opt.SetSkip(int64(pageSize * curPage))
	opt.SetLimit(int64(pageSize))

	cur, err := coll_.Find(ctx, bson.M{"$or": bson.A{bson.M{"uid": uid}, bson.M{"b_id": uid}}}, &opt)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	type TempTongZhuo struct {
		ID     primitive.ObjectID `json:"-" bson:"_id,omitempty"`
		UID    int64              `json:"uid" bson:"uid"`
		ToUID  int64              `json:"bID" bson:"b_id"`
		RCount int                `json:"rCount" bson:"rCount"`
	}

	arr := make([]interface{}, 0)
	for cur.Next(ctx) {
		temp_ := &TempTongZhuo{}
		err = cur.Decode(temp_)
		if err == nil {
			arr = append(arr, temp_)
		}
	}
	*get = arr
	return nil
}

// 获取机器人配置
func GetClubRobotCfg(mzClubID int32, arr *[]dbCollectionDefine.DBRobotClubPlayConfig) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollRobotConfig)
	ctx := context.Background()

	cur, err := coll_.Find(ctx, bson.D{{"mz_club_id", mzClubID}})
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return cur.All(ctx, arr)
}

// 获取机器人单人配置
func GetRobotItemCfg(mzClubID int32, uid int64, pageSize, curPage int64, arr *[]dbCollectionDefine.DBRobotSingle) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollRobotSingle)
	ctx := context.Background()

	filter := bson.D{{"mz_club_id", mzClubID}}
	if uid != 0 {
		filter = append(filter, primitive.E{"uid", uid})
	}

	opt := options.FindOptions{}
	opt.SetLimit(pageSize)
	opt.SetSkip(curPage * pageSize)

	cur, err := coll_.Find(ctx, filter, &opt)
	if err != nil {
		return nil
	}
	defer cur.Close(ctx)

	cur.Next(ctx)

	return cur.All(ctx, arr)
}

func DeleteRobotCfg(mzClubID int32, clubPlayID int64) {

	{
		coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollRobotSingle)
		coll_.UpdateOne(nil, bson.M{"mz_club_id": mzClubID}, bson.M{"$pull": bson.M{"item": bson.M{"club_play_id": clubPlayID}}})
	}
	{
		coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollRobotConfig)
		coll_.DeleteMany(nil, bson.D{{"mz_club_id", mzClubID}, {"club_play_id", clubPlayID}})
	}
}

func GetIsRobot(mzClubID int32, playerIDArr []int64, result *map[int64]bool) error {
	coll_ := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollRobotSingle)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"uid": 1})
	cur, err := coll_.Find(ctx, bson.D{{"mz_club_id", mzClubID}, {"uid", bson.M{"$in": playerIDArr}}}, &opt)
	if err != nil {
		return err
	}

	defer cur.Close(ctx)

	for cur.Next(ctx) {
		t_ := dbCollectionDefine.DBRobotSingle{}
		err = cur.Decode(&t_)
		if err != nil {
			glog.Warning(err.Error())
			continue
		}
		(*result)[t_.ID] = true
	}
	return nil
}
