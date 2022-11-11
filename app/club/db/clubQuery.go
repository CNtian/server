package db

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
	commonDef "vvService/commonPackge"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func GetClubList(clubIDArr []int32) ([]*collClub.DBClubData, error) {
	collClubInfo := mongoDBClient.Database(databaseName).Collection(collClub.CollClubInfo)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"members": 0, "plays": 0, "member_mutex": 0})
	cur, err := collClubInfo.Find(ctx, bson.M{"club_id": bson.M{"$in": clubIDArr}}, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	arr := make([]*collClub.DBClubData, 0, len(clubIDArr))
	for cur.Next(ctx) {
		temp := collClub.DBClubData{}
		err = cur.Decode(&temp)
		if err != nil {
			glog.Warning("GetClubList() err. err:=", err.Error(), ",clubID:=", clubIDArr)
		}
		arr = append(arr, &temp)
	}
	return arr, nil
}

// 统计不可用分
func TotalClubUnusable(clubIDArr []int32) (int64, error) {
	coPlayer := mongoDBClient.Database(databaseName).Collection(collPlayer.CollPlayerInfo)
	unusableScore := int64(0)

	for _, v := range clubIDArr {
		unwind := bson.M{"$unwind": "$club_info"}
		match := bson.M{"$match": bson.M{"club_info.clubID": v, "club_info.score": bson.M{"$lt": 0}}}
		project := bson.M{"$project": bson.M{"club_info": 1}}
		group := bson.M{"$group": bson.M{"_id": "$club_info.clubID", "totalScore": bson.M{"$sum": "$club_info.score"}}}

		ctx := context.Background()
		cur, err := coPlayer.Aggregate(ctx, []interface{}{unwind, match, project, group})
		if err != nil {
			return 0, err
		}

		type BsonTotalScore struct {
			TotalScore int64 `bson:"totalScore"`
		}
		temp := BsonTotalScore{}
		for cur.Next(ctx) {
			err = cur.Decode(&temp)
			if err != nil {
				cur.Close(ctx)
				return 0, err
			}
		}
		cur.Close(ctx)
		unusableScore += temp.TotalScore
	}

	return unusableScore, nil
}

// 俱乐部分 日志
func GetClubScoreLog(mzClubID, clubID int32, date_ int, playerUID int64, logType []int32, curPage, pageSize int) ([]*collClub.DBClubScoreLog, error) {

	collName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date_)
	collClubScoreLog := mongoDBClient.Database(databaseName).Collection(collName)
	ctx := context.Background()

	//filter := bson.M{"clubIDs": clubID, "players": playerUID}
	filter := bson.M{"clubID": clubID, "playerID": playerUID}
	if len(logType) == 1 && logType[0] == 0 {

	} else {
		filter["category"] = bson.M{"$in": logType}
	}

	opt := options.FindOptions{}
	opt.SetLimit(int64(pageSize))
	opt.SetSkip(int64(curPage * pageSize))
	opt.SetSort(bson.D{{"create_time", -1}, {"_id", -1}})
	cur, err := collClubScoreLog.Find(ctx, filter, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	logArr := make([]*collClub.DBClubScoreLog, 0, pageSize)
	for cur.Next(ctx) {
		logItem := collClub.DBClubScoreLog{}

		rawValue := cur.Current.Lookup("create_time")
		logItem.CreateTime = rawValue.Time()

		rawValue = cur.Current.Lookup("category")
		logItem.Category = collClub.ClubScoreLogType(rawValue.AsInt32())

		logData := cur.Current.Lookup("data")
		switch logItem.Category {
		case collClub.LogMangeFei:
			temp := &collClub.DBClubManageFeeLog{}
			logData.Unmarshal(temp)
			temp.PayCurClubScoreText = commonDef.ScoreToClient(temp.PayCurClubScore)
			temp.ConsumeCountText = commonDef.ScoreToClient(temp.ConsumeCount)
			temp.ManageFeeScoreText = commonDef.ScoreToClient(temp.ManageFeeScore)
			temp.GotCurClubScoreText = commonDef.ScoreToClient(temp.GotCurClubScore)
			logItem.Data = temp
		case collClub.LogGame:
			temp := &collClub.DBClubGameScoreLog{}
			logData.Unmarshal(temp)
			temp.GameScoreText = commonDef.ScoreToClient(temp.GameScore)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			logItem.Data = temp
		case collClub.LogXiaoHaoValue:
			temp := &collClub.DBClubXiaoHaoScoreLog{}
			logData.Unmarshal(temp)
			temp.GameScoreText = commonDef.ScoreToClient(temp.GameScore)
			temp.XiaoHaoText = commonDef.ScoreToClient(temp.XiaoHao)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			logItem.Data = temp
		case collClub.LogCaiPan:
			temp := &collClub.DBClubJudgeLog{}
			logData.Unmarshal(temp)
			temp.ValueText = commonDef.ScoreToClient(temp.Value)
			temp.CurAScoreText = commonDef.ScoreToClient(temp.CurAScore)
			temp.CurBScoreText = commonDef.ScoreToClient(temp.CurBScore)
			logItem.Data = temp
		case collClub.LogJiangLi:
			temp := &collClub.DBClubJiangLiScoreLog{}
			logData.Unmarshal(temp)
			//temp.GameScoreText = commonDef.ScoreToClient(temp.GameScore)
			//temp.GongXianText = commonDef.ScoreToClient(temp.GongXian)
			temp.JiangLiScoreText = commonDef.ScoreToClient(temp.JiangLiScore)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			for i, _ := range temp.SubArr {
				temp.SubArr[i].JiangLiScoreText = commonDef.ScoreToClient(temp.SubArr[i].JiangLiScore)
				temp.SubArr[i].CurClubScoreText = commonDef.ScoreToClient(temp.SubArr[i].CurClubScore)
			}
			logItem.Data = temp
		case collClub.LogActivityAward:
			temp := &collClub.DBClubActivityLog{}
			logData.Unmarshal(temp)
			temp.ValueText = commonDef.ScoreToClient(temp.Value)
			temp.CurAScoreText = commonDef.ScoreToClient(temp.CurAScore)
			temp.CurBScoreText = commonDef.ScoreToClient(temp.CurBScore)
			logItem.Data = temp
		case collClub.LogBaoDi:
			temp := &collClub.DBClubBaoDiScoreLog{}
			logData.Unmarshal(temp)
			//temp.GongXianText = commonDef.ScoreToClient(temp.GongXian)
			temp.JiangLiScoreText = commonDef.ScoreToClient(temp.JiangLiScore)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			for i, _ := range temp.SubArr {
				temp.SubArr[i].JiangLiScoreText = commonDef.ScoreToClient(temp.SubArr[i].JiangLiScore)
				temp.SubArr[i].CurClubScoreText = commonDef.ScoreToClient(temp.SubArr[i].CurClubScore)
			}
			logItem.Data = temp
		case collClub.LogReceivedBD, collClub.LogReceivedJL:
			temp := &collClub.DBReceivedLog{}
			logData.Unmarshal(temp)

			temp.ScoreString = commonDef.ScoreToClient(temp.ScoreInt)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)

			logItem.Data = temp
		default:
		}

		logArr = append(logArr, &logItem)
	}
	return logArr, nil
}

// 俱乐部操作 日志
func GetClubOperationLog(clubID int32) ([]*collClub.DBClubOperationLog, error) {

	coll := mongoDBClient.Database(databaseName).Collection(collClub.CollClubOperationLog)
	ctx := context.Background()

	opt := &options.FindOptions{}
	opt.SetLimit(100).SetSort(bson.M{"date": -1})
	cur, err := coll.Find(ctx, bson.M{"club_id": clubID}, opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	logArr := make([]*collClub.DBClubOperationLog, 0, 100)
	for cur.Next(ctx) {
		logItem := collClub.DBClubOperationLog{}

		rawValue := cur.Current.Lookup("date")
		logItem.Date = rawValue.Time()

		rawValue = cur.Current.Lookup("oper_id")
		logItem.OperID = rawValue.AsInt64()

		rawValue = cur.Current.Lookup("oper_name")
		logItem.OperName = rawValue.StringValue()

		rawValue = cur.Current.Lookup("category")
		logItem.Category = rawValue.AsInt32()

		logData := cur.Current.Lookup("data")
		switch logItem.Category {
		case 1, 2, 10:
			temp := &collClub.DBPlayerJoinExitClub{}
			logData.Unmarshal(temp)
			logItem.Data = temp
		case 3, 4:
			temp := &collClub.DBMergeClub{}
			logData.Unmarshal(temp)
			logItem.Data = temp
		case 5, 6, 7:
			temp := &collClub.DBPlayUpdate{}
			logData.Unmarshal(temp)
			logItem.Data = temp
		case 8:
			temp := &collClub.DBStatusUpdate{}
			logData.Unmarshal(temp)
			logItem.Data = temp
		case 9:
			temp := &collClub.DBPowerUpdate{}
			logData.Unmarshal(temp)
			logItem.Data = temp
		case 11:
		default:

		}
		logArr = append(logArr, &logItem)
	}
	return logArr, nil
}

// 圈子战绩统计
func GetClubGameRecordTotal(clubID []int32, playerID int64, date int32, clubPlayID int64, tableID int32,
	pageSize, curPage int) ([]*dbCollectionDefine.DBGameOverRecord, time.Time, error) {

	var curDate time.Time
	year, month, day := time.Now().Date()
	curDate = time.Date(year, month, day, 0, 0, 0, 0, time.Local)
	if len(clubID) < 1 {
		return nil, curDate, nil
	}

	collName := fmt.Sprintf("%s_%d", dbCollectionDefine.CollGameRecord, date)
	collGameRecord := mongoDBClient.Database(databaseName).Collection(collName)
	ctx := context.Background()

	filter := bson.M{"players.clubID": bson.M{"$in": clubID}}
	if playerID != 0 {
		filter["players.uid"] = playerID
	}

	//_1, _2 := commonDef.GetOneDayTime(date)
	//filter["create_time"] = bson.M{"$gte": _1, "$lt": _2}

	if clubPlayID != 0 {
		filter["club_play_id"] = clubPlayID
	}
	if tableID != 0 {
		filter["table_id"] = tableID
	}

	opt := options.FindOptions{}
	opt.SetLimit(int64(pageSize)).SetSort(bson.M{"create_time": -1}).SetSkip(int64(pageSize * curPage))
	cur, err := collGameRecord.Find(ctx, filter, &opt)
	if err != nil {
		return nil, curDate, err
	}
	defer cur.Close(ctx)

	logArr := make([]*dbCollectionDefine.DBGameOverRecord, 0, pageSize)
	for cur.Next(ctx) {
		log := dbCollectionDefine.DBGameOverRecord{}
		err = cur.Decode(&log)
		if err != nil {
			glog.Warning("GetClubTotal() err. err:=", err.Error(), ",clubID:=", clubID)
			break
		}
		for _, v := range log.PlayerScore {
			v.ScoreText = commonDef.ScoreToClient(v.SScore)
		}
		logArr = append(logArr, &log)
	}
	return logArr, curDate, nil
}

/*
func GetClubGameRecordTotal2(clubID int32) ([]*dbCollectionDefine.DBGameOverRecord, time.Time, error) {
	collGameRecord := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollGameRecord)
	ctx := context.Background()

	filter := bson.M{"players.clubID": clubID}
	var curDate time.Time

	year, month, day := time.Now().Date()
	curDate = time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	_1, _2 := commonDef.GetOneDayTime(0)
	filter["create_time"] = bson.M{"$gte": _1, "$lt": _2}

	opt := options.FindOptions{}
	opt.SetLimit(100).SetSort(bson.M{"create_time": -1})
	cur, err := collGameRecord.Aggregate(ctx, bson.A{bson.M{"$match": filter}, bson.M{"$project": bson.M{""}}}, &opt)
	if err != nil {
		return nil, curDate, err
	}
	defer cur.Close(ctx)

	logArr := make([]*dbCollectionDefine.DBGameOverRecord, 0, 100)
	for cur.Next(ctx) {
		log := dbCollectionDefine.DBGameOverRecord{}
		err = cur.Decode(&log)
		if err != nil {
			glog.Warning("GetClubTotal() err. err:=", err.Error(), ",clubID:=", clubID)
			break
		}
		for _, v := range log.PlayerScore {
			v.ScoreText = commonDef.ScoreToClient(v.SScore)
		}
		logArr = append(logArr, &log)
	}
	return logArr, curDate, nil
}
*/
type ClubPlayerTotalItem struct {
	ClubID   int32  `json:"clubID"`
	ClubName string `json:"clubName"`
	UID      int64  `json:"uid"`
	Nick     string `json:"nick"`
	HeadURL  string `json:"head"`

	PlayerGameRound int32 `json:"playerGameRound"`
	//PlayerScore     int64 `json:"-"`
	//PlayerHaoKa     int64 `json:"-"`
	//PlayerGongXian  int64 `json:"-"`
	//PlayerJiangLi   int64 `json:"-"`
	//JiangLiArr []*dbCollectionDefine.ClubJiangli `json:"JL"`

	PlayerScoreText    string `json:"playerScore`
	PlayerHaoKaText    string `json:"playerHaoKa"`
	PlayerGongXianText string `json:"playerGongXian"`
	PlayerXiaoHaoText  string `json:"playerXiaoHao`
}

type GetClubPlayerTotal struct {
	Item []*ClubPlayerTotalItem `json:"item"`

	Players      int32  `json:"players"`    // 总人数
	RoundCount   int64  `json:"roundCount"` // 总局数
	HaoKaText    string `json:"haoKa"`      // 总耗卡
	ScoreText    string `json:"score"`      // 总战绩
	GongXianText string `json:"gongXian"`   // 总贡献
	JiangLiText  string `json:"jiangLi"`    // 总奖励
}

/*
// 玩家统计
func GetPlayerTotal(clubID int32, playerID int64, date int64) (*GetClubPlayerTotal, time.Time, error) {

	collGameRecord := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollGameRecord)
	ctx := context.Background()

	year, month, day := time.Now().Date()
	curDate := time.Date(year, month, day, 0, 0, 0, 0, time.Local)

	unwind := bson.M{"$unwind": "$players"}
	_1, _2 := commonDef.GetOneDayTime(date)
	filterTime := bson.M{"create_time": bson.M{"$gte": _1, "$lt": _2}}
	matchTime := bson.M{"$match": filterTime}

	filterClub := bson.M{}
	if playerID == 0 {
		filterClub["players.clubID"] = clubID
	} else {
		filterClub["players.uid"] = playerID
		filterClub["players.clubID"] = bson.M{"$ne": 0}
	}
	match2 := bson.M{"$match": filterClub}

	project := bson.M{"$project": bson.M{"players": 1, "_id": 0}}

	singlePlayerGroup := bson.M{"$group": bson.M{
		"_id":       bson.M{"uid": "$players.uid", "clubID": "$players.clubID"},
		"score":     bson.M{"$sum": "$players.score"},
		"playCount": bson.M{"$sum": 1},
		"haoka":     bson.M{"$sum": "$players.haoKa"},
		"gongXian":  bson.M{"$sum": "$players.gongXian"}}}

	totalPlayerGroup := bson.M{"$group": bson.M{
		"_id":            nil,
		"zong_zhan_ji":   bson.M{"$sum": "$players.score"},
		"zong_ju_shu":    bson.M{"$sum": 1},
		"zong_hao_ka":    bson.M{"$sum": "$players.haoKa"},
		"zong_gong_xian": bson.M{"$sum": "$players.gongXian"}}}

	renShuGroup := bson.M{"$group": bson.M{
		"_id":   bson.M{"uid": "$players.uid"},
		"RSsum": bson.M{"$sum": 1},
	}}

	facet := bson.M{"$facet": bson.M{
		//"playerList":  bson.A{singlePlayerGroup, bson.M{"$limit": param.PageSize}, bson.M{"$skip": param.PageSize * param.CurPage}},
		//"totalPlayer": bson.A{totalPlayerGroup},
		//"totalRenShu": bson.A{RenShuGroup, bson.M{"$group": bson.M{"_id": nil, "renShu": bson.M{"$sum": 1}, "juShu": bson.M{"$sum": "$RSsum"}}}},
		"playerList":  bson.A{singlePlayerGroup, bson.M{"$limit": 100}},
		"totalPlayer": bson.A{totalPlayerGroup},
		"totalRenShu": bson.A{renShuGroup, bson.M{"$group": bson.M{"_id": nil, "RS": bson.M{"$sum": 1}}}},
	},
	}

	cur, err1 := collGameRecord.Aggregate(ctx, bson.A{matchTime, unwind, project, match2, facet})
	if err1 != nil {
		return nil, curDate, err1
	}
	defer cur.Close(ctx)

	type QueryResult struct {
		PlayerList []*struct {
			ID struct {
				UID    int64 `json:"uid" bson:"uid"`
				ClubID int32 `json:"clubID" bson:"clubID"`
			} `json:"id" bson:"_id"`
			Score     int64 `json:"score" bson:"score"`
			PlayCount int32 `json:"playCount" bson:"playCount"`
			Haoka     int64 `json:"haoka" bson:"haoka"`
			GongXian  int64 `json:"gongXian" bson:"gongXian"`
		} `json:"playerList" bson:"playerList"`

		TotalPlayer []struct {
			ZongZhanJi   int64 `json:"zong_zhan_ji" bson:"zong_zhan_ji"`
			ZongJuShu    int64 `json:"zong_ju_shu" bson:"zong_ju_shu"`
			ZongHaoKa    int64 `json:"zong_hao_ka" bson:"zong_hao_ka"`
			ZongGongXian int64 `json:"zong_gong_xian" bson:"zong_gong_xian"`
		} `json:"totalPlayer" bson:"totalPlayer"`

		TotalRenShu []struct {
			RS int32 `json:"RS" bson:"RS"`
		}
	}

	res := GetClubPlayerTotal{}

	for cur.Next(ctx) {
		tempQueryResult := QueryResult{}
		err1 = cur.Decode(&tempQueryResult)
		if err1 != nil {
			glog.Warning("GetPlayerTotal(). err:=", err1.Error())
		}

		res.Item = make([]*ClubPlayerTotalItem, 0, 100)

		for _, v := range tempQueryResult.PlayerList {
			temp := ClubPlayerTotalItem{
				ClubID:          v.ID.ClubID,
				UID:             v.ID.UID,
				PlayerScore:     v.Score,
				PlayerGameRound: v.PlayCount,
				PlayerHaoKa:     v.Haoka,
				PlayerGongXian:  v.GongXian,
			}
			res.Item = append(res.Item, &temp)
		}
		if len(tempQueryResult.TotalPlayer) > 0 {
			res.ScoreText = commonDef.ScoreToClient(tempQueryResult.TotalPlayer[0].ZongZhanJi)
			res.HaoKaText = commonDef.ScoreToClient(tempQueryResult.TotalPlayer[0].ZongHaoKa)
			res.GongXianText = commonDef.ScoreToClient(tempQueryResult.TotalPlayer[0].ZongGongXian)
			res.RoundCount = tempQueryResult.TotalPlayer[0].ZongJuShu
		}
		if len(tempQueryResult.TotalRenShu) > 0 {
			res.Players = tempQueryResult.TotalRenShu[0].RS
		}
	}

	return &res, curDate, nil
}
*/

func GetPlayerTotal(clubID int32, playerID []int64, date int, total *map[int64]dbCollectionDefine.DBClubPlayerTotal) error {
	if len(playerID) < 1 {
		return nil
	}

	collGameRecord := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubPlayerTotal)
	ctx := context.Background()

	opt := options.FindOptions{}
	opt.SetProjection(bson.M{"jiang_li": 0})

	filter := bson.D{}
	if clubID == 0 {
		filter = bson.D{{"date", date}, {"uid", bson.M{"$in": playerID}}}
	} else {
		filter = bson.D{{"date", date}, {"player_clubID", clubID}, {"uid", bson.M{"$in": playerID}}}
	}

	cur, err := collGameRecord.Find(ctx, filter, &opt)
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		temp := dbCollectionDefine.DBClubPlayerTotal{}
		err = cur.Decode(&temp)
		if err != nil {
			continue
		}
		(*total)[temp.PlayerID] = temp
	}
	return nil
}

// 获取玩家对某俱乐部的奖励统计
func TotalPlayerContribution(date int64, playerIDArr []int64, clubID int32) (map[int64]int64, error) {

	collClubScoreLog := mongoDBClient.Database(databaseName).Collection(collClub.CollClubScoreLog)
	ctx := context.Background()

	_1, _2 := commonDef.GetOneDayTime(date)

	match := bson.M{"$match": bson.M{"create_time": bson.M{"$gte": _1, "$lt": _2},
		"category": collClub.LogJiangLi, "data.uid": bson.M{"$in": playerIDArr}, "data.got_clubID": clubID},
	}

	project := bson.M{"$project": bson.M{"data": 1, "_id": 0}}

	group := bson.M{"$group": bson.M{"_id": bson.M{"uid": "$data.uid"}, "total": bson.M{"$sum": "$data.jiang_li"}}}

	cur, err := collClubScoreLog.Aggregate(ctx, bson.A{match, project, &group})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	type PlayerJiangli struct {
		ID struct {
			UID int64 `bson:"uid"`
		} `bson:"_id"`
		Total int64 `bson:"total"`
	}

	var tempPlayerJiangLi PlayerJiangli

	playerMap := make(map[int64]int64)
	for cur.Next(ctx) {
		err = cur.Decode(&tempPlayerJiangLi)
		if err != nil {
			glog.Warning("TotalPlayerContribution() err.err:=", err.Error())
		} else {
			playerMap[tempPlayerJiangLi.ID.UID] = tempPlayerJiangLi.Total
		}
	}
	return playerMap, nil
}

// 俱乐部统计
func GetClubTotal(clubID []int32, date int) (map[int32]*dbCollectionDefine.DBClubTotal, error) {

	coClubTotal := mongoDBClient.Database(databaseName).Collection(dbCollectionDefine.CollClubTotal)
	ctx := context.Background()

	filter := bson.M{"club_id": bson.M{"$in": clubID}}
	filter["date"] = date

	opt := options.FindOptions{}
	opt.SetLimit(100).SetSort(bson.M{"date": -1})
	cur, err := coClubTotal.Find(ctx, filter, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	findMap := make(map[int32]*dbCollectionDefine.DBClubTotal)
	for cur.Next(ctx) {
		item := new(dbCollectionDefine.DBClubTotal)
		err = cur.Decode(item)
		if err != nil {
			glog.Warning("GetClubTotal() err. err:=", err.Error(), ",clubID:=", clubID)
			break
		}
		findMap[item.ClubID] = item
	}
	return findMap, nil
}

func GetMemberJudgeLog(mzClubID, clubID int32, date_ int, uid int64, category, curPage, pageSize int) ([]*collClub.DBClubScoreLog, error) {

	collName := fmt.Sprintf("%s_%d_%d", collClub.CollClubScoreLog, mzClubID, date_)
	collClubScoreLog := mongoDBClient.Database(databaseName).Collection(collName)
	ctx := context.Background()

	filter := bson.M{"clubID": clubID, "playerID": uid}
	if category != 0 {
		filter["category"] = category //collClub.LogCaiPan
	}

	opt := options.FindOptions{}
	opt.SetLimit(int64(pageSize))
	opt.SetSkip(int64(pageSize * curPage))
	opt.SetSort(bson.D{{"create_time", -1}, {"_id", -1}})
	cur, err := collClubScoreLog.Find(ctx, filter, &opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	logArr := make([]*collClub.DBClubScoreLog, 0, 15)
	for cur.Next(ctx) {
		logItem := collClub.DBClubScoreLog{}

		rawValue := cur.Current.Lookup("create_time")
		logItem.CreateTime = rawValue.Time()

		rawValue = cur.Current.Lookup("category")
		logItem.Category = collClub.ClubScoreLogType(rawValue.AsInt32())

		logData := cur.Current.Lookup("data")
		switch logItem.Category {
		case collClub.LogMangeFei:
			temp := &collClub.DBClubManageFeeLog{}
			logData.Unmarshal(temp)
			temp.PayCurClubScoreText = commonDef.ScoreToClient(temp.PayCurClubScore)
			temp.ManageFeeScoreText = commonDef.ScoreToClient(temp.ManageFeeScore)
			temp.GotCurClubScoreText = commonDef.ScoreToClient(temp.GotCurClubScore)
			logItem.Data = temp
		case collClub.LogGame:
			temp := &collClub.DBClubGameScoreLog{}
			logData.Unmarshal(temp)
			temp.GameScoreText = commonDef.ScoreToClient(temp.GameScore)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			logItem.Data = temp
		case collClub.LogXiaoHaoValue:
			temp := &collClub.DBClubXiaoHaoScoreLog{}
			logData.Unmarshal(temp)
			temp.GameScoreText = commonDef.ScoreToClient(temp.GameScore)
			temp.XiaoHaoText = commonDef.ScoreToClient(temp.XiaoHao)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			logItem.Data = temp
		case collClub.LogCaiPan:
			temp := &collClub.DBClubJudgeLog{}
			logData.Unmarshal(temp)
			temp.ValueText = commonDef.ScoreToClient(temp.Value)
			temp.CurAScoreText = commonDef.ScoreToClient(temp.CurAScore)
			temp.CurBScoreText = commonDef.ScoreToClient(temp.CurBScore)
			logItem.Data = temp
		case collClub.LogJiangLi:
			temp := &collClub.DBClubJiangLiScoreLog{}
			logData.Unmarshal(temp)
			//temp.GameScoreText = commonDef.ScoreToClient(temp.GameScore)
			//temp.GongXianText = commonDef.ScoreToClient(temp.GongXian)
			temp.JiangLiScoreText = commonDef.ScoreToClient(temp.JiangLiScore)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			for i, _ := range temp.SubArr {
				temp.SubArr[i].JiangLiScoreText = commonDef.ScoreToClient(temp.SubArr[i].JiangLiScore)
				temp.SubArr[i].CurClubScoreText = commonDef.ScoreToClient(temp.SubArr[i].CurClubScore)
			}

			logItem.Data = temp
		case collClub.LogActivityAward:
			temp := &collClub.DBClubActivityLog{}
			logData.Unmarshal(temp)
			temp.ValueText = commonDef.ScoreToClient(temp.Value)
			temp.CurAScoreText = commonDef.ScoreToClient(temp.CurAScore)
			temp.CurBScoreText = commonDef.ScoreToClient(temp.CurBScore)
			logItem.Data = temp
		case collClub.LogBaoDi:
			temp := &collClub.DBClubBaoDiScoreLog{}
			logData.Unmarshal(temp)
			//temp.GongXianText = commonDef.ScoreToClient(temp.GongXian)
			temp.JiangLiScoreText = commonDef.ScoreToClient(temp.JiangLiScore)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)
			for i, _ := range temp.SubArr {
				temp.SubArr[i].JiangLiScoreText = commonDef.ScoreToClient(temp.SubArr[i].JiangLiScore)
				temp.SubArr[i].CurClubScoreText = commonDef.ScoreToClient(temp.SubArr[i].CurClubScore)
			}
			logItem.Data = temp
		case collClub.LogReceivedBD, collClub.LogReceivedJL:
			temp := &collClub.DBReceivedLog{}
			logData.Unmarshal(temp)

			temp.ScoreString = commonDef.ScoreToClient(temp.ScoreInt)
			temp.CurClubScoreText = commonDef.ScoreToClient(temp.CurClubScore)

			logItem.Data = temp
		default:
		}

		logArr = append(logArr, &logItem)
	}
	return logArr, nil
}

type TwoPlayerDefine struct {
	Uid int64 `json:"uid"`
	//Rounds    int32  `json:"rounds"`
	//Score     int64  `json:"-"`
	//ScoreText string `json:"score"`

	Find bool `json:"-"`
	//Index int  `json:"-"`
}

func GetTwoPlayerTogetherData(clubID, date int32, playerID []int64) ([]*dbCollectionDefine.DBGameOverRecord, error) {

	_collName := fmt.Sprintf("%s_%d", dbCollectionDefine.CollGameRecord, date)
	_coll := mongoDBClient.Database(databaseName).Collection(_collName)

	ctx := context.Background()
	cur, err := _coll.Find(ctx, bson.M{"players.uid": bson.M{"$in": playerID}})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	player0_ := &TwoPlayerDefine{Uid: playerID[0]}
	player1_ := &TwoPlayerDefine{Uid: playerID[1]}

	arr := make([]*dbCollectionDefine.DBGameOverRecord, 0, 100)

	for cur.Next(ctx) {
		_temp := dbCollectionDefine.DBGameOverRecord{}
		err = cur.Decode(&_temp)
		if err != nil {
			glog.Warning(err.Error(), ",", clubID, ",", date, ",", playerID)
			continue
		}
		if _temp.ClubID != clubID {
			continue
		}

		player0_.Find, player1_.Find = false, false

		for _, v := range _temp.PlayerScore {
			if v.UID == playerID[0] {
				player0_.Find = true
			} else if v.UID == playerID[1] {
				player1_.Find = true
			}
			if player0_.Find == true &&
				player1_.Find == true {
				break
			}
		}

		if player0_.Find == false ||
			player1_.Find == false {
			continue
		}

		arr = append(arr, &_temp)
	}

	return arr, nil
}
