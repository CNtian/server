package clubEvent

import (
	"encoding/json"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	commonDB "vvService/commonPackge/db"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func onCreateClubTable(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msgCreateTable := clubProto.CS_ClubCreateTable{}
	err := json.Unmarshal(msg.Data, &msgCreateTable)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	clubPlay, clubScore, memberInfo, MZClub, rspCode :=
		checkCreateClubTable(msgCreateTable.PlayerClubID, msgCreateTable.ClubPlayID, msg.SenderID)
	if rspCode != 0 {
		rsp.Status = rspCode
		return rsp
	}

	var gameServiceID string

	msgCreateTable.MZClubID = MZClub.ClubID
	msgCreateTable.PayUID = MZClub.CreatorID
	msgCreateTable.PlayID = clubPlay.GameID
	msgCreateTable.PlayConfig = clubPlay.PlayCfg
	msgCreateTable.TableConfig = clubPlay.TableCfg
	msgCreateTable.ClubConfig = clubPlay.ClubCfg
	msgCreateTable.PlayerClubScore = float64(clubScore) / commonDef.SR
	msgCreateTable.IsStop3Players = memberInfo.IsNo3
	msgCreateTable.IsStop4Players = memberInfo.IsNo4
	msgCreateTable.MaxTZCount = MZClub.MaxTZCount
	if clubPlay.RobotOpen {
		msgCreateTable.RobotJoinPlaying = clubPlay.RobotJoinPlaying
		msgCreateTable.RobotJoinReady = clubPlay.RobotJoinReady
		msgCreateTable.RobotInviteTimer = clubPlay.RobotInviteTimer
	} else {
		msgCreateTable.RobotJoinPlaying = 0
		msgCreateTable.RobotJoinReady = 0
		msgCreateTable.RobotInviteTimer = 0
	}

	memberInfo.JoinTableTime = time.Now()

	msg.Data, _ = json.Marshal(&msgCreateTable)

	gameServiceID, rsp.Status = commonDB.CheckJoinTable(db.PersonalRedisClient, msg.SenderID)
	if rsp.Status != 0 {
		if FindGameService(gameServiceID) == false {
			glog.Warning("not find game service ID.", gameServiceID)
			return rsp
		}
		// 转发 给对应的 服务 验证
		if rsp.Status == errorCodeDef.Err_In_Table_Yet {
			err = wrapMQ.PublishProto(gameServiceID, msg)
			if err != nil {
				glog.Warning("onCreateClubTable() err:=", err.Error())
			}
			return nil
		}
		return rsp
	}
	gameServiceID = findGameServiceID(clubPlay.GameID)
	if len(gameServiceID) < 1 {
		rsp.Status = errorCodeDef.Err_Not_Find_Game_Service
		glog.Warning("not find game service.", gameServiceID)
		return rsp
	}

	err = wrapMQ.PublishProto(gameServiceID, msg)
	if err != nil {
		return rsp
	}

	return nil
}

func onJoinClubTable(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msgJoinTable := clubProto.CS_ClubJoinTable{}
	err := json.Unmarshal(msg.Data, &msgJoinTable)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var (
		tableGSID, playerGSID string
		clubScore             int64
		memberInfo            *collClub.DBClubMember
	)

	rsp.Status, clubScore, memberInfo, tableGSID = checkEntryClubTable(msgJoinTable.ClubID, msgJoinTable.TableNumber, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	memberInfo.JoinTableTime = time.Now()
	msgJoinTable.PlayerClubScore = float64(clubScore) / commonDef.SR
	msg.Data, _ = json.Marshal(&msgJoinTable)

	playerGSID, rsp.Status = commonDB.CheckJoinTable(db.PersonalRedisClient, msg.SenderID)
	if rsp.Status != 0 {
		if FindGameService(playerGSID) == false {
			glog.Warning("not find game service ID.", playerGSID)
			return rsp
		}
		// 转发 给对应的 服务 验证
		if rsp.Status == errorCodeDef.Err_In_Table_Yet {
			err = wrapMQ.PublishProto(playerGSID, msg)
			if err != nil {
				glog.Warning("onJoinClubTable() err:=", err.Error())
			}
			return nil
		}
		return rsp
	}

	err = wrapMQ.PublishProto(tableGSID, msg)
	if err != nil {
		return rsp
	}
	return nil
}

func onQuickStartGame(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msgQuickStart := clubProto.CS_ClubQuickStart{}
	err := json.Unmarshal(msg.Data, &msgQuickStart)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	var clubData *collClub.DBClubData
	clubData, err = loadClubData(msgQuickStart.PlayerClubID)
	if err != nil {
		rsp.Status = errorCodeDef.ErrClubNotExist
		return rsp
	}
	if _, ok := clubData.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}
	if clubData.MengZhuClubID != 0 {
		msgQuickStart.MZClubID = clubData.MengZhuClubID
	} else {
		msgQuickStart.MZClubID = msgQuickStart.PlayerClubID
	}

	{
		mzClubData, err := loadClubData(msgQuickStart.MZClubID)
		if err != nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}
		play, ok := mzClubData.PlayIDMap[msgQuickStart.ClubPlayID]
		if ok == false {
			rsp.Status = errorCodeDef.ErrNotFindClubPlay
			return rsp
		}
		msgQuickStart.GameID = play.GameID
	}

	msg.MsgBody = &msgQuickStart
	tableSnapshotEvent.PostMaTeEvent(msg)
	return nil
}

func onForceDissolveTable(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msgBody := clubProto.CS_ForceDissolveTable{}
	err := json.Unmarshal(msg.Data, &msgBody)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	rsp.Status, _ = checkClubPower(msgBody.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	var (
		source   string
		mzClubID int32
	)
	source, mzClubID, _, _, err = commonDB.GetTableInfo(db.GameRedisClient, msgBody.TableID)
	if err != nil {
		if err == redis.Nil {
			rsp.Status = errorCodeDef.ErrClubDeskNotExist
		} else {
			glog.Warning("onForceDissolveTable() err. err:=", err.Error(), ",req:=", msgBody)
			rsp.Status = errorCodeDef.Err_Failed
		}
		return rsp
	}
	if mzClubID < 1 || len(source) < 1 {
		rsp.Status = errorCodeDef.ErrClubDeskNotExist
		return rsp
	}
	if mzClubID != msgBody.OperationClubID {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	wrapMQ.PublishProto(source, msg)

	return rsp
}

func replyQuickStart(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	msgBody := msg.MsgBody.(*clubProto.SS_ReplyQuickStart)

	defer func() {
		msg.MessageID = clubProto.ID_QuickStart
	}()

	if msgBody.TableNum > 0 {
		joinTable := clubProto.CS_ClubJoinTable{
			ClubID:      msgBody.QuickStartData.PlayerClubID,
			TableNumber: msgBody.TableNum,
			IP:          msgBody.QuickStartData.IP,
			Longitude:   msgBody.QuickStartData.Longitude,
			Latitude:    msgBody.QuickStartData.Latitude,
		}
		msg.Data, _ = json.Marshal(&joinTable)
		msg.MessageID = clubProto.ID_ClubJoinTable
		return onJoinClubTable(msg)
	} else if msgBody.TableNum == 0 {
		createTable := clubProto.CS_ClubCreateTable{
			PlayerClubID: msgBody.QuickStartData.PlayerClubID,
			ClubPlayID:   msgBody.QuickStartData.ClubPlayID,
			IP:           msgBody.QuickStartData.IP,
			Longitude:    msgBody.QuickStartData.Longitude,
			Latitude:     msgBody.QuickStartData.Latitude,
		}
		msg.Data, _ = json.Marshal(&createTable)
		msg.MessageID = clubProto.ID_ClubCreateTable
		return onCreateClubTable(msg)
	}

	rsp := &mateProto.JsonResponse{}
	rsp.Status = msgBody.TableNum
	return rsp
}

// 创建桌子前的检查
// ():(玩法,玩家排位分,是否禁止3人桌,盟主俱乐部,错误码)
func checkCreateClubTable(clubID int32, playID, uid int64) (*collClub.DBClubPlay, int64, *collClub.DBClubMember, *collClub.DBClubData, int32) {

	clubData, err := loadClubData(clubID)
	if err != nil {
		glog.Warning("not find club.clubID:=", clubID, ",err:=", err.Error())
		return nil, 0, nil, nil, errorCodeDef.ErrClubNotExist
	}
	// 是否是俱乐部成员
	playerInfo, ok := clubData.MemberMap[uid]
	if ok == false {
		glog.Warning("not club member.clubID:=", clubID, ",uid:=", uid)
		return nil, 0, nil, nil, errorCodeDef.ErrClubNotMember
	}
	// 是否冻结
	if playerInfo.Frozen.IsFrozen == true {
		return nil, 0, playerInfo, nil, errorCodeDef.ErrClubPlayerStatusIllegality
	}

	// 盟主
	mengzhuClubData := clubData
	if clubData.MengZhuClubID > 0 {
		mengzhuClubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			glog.Warning("not find club.clubID:=", clubID, ",err:=", err.Error())
			return nil, 0, playerInfo, nil, errorCodeDef.ErrClubNotExist
		}
	}
	if mengzhuClubData.IsSeal == true {
		return nil, 0, playerInfo, nil, errorCodeDef.ErrClubStatusIllegality
	}

	var (
		playRule                        *collClub.DBClubPlay
		clubPaiWeiFen, illegalClubCount int64
	)
	// 是否有对应的玩法
	playRule, ok = mengzhuClubData.PlayIDMap[playID]
	if ok == false {
		glog.Warning("not match play in club.clubID:=", clubID, ",uid:=", uid, ",playID:=", playID)
		return nil, 0, playerInfo, nil, errorCodeDef.ErrClubNotFindPlayID
	}
	// 玩法是否删除
	if playRule.IsDelete == true {
		glog.Warning("not match play in club.clubID:=", clubID, ",uid:=", uid, ",playID:=", playID)
		return nil, 0, playerInfo, nil, errorCodeDef.ErrClubNotFindPlayID
	}

	// todo... 统一查询间隔1秒  或者 联盟之间隔离  或者 根据分数差来设置查询间隔
	// 这条线 状态是否正常,(总分是否 大于 保底分)、(状态)
	superiorClubIDArr := make([]int32, 0, 6)
	tempSuperiorClubID := clubID
	for i := 0; i < 100 && tempSuperiorClubID > 0; i++ {
		superiorClubIDArr = append(superiorClubIDArr, tempSuperiorClubID)
		clubData, err = loadClubData(tempSuperiorClubID)
		if err != nil {
			glog.Warning("ClubTotalIsGreaterMinScore().clubID:=", tempSuperiorClubID)
			return nil, 0, playerInfo, nil, errorCodeDef.ErrClubNotExist
		}
		// 是否 正常状态
		if clubData.IsOpen != true || clubData.IsFrozen == true || clubData.IsStocking == true {
			return nil, 0, playerInfo, nil, errorCodeDef.ErrClubStatusIllegality
		}

		tempSuperiorClubID = clubData.DirectSupervisor.ClubID
	}

	// 普通圈子 不用检查
	if mengzhuClubData.Level < 1 {
		return playRule, clubPaiWeiFen, playerInfo, mengzhuClubData, 0
	}

	// 是否满足 最低分进入要求
	clubPaiWeiFen, err = db.GetPlayerClubScore(clubID, uid)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, 0, playerInfo, nil, errorCodeDef.ErrClubNotMember
		}
		glog.Warning("get pai wei fen error.clubID:=", clubID, ",uid:=", uid, ",playID:=", playID,
			",err:=", err.Error())
		return nil, 0, playerInfo, nil, errorCodeDef.Err_Failed
	}

	if clubPaiWeiFen < commonDef.ScoreToService(playRule.ClubRule.MinEntryScoreInt) {
		//glog.Warning("pai wei fen not enough.clubID:=", clubID, ",uid:=", uid, ",playID:=", playID,
		//	",clubPaiWeiFen:=", clubPaiWeiFen, ",MinEntry:=", playRule.ClubRule.MinEntryScoreInt)
		return nil, 0, playerInfo, nil, errorCodeDef.ErrClubPaiWeiScoreNotEnough
	}

	illegalClubCount, err = db.CheckClubTotalScore(superiorClubIDArr)
	if err != nil {
		glog.Warning("ClubTotalIsGreaterMinScore()err.clubID:=", tempSuperiorClubID, ",uid:=", uid, ",playID:=", playID,
			",err:=", err.Error())
		return nil, 0, playerInfo, nil, errorCodeDef.Err_Failed
	}
	if illegalClubCount != 0 {
		glog.Warning("ClubTotalIsGreaterMinScore().clubID:=", superiorClubIDArr, ",", illegalClubCount)
		return nil, 0, playerInfo, nil, errorCodeDef.ErrClubTotalScoreLess
	}

	return playRule, clubPaiWeiFen, playerInfo, mengzhuClubData, 0
}

// 进入桌子前的检查
// ():状态,排位分,玩家俱乐部信息,游戏服ID
func checkEntryClubTable(playerClubID, tableNumber int32, uid int64) (int32, int64, *collClub.DBClubMember, string) {

	gameServiceID, mzClubID, maxPlayers, clubPlayID, err :=
		commonDB.GetTableInfo(db.GameRedisClient, tableNumber)
	if err != nil {
		if err == redis.Nil {
			return errorCodeDef.Err_Not_Find_Game_Service, 0, nil, ""
		}
		glog.Warning("uid:=", uid, ",onJoinClubTable() err:=", err.Error())
		return errorCodeDef.Err_Not_Find_Game_Service, 0, nil, ""
	}
	if mzClubID < 1 || clubPlayID < 0 {
		glog.Warning("GetTableInfo() mzClubID,clubPlayID is empty. tableNumber:=", tableNumber, ",mzClubID:=", mzClubID, ",clubPlayID:=", clubPlayID)
		return errorCodeDef.ErrClubDeskNotExist, 0, nil, ""
	}

	if len(gameServiceID) < 1 {
		return errorCodeDef.ErrClubDeskNotExist, 0, nil, ""
	}

	var (
		playerClubData *collClub.DBClubData
		mzClubData     *collClub.DBClubData
	)
	playerClubData, err = loadClubData(playerClubID)
	if err != nil {
		glog.Warning("not find club.clubID:=", playerClubID, ",err:=", err.Error())
		return errorCodeDef.ErrClubNotExist, 0, nil, ""
	}
	// 是否是俱乐部成员
	memberData, ok := playerClubData.MemberMap[uid]
	if ok == false {
		glog.Warning("not club member.clubID:=", playerClubID, ",uid:=", uid)
		return errorCodeDef.ErrClubNotMember, 0, nil, ""
	}
	if memberData.IsNo3 == true && maxPlayers == 3 {
		return errorCodeDef.ErrClubStop3Player, 0, nil, ""
	}
	if memberData.IsNo4 == true && maxPlayers == 4 {
		return errorCodeDef.ErrClubStop3Player, 0, nil, ""
	}
	// 是否冻结
	if memberData.Frozen.IsFrozen == true {
		return errorCodeDef.ErrClubPlayerStatusIllegality, 0, nil, ""
	}

	// 盟主
	mzClubData = playerClubData
	if playerClubData.MengZhuClubID > 0 {
		mzClubData, err = loadClubData(playerClubData.MengZhuClubID)
		if err != nil {
			glog.Warning("not find club.clubID:=", playerClubData.MengZhuClubID, ",err:=", err.Error())
			return errorCodeDef.ErrClubNotExist, 0, nil, ""
		}
		//return errorCodeDef.ErrClubRelation, 0, ""
	}
	if mzClubData.ClubID != mzClubID {
		return errorCodeDef.ErrClubRelation, 0, nil, ""
	}

	var (
		playRule      *collClub.DBClubPlay
		clubPaiWeiFen int64
	)

	playRule, ok = mzClubData.PlayIDMap[clubPlayID]
	if ok == false {
		return errorCodeDef.ErrClubNotFindPlayID, 0, nil, ""
	}

	var clubData *collClub.DBClubData
	superiorClubIDArr := make([]int32, 0, 6)
	tempSuperiorClubID := playerClubID
	for i := 0; i < 100 && tempSuperiorClubID > 0; i++ {
		superiorClubIDArr = append(superiorClubIDArr, tempSuperiorClubID)
		clubData, err = loadClubData(tempSuperiorClubID)
		if err != nil {
			glog.Warning("ClubTotalIsGreaterMinScore().clubID:=", tempSuperiorClubID)
			return errorCodeDef.ErrClubNotExist, 0, nil, ""
		}
		// 是否 正常状态
		if clubData.IsOpen != true || clubData.IsFrozen == true || clubData.IsStocking == true {
			return errorCodeDef.ErrClubStatusIllegality, 0, nil, ""
		}

		tempSuperiorClubID = clubData.DirectSupervisor.ClubID
	}

	if mzClubData.Level < 1 {
		return 0, clubPaiWeiFen, memberData, gameServiceID
	}

	// 是否满足 最低分进入要求
	clubPaiWeiFen, err = db.GetPlayerClubScore(playerClubID, uid)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errorCodeDef.ErrClubNotMember, 0, nil, ""
		}
		glog.Warning("get pai wei fen error.clubID:=", playerClubID, ",uid:=", uid, ",playID:=", clubPlayID,
			",err:=", err.Error())
		return errorCodeDef.Err_Failed, 0, nil, ""
	}

	if clubPaiWeiFen < commonDef.ScoreToService(playRule.ClubRule.MinEntryScoreInt) {
		//glog.Warning("pai wei fen not enough.clubID:=", playerClubID, ",uid:=", uid, ",playID:=", clubPlayID,
		//	",clubPaiWeiFen:=", clubPaiWeiFen, ",MinEntry:=", playRule.ClubRule.MinEntryScoreInt)
		return errorCodeDef.ErrClubPaiWeiScoreNotEnough, 0, nil, ""
	}

	var illegalClubCount int64
	illegalClubCount, err = db.CheckClubTotalScore(superiorClubIDArr)
	if err != nil {
		glog.Warning("ClubTotalIsGreaterMinScore()err.clubID:=", tempSuperiorClubID, ",uid:=", uid, ",playID:=",
			",err:=", err.Error())
		return errorCodeDef.Err_Failed, 0, nil, ""
	}
	if illegalClubCount != 0 {
		glog.Warning("ClubTotalIsGreaterMinScore().clubID:=", superiorClubIDArr, ",", illegalClubCount)
		return errorCodeDef.ErrClubTotalScoreLess, 0, nil, ""
	}

	return 0, clubPaiWeiFen, memberData, gameServiceID
}
