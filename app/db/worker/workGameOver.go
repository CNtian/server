package worker

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"sort"
	"strconv"
	"time"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appDB/collIndex"
	"vvService/appDB/db"
	"vvService/appDB/protoDefine"
	"vvService/appDB/wrapMQ"
	commonDef "vvService/commonPackge"
	commonDB "vvService/commonPackge/db"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
)

func onGameOver(msg *mateProto.MessageMaTe) {

	gameoverRecord := protoDefine.SS_GameOverRecord{}
	err := json.Unmarshal(msg.Data, &gameoverRecord)
	if err != nil {
		glog.Warning("onGameOver() err:=", err.Error(), ",data:=", string(msg.Data))
		return
	}

	clubPlayName := ""
	tt := time.Now()

	// 俱乐部玩家 才计算
	if gameoverRecord.MZClubID > 0 && gameoverRecord.PlayerScore.Len() > 0 {

		mzClubData, err1 := loadClubData(gameoverRecord.MZClubID)
		if err1 != nil {
			glog.Warning("onGameOver() err:=", err1.Error(), ",clubID:=", gameoverRecord.MZClubID, ",data:=", string(msg.Data))
			return
		}
		calculateRebate(&tt, &gameoverRecord, gameoverRecord.RoundID)

		clubPlay, ok := mzClubData.PlayIDMap[gameoverRecord.ClubPlayID]
		if ok == false {
			glog.Warning("onGameOver() err.", ",clubID:=", gameoverRecord.MZClubID, ",data:=", string(msg.Data))
			return
		}
		clubPlayName = clubPlay.Name

		PutNewRec(&gameoverRecord)

		// 活动
		activity(&gameoverRecord)
	}

	gameRecord := dbCollectionDefine.DBGameOverRecord{
		CrateTime:    tt,
		RoundID:      gameoverRecord.RoundID,
		TableID:      gameoverRecord.TableID,
		GameID:       gameoverRecord.GameID,
		GameName:     gameoverRecord.GameName,
		ClubID:       gameoverRecord.MZClubID,
		ClubPlayID:   gameoverRecord.ClubPlayID,
		ClubPlayName: clubPlayName,
		PayerID:      gameoverRecord.PayPlayerID,
		Consumables:  gameoverRecord.ConsumeCount,
		PlayerScore:  gameoverRecord.PlayerScore,
		Begin:        gameoverRecord.Begin,
		End:          gameoverRecord.End,
	}

	year, month, day := tt.Date()
	date__, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	collName := fmt.Sprintf("%s_%d", dbCollectionDefine.CollGameRecord, date__)
	coll := db.MongoClient.Database(db.CurDBName).Collection(collName)
	_, err = coll.InsertOne(nil, &gameRecord)
	if err != nil {
		glog.Warning("onGameOver() err:=", err.Error(), ",data:=", string(msg.Data))
	}

	collIndex.FindAndCreate_game_record(coll, collName)
}

type clubTotal struct {
	Players         int64
	CreatorID       int64
	ClubScore       int64 // 俱乐部 总分变化
	GameScore       int64 // 俱乐部 玩家游戏分
	HaoKa           int64
	GongXian        int64
	SelfJiangLi     int64 // 自己的奖励
	SubJiangLiCount int64 // 子圈奖励叠加

	XiaoHao int64 // 消耗 叠加

	SelfBaoDi int64 // 保底
	//SubBaoDiCount int64 // 子圈保底叠加

	JiangLiLogArr []collClub.DBClubJiangLiScoreLog
	BaoDiLiLogArr []collClub.DBClubBaoDiScoreLog
}

func clubTotalFunc(clubMap map[int32]*clubTotal, clubID int32, scoreType collClub.ClubScoreLogType, value int64) {
	v, ok := clubMap[clubID]
	if ok == false {
		clubData, _ := loadClubData(clubID)
		v = &clubTotal{CreatorID: clubData.CreatorID}
		v.JiangLiLogArr = make([]collClub.DBClubJiangLiScoreLog, 0, 5)
		v.BaoDiLiLogArr = make([]collClub.DBClubBaoDiScoreLog, 0, 5)
		clubMap[clubID] = v
	}

	switch scoreType {
	case 0:
		v.ClubScore += value
	case collClub.LogGame:
		v.GameScore += value
	case collClub.LogJiangLi:
		v.SelfJiangLi += value
		v.Players += 1
	case collClub.LogXiaoHaoValue:
		v.XiaoHao += value
	case collClub.LogHaoKa:
		v.HaoKa += value
	case collClub.LogGongXianValue:
		v.GongXian += value
	case collClub.LogBaoDi:
		v.SelfBaoDi += value
	default:

	}
}

type unusableInfo struct {
	ClubID int32
	Score  int64
}

// 计算 返佣
func calculateRebate(ntt *time.Time, msg *protoDefine.SS_GameOverRecord, gameRecordID primitive.ObjectID) {

	mzClubData, err := loadClubData(msg.MZClubID)
	if err != nil {
		glog.Warning("calculateRebate() err:=", err.Error(), ",data:=", msg)
		return
	}

	playData, ok := mzClubData.PlayIDMap[msg.ClubPlayID]
	if ok == false {
		glog.Warning("calculateRebate() err:=", msg.ClubPlayID, ",data:=", msg)
		return
	}

	// 开始 计算
	nowTT := *ntt
	year, month, day := nowTT.Date()
	today, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	playerClubName := ""
	xiaoHaoLog := collClub.DBClubXiaoHaoScoreLog{RoundID: msg.RoundID}
	gameScoreLog := collClub.DBClubGameScoreLog{
		MZClubPlayID: msg.ClubPlayID, MZClubPlayName: playData.Name,
		RoundID: msg.RoundID}
	jiangLiLog := collClub.DBClubJiangLiScoreLog{RoundID: msg.RoundID, TableID: msg.TableID}
	baoDiLog := collClub.DBClubBaoDiScoreLog{RoundID: msg.RoundID, TableID: msg.TableID}
	clubScoreMap := make(map[int32]*clubTotal)
	unusableScoreMap := map[int64]*unusableInfo{} // key:玩家ID  value:当前俱乐部分
	haoKa := int64(0)
	tempFloat64 := float64(msg.ConsumeCount) / float64(msg.PlayerScore.Len())
	haoKa = commonDef.ScoreToService(tempFloat64)

	var levelInfo *collClub.BrokerageLevel
	bigWinnerCount := 0

	xiaoHaoFloat64 := float64(0)
	gongXianInt64, gongXianFloat64 := int64(0), float64(0)

	// 排序 取得大赢家
	lastBigWinnerScore := int64(0)
	sort.Sort(msg.PlayerScore)
	for _, v := range msg.PlayerScore {
		if bigWinnerCount > 0 {
			if v.SScore < lastBigWinnerScore {
				break
			}
		}
		v.IsMaxWinner = true
		bigWinnerCount += 1
		lastBigWinnerScore = v.SScore
	}
	if bigWinnerCount < 1 {
		bigWinnerCount = 1
	}

	// 查找阶段
	for i, _ := range playData.ClubRule.LevelArr {
		if playData.ClubRule.LevelArr[i].S <= msg.PlayerScore[0].SScore &&
			msg.PlayerScore[0].SScore < playData.ClubRule.LevelArr[i].T {
			levelInfo = &playData.ClubRule.LevelArr[i]
			break
		}
	}
	if levelInfo == nil {
		glog.Warning("not find levelIndex")
		return
	}

	if playData.ClubRule.PayWay == collClub.PaywayBigWin {
		if playData.ClubRule.BaoMingWay == collClub.BrokerageWayPercentage {
			xiaoHaoFloat64 = float64(msg.PlayerScore[0].SScore) * (levelInfo.Percent / 100)
		} else {
			xiaoHaoFloat64 = float64(levelInfo.Score)
		}

		_t := xiaoHaoFloat64 / float64(bigWinnerCount)
		for _, v := range msg.PlayerScore {
			if v.IsMaxWinner == false {
				break
			}
			v.XiaoHao = commonDef.Float64ScoreToInt64(_t)
		}
	} else if playData.ClubRule.PayWay == collClub.PaywayAllWin {
		for _, v := range msg.PlayerScore {
			if v.SScore < 0 {
				break
			}
			_t := float64(0)
			if playData.ClubRule.BaoMingWay == collClub.BrokerageWayPercentage {
				_t += float64(v.SScore) * (levelInfo.Percent / 100)
			} else {
				_t += float64(levelInfo.Score)
			}
			xiaoHaoFloat64 += _t
			v.XiaoHao = commonDef.Float64ScoreToInt64(_t)
		}
	} else {
		for _, v := range msg.PlayerScore {
			_t := float64(0)
			if playData.ClubRule.BaoMingWay == collClub.BrokerageWayPercentage {
				_t += float64(v.SScore) * (levelInfo.Percent / 100)
			} else {
				_t += float64(levelInfo.Score)
			}
			xiaoHaoFloat64 += _t
			v.XiaoHao = commonDef.Float64ScoreToInt64(_t)
		}
	}

	_tBaoDi := float64(levelInfo.BaoDi)
	if xiaoHaoFloat64-float64(levelInfo.BaoDi) < 0 {
		_tBaoDi = xiaoHaoFloat64
	}

	//baoDiFloat64 := _tBaoDi / float64(len(msg.PlayerScore))
	baoDiFloat64 := _tBaoDi
	baoDiInt64 := commonDef.Float64ScoreToInt64(baoDiFloat64)

	if mzClubData.GongXianWay == 0 {
		gongXianFloat64 = (xiaoHaoFloat64 - _tBaoDi) / float64(len(msg.PlayerScore))
		gongXianInt64 = commonDef.Float64ScoreToInt64(gongXianFloat64)
	} else {
		gongXianFloat64 = (xiaoHaoFloat64 - _tBaoDi) / float64(bigWinnerCount)
		gongXianInt64 = commonDef.Float64ScoreToInt64(gongXianFloat64)
	}

	uidArr := make([]int64, 0, len(msg.PlayerScore))
	for _, v := range msg.PlayerScore {
		uidArr = append(uidArr, v.UID)

		v.JiangLiArr = make([]*protoDefine.JiangLiDetail, 0, 10)
		v.JiangLiMap = make(map[int32]*protoDefine.JiangLiDetail)
		playerClubName = "未知"
		playerClubData, err := loadClubData(v.ClubID)
		if err != nil {
			glog.Warning("calculateRebate() err.err:=", v.ClubID)
		} else {
			v.ClubCreator = playerClubData.CreatorID
			playerClubName = playerClubData.Name
		}
		v.ClubName = playerClubName

		gameScoreLog.PlayerClubID, gameScoreLog.PlayerClubName = v.ClubID, playerClubName
		gameScoreLog.UID, gameScoreLog.Nick = v.UID, v.Nick
		gameScoreLog.GameScore = v.SScore
		gameScoreLog.IsLeave = v.IsLeave
		clubGameScoreLog(nowTT, today, mzClubData.ClubID, &gameScoreLog)
		if gameScoreLog.CurClubScore < 0 {
			unusableScoreMap[gameScoreLog.UID] = &unusableInfo{ClubID: gameScoreLog.PlayerClubID, Score: gameScoreLog.CurClubScore}
		}
		updatePlayerTotalGame(today, gameScoreLog.PlayerClubID, gameScoreLog.UID,
			gameScoreLog.MZClubPlayName, gameScoreLog.MZClubPlayID, gameScoreLog.GameScore)
	}

	// 消耗
	for _, v := range msg.PlayerScore {
		if v.XiaoHao > 0 {
			xiaoHaoLog.PlayerClubID, xiaoHaoLog.PlayerClubName = v.ClubID, playerClubName
			xiaoHaoLog.UID, xiaoHaoLog.Nick = v.UID, v.Nick
			xiaoHaoLog.GameScore, xiaoHaoLog.ClubPlayName = v.SScore, playData.Name
			xiaoHaoLog.XiaoHao = v.XiaoHao
			clubXiaoHaoLog(nowTT, today, mzClubData.ClubID, &xiaoHaoLog)
			if xiaoHaoLog.CurClubScore < 0 {
				unusableScoreMap[xiaoHaoLog.UID] = &unusableInfo{ClubID: xiaoHaoLog.PlayerClubID, Score: xiaoHaoLog.CurClubScore}
			}
			NoticeClubScoreChanged(xiaoHaoLog.PlayerClubID, xiaoHaoLog.UID, xiaoHaoLog.CurClubScore)
		} else {
			NoticeClubScoreChanged(gameScoreLog.PlayerClubID, gameScoreLog.UID, gameScoreLog.CurClubScore)
		}
	}

	// 保底
	baoDiLog.PlayerClubID, baoDiLog.PlayerClubName = mzClubData.ClubID, playerClubName
	baoDiLog.GotClubID, baoDiLog.GotID = mzClubData.ClubID, mzClubData.CreatorID
	baoDiLog.GameScore, baoDiLog.ClubPlayName = 0, playData.Name
	baoDiLog.JiangLiScore = baoDiInt64
	err = db.PushBaoDiItem(nowTT, today, msg.MZClubID, mzClubData.CreatorID, &baoDiLog)
	if err != nil {
		glog.Warning("PushBaoDiItem() err. param:=", baoDiLog, err.Error())
	}

	// 奖励
	for _, v := range msg.PlayerScore {
		jiangLiLog.PlayerClubID, jiangLiLog.PlayerClubName = v.ClubID, playerClubName
		jiangLiLog.UID, jiangLiLog.Nick = v.UID, v.Nick
		jiangLiLog.GameScore, jiangLiLog.ClubPlayName = v.SScore, playData.Name

		if mzClubData.GongXianWay == 0 || v.IsMaxWinner {
			jiangLiLog.GongXian = gongXianInt64
			v.GongXian += gongXianInt64
		} else {
			jiangLiLog.GongXian = 0
		}

		clubJiangLiLog(nowTT, today, msg.MZClubID, msg.ClubPlayID, clubScoreMap, v, &jiangLiLog)

		v.HaoKa += haoKa
	}

	// 结算
	for _, v := range msg.PlayerScore {
		temp := v.SScore - v.XiaoHao
		// 一条线的俱乐部 都要加相应的 游戏分\消耗
		for _, vJL := range v.JiangLiArr {
			if vJL.CurClubScore < 0 {
				unusableScoreMap[vJL.ClubCreator] = &unusableInfo{ClubID: vJL.ClubID, Score: vJL.CurClubScore}
			}

			clubTotalFunc(clubScoreMap, vJL.ClubID, 0, temp)
			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogHaoKa, haoKa)
			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogGame, v.SScore)
			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogGongXianValue, gongXianInt64)
			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogXiaoHaoValue, v.XiaoHao)
		}
	}

	// 更新 俱乐部总分
	for k, v := range clubScoreMap {
		// 俱乐部总分
		_, err = db.UpdateClubCountScore(k, v.ClubScore)
		if err != nil {
			glog.Warning("UpdateClubCountScore() err. clubID:=", k, ",score:=", v, ",err:=", err.Error())
		}

		// 俱乐部统计
		err = clubUpdateClubTotal(today, k, v)
		if err != nil {
			glog.Warning("UpdateClubTotal() err. ,err:=", err.Error(), ",value:=", *v)
		}

		if len(v.JiangLiLogArr) > 0 {
			err = db.PushJiangLiItem(nowTT, today, k, v.CreatorID, v.JiangLiLogArr, v.SelfJiangLi)
			if err != nil {
				glog.Warning("UpdateClubJiangLiScore() err. err:=", err.Error(), ",value:=", 0)
			}
		}
	}

	// 玩家统计
	err = updateClubPlayerTotalData(today, msg.PlayerScore, mzClubData.ClubID)
	if err != nil {
		glog.Warning("updateClubPlayerTotalData() err. ,err:=", err.Error(), ",value:=", msg.RoundID.Hex())
	}

	err = db.RedisPutMaxTongZhuoPlayer(today, uidArr)
	if err != nil {
		glog.Warning("RedisPutMaxTongZhuoPlayer() err. ,err:=", err.Error())
	}

	err = db.PutTongZhuItem(msg.MZClubID, today, uidArr, msg.RoundID)
	if err != nil {
		glog.Warning("PutTongZhuItem() err. ,err:=", err.Error())
	}

	// 俱乐部不可用分
	putClubPlayerUnusableScore(unusableScoreMap)
	updateClubUnusableScore(unusableScoreMap)
}

func GetClubPlayPercent(mzClubID, clubID int32, playID int64, clubPlayV *float64, clubBaoDiV *int32) error {
	key := fmt.Sprintf("%d_%d_%d", mzClubID, playID, clubID)
	v, ok := clubPlayPercentMap[key]
	if ok == true {
		*clubPlayV = v.ClubPlay
		*clubBaoDiV = v.ClubBaoDi
		return nil
	}

	*clubBaoDiV = 0
	*clubPlayV = float64(0)
	clubPlayPercentage := collClub.DBClubPlayPercentage{}
	err := db.GetClubPlayPercent(mzClubID, clubID, playID, &clubPlayPercentage)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	*clubBaoDiV = clubPlayPercentage.RealBaoDiPer
	*clubPlayV = clubPlayPercentage.RealPercentage

	clubPlayPercentMap[key] = &clubPercentValue{ClubPlay: *clubPlayV, ClubBaoDi: *clubBaoDiV}
	return nil
}

func clubJiangLiLog(nowTT time.Time, date_ int, mzClubID int32, clubPlayID int64, clubIDMap map[int32]*clubTotal, player *protoDefine.PlayerGameScore, param *collClub.DBClubJiangLiScoreLog) {

	// 直属上级
	directSupervisorClubID := param.PlayerClubID
	// 贡献分
	gongXian := float64(param.GongXian)
	// 下级的百分比
	lastPercentage := float64(0)
	// 贡献值记录  // 从下往上 奖励 叠加
	recGongXian := int64(0)

	//percentageText := ""

	directSupervisorPer, dsBaoDi := float64(0), int32(0)
	// 逐级往上找
	for forTime := 0; forTime < 100 && directSupervisorClubID > 0; forTime++ {
		tempClubDetail, err := loadClubData(directSupervisorClubID)
		if err != nil {
			glog.Warning("clubFanLiLog(). err:=", err.Error(), ",clubID:=", directSupervisorClubID)
			break
		}

		tempPercentage := float64(0)

		// 没有上级
		if tempClubDetail.DirectSupervisor.ClubID < 1 {
			tempPercentage = 100 - lastPercentage
			if tempPercentage < 0 {
				glog.Warning("tempPercentage<1 error.HigherClub:=", tempClubDetail.DirectSupervisor.ClubID, "lastClubID:=", directSupervisorClubID)
				break
			}
			//percentageText = fmt.Sprintf("up:%d down:%d use:%d", 100, lastPercentage, tempPercentage)
		} else {
			//tempPercentage = tempClubDetail.DirectSupervisor.Percentage - lastPercentage
			err = GetClubPlayPercent(mzClubID, directSupervisorClubID, clubPlayID, &directSupervisorPer, &dsBaoDi)
			if err != nil {
				tempPercentage = 0
				glog.Warning("GetClubPlayPercent() err.", err.Error(), ",", mzClubID, tempClubDetail.DirectSupervisor.ClubID, clubPlayID)
			} else {
				tempPercentage = directSupervisorPer - lastPercentage
			}

			if tempPercentage < 0 {
				glog.Warning("tempPercentage<1 error.directSupervisorClubID:=", directSupervisorClubID, ",pre:=", directSupervisorPer,
					",tempLastClub:=", ",pre:=", lastPercentage, ",tempPercentage:=", tempPercentage)
				break
			}
			//percentageText = fmt.Sprintf("up:%d down:%d use:%d", tempClubDetail.DirectSupervisor.Percentage, lastPercentage, tempPercentage)
			//percentageText = fmt.Sprintf("up:%d down:%d use:%d", tempClubDetail.DirectSupervisor.RealPercentage, lastPercentage, tempPercentage)

			//lastPercentage = tempClubDetail.DirectSupervisor.Percentage // 百分比
			lastPercentage = directSupervisorPer
		}

		tempJiangLi := gongXian * (float64(tempPercentage) / 100)
		param.JiangLiScore = commonDef.Float64ScoreToInt64(tempJiangLi)

		param.GotID = tempClubDetail.CreatorID
		param.GotClubID = tempClubDetail.ClubID
		//err = db.UpdateClubJiangLiScore(nowTT, date_, tempClubDetail.ClubID, param)
		//err = db.PushJiangLiItem(nowTT, date_, tempClubDetail.ClubID, tempClubDetail.CreatorID, param)
		//if err != nil {
		//	glog.Warning("UpdateClubJiangLiScore() err. err:=", err.Error(), ",value:=", param)
		//}

		updatePlayerTotalToClubJl(date_, param.PlayerClubID, param.UID, tempClubDetail.ClubID, param.JiangLiScore)

		// 统计
		clubTotalFunc(clubIDMap, directSupervisorClubID, collClub.LogJiangLi, param.JiangLiScore)
		clubIDMap[directSupervisorClubID].SubJiangLiCount = recGongXian + param.JiangLiScore
		clubIDMap[directSupervisorClubID].JiangLiLogArr = append(clubIDMap[directSupervisorClubID].JiangLiLogArr, *param)

		jl := &protoDefine.JiangLiDetail{ClubID: tempClubDetail.ClubID, ClubCreator: tempClubDetail.CreatorID,
			JiangLi: param.JiangLiScore, CurClubScore: param.CurClubScore}
		player.JiangLiArr = append(player.JiangLiArr, jl)
		player.JiangLiMap[tempClubDetail.ClubID] = jl

		// 替换 上级俱乐部
		directSupervisorClubID = tempClubDetail.DirectSupervisor.ClubID
		recGongXian += param.JiangLiScore
	}

	//if recGongXian != param.GongXian {
	//	glog.Info("clubFanLiLog() gx:=", param.GongXian, ", recGX:=", recGongXian, ",value:=", param)
	//}
}

func clubXiaoHaoLog(nowTT time.Time, date_ int, mzClubID int32, param *collClub.DBClubXiaoHaoScoreLog) {

	err := db.UpdateClubXiaoHaoScore(nowTT, date_, mzClubID, param.PlayerClubID, param)
	if err != nil {
		glog.Warning("clubXiaoHaoLog() err. err:=", err.Error(), ",value:=", param)
	}
}

func clubGameScoreLog(nowTT time.Time, date_ int, mzClubID int32, param *collClub.DBClubGameScoreLog) {

	err := db.UpdateClubGameScore(nowTT, date_, mzClubID, param.PlayerClubID, param)
	if err != nil {
		glog.Warning("clubGameScoreLog() err. err:=", err.Error(), ",value:=", param)
	}
}

func clubUpdateClubTotal(today int, clubID int32, value *clubTotal) error {
	clubTotal := dbCollectionDefine.DBClubTotal{
		Date:         int32(today),
		ClubID:       clubID,
		ClubCreator:  value.CreatorID,
		GameScore:    value.GameScore,
		HaoKa:        value.HaoKa,
		GongXian:     value.GongXian,
		JiangLi:      value.SelfJiangLi,
		BaoDi:        value.SelfBaoDi,
		XiaoHaoCount: value.XiaoHao,
	}
	return db.UpdateClubTotal(&clubTotal, value.Players)
}

func putClubPlayerUnusableScore(value map[int64]*unusableInfo) {

	var err error
	for k, v := range value {
		err = db.UpdateClubPlayerUnusableScore(k, v.Score, v.ClubID)
		if err != nil {
			glog.Warning("UpdateClubPlayerUnusableScore(). id:=", k, ",clubID:=", v.ClubID, ",err:=", err.Error())
		}
	}
}

func updateClubUnusableScore(value map[int64]*unusableInfo) {

	if len(value) < 1 {
		return
	}
	writeArr := make([]mongo.WriteModel, 0, len(value)*3)

	for _, v := range value {
		tempClubID := v.ClubID
		for i := 0; i < 100 && tempClubID > 0; i++ {
			clubData, err := loadClubData(tempClubID)
			if err != nil {
				glog.Warning("updateClubUnusableScore() clubID:=", v.ClubID, ",score:=", v.Score)
				break
			}

			dbWrite := mongo.NewUpdateOneModel()
			dbWrite.SetFilter(bson.M{"club_id": tempClubID})
			dbWrite.SetUpdate(bson.M{"$inc": bson.M{"unusable_score": v.Score}})
			writeArr = append(writeArr, dbWrite)

			tempClubID = clubData.DirectSupervisor.ClubID
		}
	}

	err := db.UpdateClubUnusableScore(writeArr)
	if err != nil {
		glog.Warning("updateClubUnusableScore(). err:=", err.Error())
	}
}

func NoticeClubScoreChanged(clubID int32, uid, score int64) {
	gateWayID, _ := commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
	if len(gateWayID) < 1 {
		return
	}

	clubScoreText := commonDef.ScoreToClient(score)
	noticeClubScoreChanged := mateProto.MessageMaTe{To: gateWayID, SenderID: uid, MessageID: clubProto.ID_CurScoreChanged}
	noticeClubScoreChanged.Data, _ = json.Marshal(&clubProto.SC_CurScoreChanged{ClubID: clubID, Score: clubScoreText})
	wrapMQ.PublishProto(gateWayID, &noticeClubScoreChanged)
}

func activity(gameoverRecord *protoDefine.SS_GameOverRecord) {
	if gameoverRecord.RuleRound != gameoverRecord.ActualRound {
		return
	}

	et, ok := _activityMap[gameoverRecord.MZClubID]
	if ok == false {
		return
	}

	noticeMsg := mateProto.MessageMaTe{MessageID: mateProto.ID_ActivityGameData}
	t := mateProto.SS_ActivityGameData{AcID: et.AcID}

	t.PlayerGameScore = make([]mateProto.ActivityPlayerGame, len(gameoverRecord.PlayerScore))
	for i, _ := range gameoverRecord.PlayerScore {
		t.PlayerGameScore[i].UID = gameoverRecord.PlayerScore[i].UID
		t.PlayerGameScore[i].Score = gameoverRecord.PlayerScore[i].SScore
	}
	noticeMsg.Data, _ = json.Marshal(&t)
	err := wrapMQ.PublishProto(fmt.Sprintf("%d", gameoverRecord.MZClubID), &noticeMsg)
	if err != nil {
		glog.Warning("mzID:=", gameoverRecord.MZClubID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", noticeMsg.MessageID, " ,data:=", string(noticeMsg.Data))
	}
}

func updatePlayerTotalToClubJl(date int, clubID int32, uid int64, toClubID int32, jlValue int64) {
	key := fmt.Sprintf("%d_%d_%d", date, clubID, uid)
	v, ok := _recPlayerToClubJL[key]
	if ok == false {
		new_ := &playerTempTotal{}
		readValue := dbCollectionDefine.DBClubPlayerTotal{}
		err := db.GetClubPlayerTotal(date, clubID, uid, &readValue)
		if err != nil && err != mongo.ErrNoDocuments {
			glog.Warning(err.Error(), ",", date, ",", clubID, ",", uid)
		}
		new_.jlItem = readValue.ClubJiangLi
		new_.GameTotal = readValue.GameTotalItem

		v = new_
		_recPlayerToClubJL[key] = v
	}
	find := false
	for i, _ := range v.jlItem {
		if v.jlItem[i].ClubID == toClubID {
			v.jlItem[i].JiangLi += jlValue
			find = true
			break
		}
	}

	if find == false {
		v.jlItem = append(v.jlItem, dbCollectionDefine.ClubJiangliItem{ClubID: toClubID, JiangLi: jlValue})
	}
}

func updatePlayerTotalGame(date int, clubID int32, uid int64, playName string, playID, gameScore int64) {
	key := fmt.Sprintf("%d_%d_%d", date, clubID, uid)
	v, ok := _recPlayerToClubJL[key]
	if ok == false {
		new_ := &playerTempTotal{}
		readValue := dbCollectionDefine.DBClubPlayerTotal{}
		err := db.GetClubPlayerTotal(date, clubID, uid, &readValue)
		if err != nil && err != mongo.ErrNoDocuments {
			glog.Warning(err.Error(), ",", date, ",", clubID, ",", uid)
		}
		new_.jlItem = readValue.ClubJiangLi
		new_.GameTotal = readValue.GameTotalItem

		v = new_
		_recPlayerToClubJL[key] = v
	}
	find := false
	for i, _ := range v.GameTotal {
		if v.GameTotal[i].PlayID == playID {
			v.GameTotal[i].RoundCount += 1
			v.GameTotal[i].TotalScore += gameScore
			find = true
			break
		}
	}

	if find == false {
		v.GameTotal = append(v.GameTotal,
			dbCollectionDefine.GameItem{PlayID: playID, PlayName: playName, RoundCount: 1, TotalScore: gameScore})
	}
}

//func calculateDuanWei(uid int64, gameScore int64) {
//	if gameScore == 0 {
//		return
//	}
//	var (
//		isWin             = false
//		isFinal           = false
//		isFinalSucceed    = 0
//		isDemotion        = false
//		isDemotionSucceed = 0
//		isBigDW, maxEXP   = 0, 0
//		ok                = false
//	)
//	if gameScore > 0 {
//		isWin = true
//	}
//
//	duanwei, exp, finalW, finalL, err := db.GetPlayerDuanWei(uid)
//	if err != nil {
//		//glog.Warning("calculateDuanWei uid:=", uid, ",gameScore:=", gameScore)
//		return
//	}
//
//	dirMap := map[int]int{ // key:段位  value:经验值
//		1: 10, 2: 12, 3: 16,
//		4: 20, 5: 24, 6: 28,
//		7: 32, 8: 36, 9: 40,
//		10: 44, 11: 48, 12: 52,
//		13: 56, 14: 60, 15: 64,
//		16: 68, 17: 72,
//	}
//
//	// 是否在晋级赛
//	maxEXP, ok = dirMap[duanwei]
//	if ok == false {
//		//glog.Warning("calculateDuanWei uid:=", uid, ",gameScore:=", gameScore)
//		return
//	}
//
//	if exp >= maxEXP {
//		if duanwei%3 == 0 {
//			isBigDW = 1
//		}
//		isFinal = true
//		if isWin {
//			finalW += 1
//		} else {
//			finalL += 1
//		}
//	} else {
//		if duanwei > 1 && exp < 1 {
//			isDemotion = true
//			if isWin {
//				finalW += 1
//			} else {
//				finalL += 1
//			}
//		} else {
//			if isWin {
//				exp += 2
//			} else {
//				exp -= 1
//			}
//		}
//	}
//
//	// 最大经验值
//	if exp > 64 {
//		exp = 64
//	}
//
//	// 晋级
//	if isFinal == true {
//		if isBigDW == 1 {
//			if finalW >= 3 {
//				isFinalSucceed = 1
//			} else if finalL >= 3 {
//				isFinalSucceed = -1
//			} else if finalL+finalW == 5 {
//				if finalW < finalL {
//					isFinalSucceed = -1
//				}
//			}
//		} else {
//			if finalW >= 2 {
//				isFinalSucceed = 1
//			} else if finalL >= 2 {
//				isFinalSucceed = -1
//			} else if finalL+finalW == 3 {
//				if finalW < finalL {
//					isFinalSucceed = -1
//				}
//			}
//		}
//	}
//
//	// 降级
//	if isDemotion == true {
//		if isBigDW == 1 {
//			if finalW >= 3 {
//				isDemotionSucceed = -1
//			} else if finalL >= 3 {
//				isDemotionSucceed = 1
//			} else if finalL+finalW == 5 {
//				if finalW < finalL {
//					isDemotionSucceed = 1
//				}
//			}
//		} else {
//			if finalW >= 2 {
//				isDemotionSucceed = -1
//			} else if finalL >= 2 {
//				isDemotionSucceed = 1
//			} else if finalL+finalW == 3 {
//				if finalW < finalL {
//					isDemotionSucceed = 1
//				}
//			}
//		}
//	}
//
//	if isFinal == true {
//		if isFinalSucceed == 1 {
//			duanwei += 1
//			exp = 3
//
//			finalW = 0
//			finalL = 0
//		} else if isFinalSucceed == -1 {
//			exp /= 2
//
//			finalW = 0
//			finalL = 0
//		}
//	}
//
//	if isDemotion == true {
//		if isDemotionSucceed == 1 {
//			duanwei -= 1
//			exp = 3
//
//			finalW = 0
//			finalL = 0
//		} else if isDemotionSucceed == -1 {
//			exp = 3
//
//			finalW = 0
//			finalL = 0
//		}
//	}
//
//	if duanwei < 1 {
//		duanwei = 1
//		exp = 0
//		finalW = 0
//		finalL = 0
//	}
//	if exp < 1 {
//		exp = 0
//	}
//
//	err = db.SetPlayerDuanWei(uid, duanwei, exp, finalW, finalL)
//	if err != nil {
//		glog.Warning("calculateDuanWei uid:=", uid, ",gameScore:=", gameScore)
//		return
//	}
//}

//func calculateRebate(ntt *time.Time, msg *protoDefine.SS_GameOverRecord, gameRecordID primitive.ObjectID) {
//
//	mzClubData, err := loadClubData(msg.MZClubID)
//	if err != nil {
//		glog.Warning("calculateRebate() err:=", err.Error(), ",data:=", msg)
//		return
//	}
//
//	playData, ok := mzClubData.PlayIDMap[msg.ClubPlayID]
//	if ok == false {
//		glog.Warning("calculateRebate() err:=", msg.ClubPlayID, ",data:=", msg)
//		return
//	}
//
//	// 开始 计算
//	nowTT := *ntt
//	year, month, day := nowTT.Date()
//	today, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
//	playerClubName := ""
//	xiaoHaoLog := collClub.DBClubXiaoHaoScoreLog{RoundID: msg.RoundID}
//	gameScoreLog := collClub.DBClubGameScoreLog{
//		MZClubPlayID: msg.ClubPlayID, MZClubPlayName: playData.Name,
//		RoundID: msg.RoundID}
//	jiangLiLog := collClub.DBClubJiangLiScoreLog{RoundID: msg.RoundID}
//	percent := playData.ClubRule.AllWinner / 100
//	clubScoreMap := make(map[int32]*clubTotal)
//	unusableScoreMap := map[int64]*unusableInfo{} // key:玩家ID  value:当前俱乐部分
//
//	haoKa := int64(0)
//	tempFloat64 := float64(msg.ConsumeCount) / float64(msg.PlayerScore.Len())
//	haoKa = commonDef.ScoreToService(tempFloat64)
//
//	{
//		sort.Sort(msg.PlayerScore)
//		msg.PlayerScore[0].IsMaxWinner = true
//		for i := 1; i < len(msg.PlayerScore); i++ {
//			if msg.PlayerScore[i].SScore < msg.PlayerScore[0].SScore {
//				break
//			}
//			msg.PlayerScore[i].IsMaxWinner = true
//		}
//	}
//
//	xiaoHaoInt64, xiaoHaoFloat64 := int64(0), float64(0)
//	gongXianInt64, gongXianFloat64 := int64(0), float64(0)
//	for _, v := range msg.PlayerScore {
//
//		v.JiangLiArr = make([]*protoDefine.JiangLiDetail, 0, 10)
//		v.JiangLiMap = make(map[int32]*protoDefine.JiangLiDetail)
//		playerClubName = "未知"
//		playerClubData, err := loadClubData(v.ClubID)
//		if err != nil {
//			glog.Warning("calculateRebate() err.err:=", v.ClubID)
//		} else {
//			v.ClubCreator = playerClubData.CreatorID
//			playerClubName = playerClubData.Name
//		}
//		v.ClubName = playerClubName
//
//		xiaoHaoInt64, xiaoHaoFloat64 = int64(0), float64(0)
//		gongXianInt64, gongXianFloat64 = int64(0), float64(0)
//
//		// 百分比
//		if playData.ClubRule.GongXianMode == 0 {
//			if v.SScore > 0 {
//				xiaoHaoFloat64 = float64(v.SScore) * percent
//			}
//			gongXianFloat64 = (math.Abs(float64(v.SScore)) * percent) / 2
//		} else {
//			// 固定
//			if v.IsMaxWinner == true {
//				gongXianFloat64 = playData.ClubRule.MaxWinner * commonDef.SR
//			} else {
//				gongXianFloat64 = playData.ClubRule.OtherPlayer * commonDef.SR
//			}
//			xiaoHaoFloat64 = gongXianFloat64
//		}
//		xiaoHaoInt64 = commonDef.Float64ScoreToInt64(xiaoHaoFloat64)
//		v.XiaoHao = xiaoHaoInt64
//		gongXianInt64 = commonDef.Float64ScoreToInt64(gongXianFloat64)
//
//		gameScoreLog.PlayerClubID, gameScoreLog.PlayerClubName = v.ClubID, playerClubName
//		gameScoreLog.UID, gameScoreLog.Nick = v.UID, v.Nick
//		gameScoreLog.GameScore = v.SScore
//		clubGameScoreLog(nowTT, today, &gameScoreLog)
//		if gameScoreLog.CurClubScore < 0 {
//			unusableScoreMap[gameScoreLog.UID] = &unusableInfo{ClubID: gameScoreLog.PlayerClubID, Score: gameScoreLog.CurClubScore}
//		}
//
//		if xiaoHaoInt64 > 0 {
//			xiaoHaoLog.PlayerClubID, xiaoHaoLog.PlayerClubName = v.ClubID, playerClubName
//			xiaoHaoLog.UID, xiaoHaoLog.Nick = v.UID, v.Nick
//			xiaoHaoLog.GameScore, xiaoHaoLog.ClubPlayName = v.SScore, playData.Name
//			xiaoHaoLog.XiaoHao = xiaoHaoInt64
//			clubXiaoHaoLog(nowTT, today, &xiaoHaoLog)
//			if xiaoHaoLog.CurClubScore < 0 {
//				unusableScoreMap[xiaoHaoLog.UID] = &unusableInfo{ClubID: xiaoHaoLog.PlayerClubID, Score: xiaoHaoLog.CurClubScore}
//			}
//
//			NoticeClubScoreChanged(xiaoHaoLog.PlayerClubID, xiaoHaoLog.UID, xiaoHaoLog.CurClubScore)
//		} else {
//			NoticeClubScoreChanged(gameScoreLog.PlayerClubID, gameScoreLog.UID, gameScoreLog.CurClubScore)
//		}
//
//		jiangLiLog.PlayerClubID, jiangLiLog.PlayerClubName = v.ClubID, playerClubName
//		jiangLiLog.UID, jiangLiLog.Nick = v.UID, v.Nick
//		jiangLiLog.GameScore, jiangLiLog.ClubPlayName = v.SScore, playData.Name
//		jiangLiLog.GongXian = gongXianInt64
//		clubJiangLiLog(nowTT, today, msg.MZClubID, msg.ClubPlayID, clubScoreMap, v, &jiangLiLog)
//
//		v.GongXian += gongXianInt64
//		v.HaoKa += haoKa
//
//		temp := v.SScore - xiaoHaoInt64
//
//		//fmt.Println("uid:=", v.UID, ",temp:=", temp, ",score:=", v.SScore, ",xiaoHao:=", xiaoHaoInt64)
//
//		// 一条线的俱乐部 都要加相应的 游戏分\消耗
//		for _, vJL := range v.JiangLiArr {
//			if vJL.CurClubScore < 0 {
//				unusableScoreMap[vJL.ClubCreator] = &unusableInfo{ClubID: vJL.ClubID, Score: vJL.CurClubScore}
//			}
//
//			clubTotalFunc(clubScoreMap, vJL.ClubID, 0, temp)
//			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogHaoKa, haoKa)
//			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogGame, v.SScore)
//			clubTotalFunc(clubScoreMap, vJL.ClubID, collClub.LogGongXianValue, gongXianInt64)
//			//fmt.Println("clubID:=", vJL.ClubID, ",data:=", *clubScoreMap[vJL.ClubID])
//		}
//		//fmt.Println("-------------------------------")
//	}
//
//	bulckWriteArr := make([]mongo.WriteModel, 0, 14)
//	preClubScore := int64(0)
//	// 更新 俱乐部总分
//	for k, v := range clubScoreMap {
//		// 俱乐部总分
//		preClubScore, err = db.UpdateClubCountScore(k, v.ClubScore)
//		if err != nil {
//			glog.Warning("UpdateClubCountScore() err. clubID:=", k, ",score:=", v, ",err:=", err.Error())
//		} else if v.ClubScore != 0 {
//			lg := mongo.NewInsertOneModel().SetDocument(dbCollectionDefine.DBClubScoreLog{ClubID: k,
//				PreValue:     preClubScore,
//				ChangedValue: v.ClubScore,
//				ID:           gameRecordID,
//				Category:     int32(collClub.LogGame)})
//			bulckWriteArr = append(bulckWriteArr, lg)
//		}
//
//		// 俱乐部统计
//		err = clubUpdateClubTotal(today, k, v)
//		if err != nil {
//			glog.Warning("UpdateClubTotal() err. ,err:=", err.Error(), ",value:=", *v)
//		}
//	}
//	if len(bulckWriteArr) > 0 {
//		db.PutClubScoreLog(bulckWriteArr)
//	}
//
//	// 玩家统计
//	err = updateClubPlayerTotalData(today, msg.PlayerScore)
//	if err != nil {
//		glog.Warning("updateClubPlayerTotalData() err. ,err:=", err.Error(), ",value:=", msg.RoundID.Hex())
//	}
//
//	// 俱乐部不可用分
//	putClubPlayerUnusableScore(unusableScoreMap)
//	updateClubUnusableScore(unusableScoreMap)
//}
