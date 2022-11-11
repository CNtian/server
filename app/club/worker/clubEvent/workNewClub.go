package clubEvent

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"vvService/appClub/db"
	"vvService/appClub/localConfig"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDB "vvService/commonPackge/db"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func OnNewClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_NewClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if len(param.Name) > 18 || len(param.Name) < 1 {
		rsp.Status = errorCodeDef.Err_Param
		rsp.Detail = fmt.Sprintf("field name too long. %d", len(param.Name))
		return rsp
	}

	var playerClubInfo *collPlayer.PlayerInfo
	playerClubInfo, err = db.GetPlayerClubInfo(msg.SenderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
		} else {
			glog.Warning("onNewClub() err:=", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
		}
		return rsp
	}

	if len(playerClubInfo.ClubData) > 2 {
		rsp.Status = errorCodeDef.ErrClubJoinMore
		return rsp
	}
	if playerClubInfo.IsCreateClub == false {
		rsp.Status = errorCodeDef.ErrPowerNotEnough
		return rsp
	}

	var dbClubData *collClub.DBClubData
	dbClubData, err = db.NewClub(msg.SenderID, 0, param.Name, nil)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onNewClub() uid:=", msg.SenderID, ",err := ", err.Error())
		return rsp
	}

	rsp.Data = dbClubData
	return rsp
}

func upgradeClub(senderID int64, data *[]byte) (int32, error) {
	rsp := &mateProto.JsonResponse{}

	type CS_UpgradeClub struct {
		ClubID   int32 `json:"clubID"` // 发送者的 圈子ID
		PlayerID int64 `json:"uid"`    // 被提升者 玩家ID
	}

	param := CS_UpgradeClub{}
	err := json.Unmarshal(*data, &param)
	if err != nil {
		return errorCodeDef.Err_Unidentifiable, nil
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.ClubID, senderID)
	if rsp.Status != 0 {
		return rsp.Status, nil
	}
	if len(clubData.DirectSubordinate) > 50 {
		rsp.Status = errorCodeDef.ErrDirSubLimit
		return rsp.Status, nil
	}

	// 是否超过最长等级
	level := 1
	tempClubID := clubData.ClubID
	for i := 0; i < 100; i++ {
		tempClubData, tempErr := loadClubData(tempClubID)
		if tempErr != nil {
			glog.Warning("onNewClub() err:=", tempErr.Error())
			return errorCodeDef.Err_Failed, nil
		}
		if tempClubData.DirectSupervisor.ClubID < 1 {
			break
		}
		tempClubID = tempClubData.DirectSupervisor.ClubID
		level += 1
	}
	if clubData.Level > 0 && level > 5 {
		return errorCodeDef.ErrClubSubordinatesTooLong, nil
	}
	if clubData.Level == 0 && level > 1 {
		return errorCodeDef.ErrClubSubordinatesTooLong, nil
	}

	// 是否是 本圈成员
	if _, ok := clubData.MemberMap[param.PlayerID]; ok == false {
		return errorCodeDef.ErrClubRelation, nil
	}
	remark := clubData.MemberMap[param.PlayerID].Remark

	// 不能是圈主
	if clubData.CreatorID == param.PlayerID {
		return errorCodeDef.ErrClubRelation, nil
	}
	// 不能是管理员
	if _, ok := clubData.AdminMemberMap[param.PlayerID]; ok == true {
		return errorCodeDef.ErrClubRelation, nil
	}

	// 是否 在游戏中
	var playerGameInfo *commonDB.PlayerGameIntro
	playerGameInfo, err = commonDB.GetPlayerGameInfo(db.PersonalRedisClient, param.PlayerID)
	if err != nil {
		glog.Warning("commonDB.GetPlayerGameInfo() err. uid:=", param.PlayerID, " ,err:=%s", err.Error())
		return errorCodeDef.Err_Failed, nil
	}
	if playerGameInfo != nil && playerGameInfo.Table != 0 {
		return errorCodeDef.Err_In_Table_Yet, nil
	}

	var playerClubInfo *collPlayer.PlayerInfo
	playerClubInfo, err = db.GetPlayerClubInfo(param.PlayerID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
		} else {
			glog.Warning("onNewClub() err:=", err.Error())
		}
		return errorCodeDef.Err_Failed, nil
	}

	if len(playerClubInfo.ClubData) > 2 {
		return errorCodeDef.ErrClubJoinMore, nil
	}
	// 找出 目前的 俱乐部分
	var clubScore int64
	for _, v := range playerClubInfo.ClubData {
		if v.ClubID == clubData.ClubID {
			clubScore = v.Score
			break
		}
	}

	var newClubData *collClub.DBClubData
	newClubData, err = db.NewClub(param.PlayerID, 0, clubData.Name, clubData)
	if err != nil {
		glog.Warning("onNewClub() uid:=", senderID, ",err := ", err.Error())
		return errorCodeDef.Err_Failed, nil
	}

	rsp.Status, err = db.MergeClub_(param.PlayerID, newClubData.ClubID, clubData.ClubID, clubScore, remark)
	if err != nil {
		db.RealDelClubData(param.PlayerID, newClubData.ClubID)

		glog.Warning("onNewClub() uid:=", senderID, ",err := ", err.Error())
		return errorCodeDef.Err_Failed, nil
	}

	//applyClubData, _ := loadClubData(param.ApplyClubID)
	//db.PutClubOperationLog(param.OperationClubID, 3,
	//	msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
	//	&collClub.DBMergeClub{
	//		//OperClubID: param.OperationClubID, OperClubName: operationClubName,
	//		ClubID: param.ApplyClubID, ClubName: applyClubData.Name})

	NoticeReGetClub(param.PlayerID, "")

	delAllClubData(true)
	deleteAllMember(clubData)

	return 0, nil
}

func remarkMember(senderID int64, data *[]byte) (int32, error) {
	rsp := &mateProto.JsonResponse{}

	type CS_Remark struct {
		OperationClubID int32  `json:"operationClubID"` // 操作人的俱乐部ID
		PlayerID        int64  `json:"uid"`             // 玩家ID
		Remark          string `json:"remark"`          // 备注

		TargetClubID int32 `json:"clubID"` // 直属圈子
	}

	param := CS_Remark{}
	err := json.Unmarshal(*data, &param)
	if err != nil {
		return errorCodeDef.Err_Unidentifiable, nil
	}
	if len(param.Remark) > 64 {
		return errorCodeDef.Err_Param, nil
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, senderID)
	if rsp.Status != 0 {
		return rsp.Status, nil
	}

	if param.TargetClubID > 0 {
		rsp.Status = errorCodeDef.ErrClubRelation
		for i, v := range clubData.DirectSubordinate {
			if v.ClubID == param.TargetClubID {
				rsp.Status, err = db.UpdateDirectSubordinateRemark(param.OperationClubID, param.TargetClubID, param.Remark)
				if err != nil {
					glog.Warning("UpdateMemberRemark() err.", param, err.Error())
					return errorCodeDef.Err_Failed, nil
				}
				clubData.DirectSubordinate[i].Remark = param.Remark
				break
			}
		}
	} else {
		m, ok := clubData.MemberMap[param.PlayerID]
		if ok == false {
			return errorCodeDef.ErrClubNotMember, nil
		}
		rsp.Status, err = db.UpdateMemberRemark(param.OperationClubID, param.PlayerID, param.Remark)
		if err != nil {
			glog.Warning("UpdateMemberRemark() err.", param, err.Error())
			return errorCodeDef.Err_Failed, nil
		}

		if rsp.Status == 0 {
			m.Remark = param.Remark
		}
	}

	return rsp.Status, nil
}

func changedRole(msg *mateProto.MessageMaTe, data *[]byte) (int32, error) {
	rsp := &mateProto.JsonResponse{}

	param := mateProto.SS_ChangedRole{}
	err := json.Unmarshal(*data, &param)
	if err != nil {
		return errorCodeDef.Err_Unidentifiable, nil
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp.Status, nil
	}
	if clubData.DirectSupervisor.ClubID > 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}
	if clubData.ClubID != param.ClubID {
		if _, ok := clubData.SubordinatesMap[param.ClubID]; ok == false {
			return errorCodeDef.ErrIsNotDirectSupervisor, nil
		}
		subClubData, err := loadClubData(param.ClubID)
		if err != nil {
			glog.Warning(err.Error(), ", ", param.ClubID)
			return errorCodeDef.Err_Failed, nil
		}
		if _, ok := subClubData.MemberMap[param.PlayerID]; ok == false {
			return errorCodeDef.ErrClubNotMember, nil
		}
	} else {
		if _, ok := clubData.MemberMap[param.PlayerID]; ok == false {
			return errorCodeDef.ErrClubNotMember, nil
		}
	}

	// 是否在游戏中
	//playerGameInfo, err := commonDB.GetPlayerGameInfo(db.PersonalRedisClient, param.PlayerID)
	//if err != nil {
	//	glog.Warning("commonDB.GetPlayerGameInfo() err. uid:=", param.PlayerID, " ,err:=%s", err.Error())
	//	return errorCodeDef.Err_Failed, fmt.Errorf("commonDB.GetPlayerGameInfo() err. err:=%s", err.Error())
	//}
	//if playerGameInfo != nil && playerGameInfo.Table != 0 {
	//	return errorCodeDef.Err_In_Table_Yet, nil
	//}

	f_ := mateProto.MessageMaTe{SenderID: msg.SenderID,
		Source:    msg.Source,
		MessageID: mateProto.ID_ChangeRole,
		Data:      msg.Data}
	f_data, _ := json.Marshal(&f_)
	wrapMQ.ForwardTo("robot", &f_data)

	return rsp.Status, nil
}

func onGetClubData(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetClubData{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if len(param.ClubIDArr) > 3 || len(param.ClubIDArr) < 1 {
		rsp.Status, rsp.Detail = errorCodeDef.Err_Param, fmt.Sprintf("clubID :=%d", len(param.ClubIDArr))
		return rsp
	}

	clubArr := make([]collClub.DBClubData, 0, len(param.ClubIDArr))

	for _, v := range param.ClubIDArr {
		var club *collClub.DBClubData
		club, err = loadClubData(v)
		if err != nil {
			glog.Warning("onGetClubData() err:=", err.Error(), ",clubID:=", v)
			continue
		}
		clubArr = append(clubArr, *club)
		clubArr[len(clubArr)-1].PlayArr = nil
		//if club.MengZhuClubID > 0 {
		//	mzClub, _ := loadClubData(club.MengZhuClubID)
		//	if mzClub != nil {
		//		club.MZNotice = mzClub.MZNotice
		//	}
		//}
	}
	rsp.Data = clubArr

	if len(param.ClubIDArr) == 1 {
		jsonData, _ := json.Marshal(&mateProto.SS_UpdatePlayerClubTime{ClubID: param.ClubIDArr[0], UID: msg.SenderID})
		wrapMQ.PublishProto("db", &mateProto.MessageMaTe{MessageID: mateProto.ID_UpdatePlayerClubTime, Data: jsonData})
	}

	return rsp
}

func onApplyMergeClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ApplyMergeClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	rsp.Status, err, rsp.Data = checkMerge(param.OperationClubID, param.TargetClubID)
	if err != nil || rsp.Status != 0 {
		return rsp
	}

	playerNH := LoadPlayerNick_Name(msg.SenderID)
	applicant := collClub.DBApplyMergeClub{
		ApplyID:          msg.SenderID,
		ApplyHeadUrl:     playerNH.HeadURL,
		ApplyAccountName: playerNH.Nick,
		ApplyClubID:      clubData.ClubID,
		ApplyClubName:    clubData.Name,
	}

	rsp.Status, err = db.ApplyMergeClub(&applicant, param.TargetClubID)
	if err != nil {
		glog.Warning("onApplyMergeClub() err. err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	noticeClubNewMail(param.TargetClubID)
	return rsp
}

func onApplyExitLeague(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ApplyLeaveLeague{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if clubData.DirectSupervisor.ClubID < 1 {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	rsp.Status, err = checkExitLeague(clubData.ClubID)
	if rsp.Status != 0 {
		return rsp
	}
	if err != nil {
		glog.Warning("onApplyExitLeague() err. err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	playerNH := LoadPlayerNick_Name(msg.SenderID)
	applicant := collClub.DBApplyExitLeague{
		InitiatorID:          msg.SenderID,
		InitiatorHeadUrl:     playerNH.HeadURL,
		InitiatorAccountName: playerNH.Nick,
		InitiatorClubID:      clubData.ClubID,
		InitiatorClubName:    clubData.Name,
		TargetClubID:         clubData.DirectSupervisor.ClubID,
	}

	rsp.Status, err = db.ApplyExitLeague(&applicant, clubData.DirectSupervisor.ClubID)
	if err != nil {
		glog.Warning("onApplyExitLeague() err. err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	noticeClubNewMail(clubData.DirectSupervisor.ClubID)
	return rsp
}

func onCheckMergeClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_CheckApplyMergeClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if param.Pass == true {
		var overlapArr []*OverlapMember
		rsp.Status, err, overlapArr = checkMerge(param.ApplyClubID, param.OperationClubID)
		if err != nil {
			if rsp.Status != 0 {
				return rsp
			}
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning("onCheckMergeClub() err.err:=", err.Error(), ",applyID:=", param.ApplyID.Hex())
			return rsp
		}
		if rsp.Status != 0 {
			if overlapArr != nil {
				rsp.Data = overlapArr
			}
			return rsp
		}
	}

	rsp.Status, err = db.CheckMergeClub(param.ApplyID, param.ApplyClubID, param.OperationClubID, param.Pass,
		msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick, clubData.Level)
	if rsp.Status != 0 || err != nil {
		db.MailOperationFailed(param.ApplyID)
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onCheckMergeClub() err.", param, ",err:=", err.Error())
		return rsp
	}
	if rsp.Status != 0 {
		return rsp
	}

	if param.Pass == true {
		applyClubData, _ := loadClubData(param.ApplyClubID)
		db.PutClubOperationLog(param.OperationClubID, 3,
			msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
			&collClub.DBMergeClub{
				//OperClubID: param.OperationClubID, OperClubName: operationClubName,
				ClubID: param.ApplyClubID, ClubName: applyClubData.Name})

		delAllClubData(true)
	}

	if x, _ := db.CheckNewMail(clubData.ClubID); x > 0 {
		clubData.IsHadNewMail = true
	} else {
		clubData.IsHadNewMail = false
	}

	return rsp
}

func onGetClubIntro(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}
	rspBody := clubProto.SC_GetClubIntro{}
	rsp.Data = &rspBody

	param := clubProto.CS_GetClubIntro{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	var clubData *collClub.DBClubData
	clubData, err = loadClubData(param.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.ErrClubNotExist
		return rsp
	}

	rspBody.Name, rspBody.Notice = clubData.Name, clubData.Notice
	rspBody.CVN = clubData.ClubVerNum

	return rsp
}

type OverlapMember struct {
	UID     int64 `json:"uid"`
	ClubIDA int32 `json:"clubIDA"`
	ClubIDB int32 `json:"clubIDB"`
}

func checkMerge(applyClubID, targetClubID int32) (int32, error, []*OverlapMember) {
	var (
		applyClubScoreCount           int64
		applyClubData, targetClubData *collClub.DBClubData
		err                           error
	)
	applyClubData, err = loadClubData(applyClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, err, nil
	}
	// 只能单独的圈子合并
	if applyClubData.DirectSupervisor.ClubID > 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil, nil
	}
	if len(applyClubData.Subordinates) > 0 {
		return errorCodeDef.ErrSubordinateExist, nil, nil
	}

	targetClubData, err = loadClubData(targetClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, err, nil
	}

	applyClubScoreCount, err = db.GetClubCountScore(applyClubID)
	if err != nil {
		glog.Warning("onApplyMergeClub() err. err:=", err.Error(), ",clubID:=", applyClubID)
		return errorCodeDef.Err_Failed, nil, nil
	}
	if applyClubScoreCount != 0 {
		return errorCodeDef.ErrClubPaiWeiScoreNot0, nil, nil
	}

	var (
		result, levelA, levelT int32
		memberMapA             map[int64]int32
		memberMapT             map[int64]int32
	)
	// 收集 操作者 圈子 的 所有成员
	result, memberMapA, err = getClubLeagueAllMember(applyClubID)
	if err != nil {
		glog.Warning("onApplyMergeClub() err. err:=", err.Error())
		return result, err, nil
	}
	if result != 0 {
		return result, err, nil
	}

	// 收集 目标 圈子 的 所有成员
	result, memberMapT, err = getClubLeagueAllMember(targetClubID)
	if err != nil {
		glog.Warning("onApplyMergeClub() err. err:=", err.Error())
		return result, err, nil
	}
	if result != 0 {
		return result, err, nil
	}

	levelA, err = getLeagueLevel(applyClubID)
	if levelA < 1 || err != nil {
		return levelA, err, nil
	}

	levelT, err = getLeagueLevel(targetClubID)
	if levelT < 1 || err != nil {
		return levelT, err, nil
	}

	if len(targetClubData.DirectSubordinate) < 1 {
		if levelA+levelT > 6 { // 最大等级 长度6
			return errorCodeDef.ErrClubSubordinatesTooLong, nil, nil
		}
	}

	overlapArr := make([]*OverlapMember, 0)
	for k, v := range memberMapA {
		if v1, ok := memberMapT[k]; ok == true {
			overlapArr = append(overlapArr, &OverlapMember{
				UID:     k,
				ClubIDA: v,
				ClubIDB: v1,
			})
			if len(overlapArr) > 10 {
				return errorCodeDef.ErrInMergeClub, nil, overlapArr
			}
		}
	}
	if len(overlapArr) > 0 {
		return errorCodeDef.ErrInMergeClub, nil, overlapArr
	}
	return 0, nil, nil
}

// ():错误码,等级长度,错误
func getSubMaxLevel(clubID int32) (int32, int32, error) {
	clubData, err := loadClubData(clubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, 0, err
	}

	var (
		tempLevel, maxLevel, result int32
	)

	for _, v := range clubData.DirectSubordinate {
		result, tempLevel, err = getSubMaxLevel(v.ClubID)
		if result != 0 || err != nil {
			return result, 0, err
		}
		tempLevel += 1
		if tempLevel > maxLevel {
			maxLevel = tempLevel
		}
		tempLevel = 0
	}

	return result, maxLevel, nil
}

// 获取联盟等级
func getLeagueLevel(clubID int32) (int32, error) {
	var (
		supLevel, subLevel, result int32
	)

	tempClubID := clubID

	var (
		clubData *collClub.DBClubData
		err      error
	)
	for tempClubID != 0 {
		clubData, err = loadClubData(tempClubID)
		if err != nil {
			return errorCodeDef.ErrClubNotExist, err
		}
		tempClubID = clubData.DirectSupervisor.ClubID
		supLevel += 1
	}

	result, subLevel, err = getSubMaxLevel(clubID)
	if result != 0 || err != nil {
		return result, err
	}

	return supLevel + subLevel, nil
}

//():结果,成员,错误
func getClubLeagueAllMember(clubID int32) (int32, map[int64]int32, error) {
	return 0, nil, nil
	clubData, err := loadClubData(clubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil, err
	}
	memberMap := make(map[int64]int32)

	if clubData.MengZhuClubID != 0 {
		clubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			return errorCodeDef.ErrClubNotExist, nil, err
		}
	}

	allClubIDArr := clubData.Subordinates
	allClubIDArr = append(allClubIDArr, clubData.ClubID)
	for _, v := range allClubIDArr {
		clubData, err = loadClubData(v)
		if err != nil {
			return errorCodeDef.ErrClubNotExist, nil, err
		}
		for k, _ := range clubData.MemberMap {
			memberMap[k] = clubData.ClubID
		}
	}
	return 0, memberMap, nil
}

func onUpdateClubLevel(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_UpdateClubLevel{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	c, _ := strconv.Atoi(localConfig.GetConfig().ID)
	if param.ClubID != int32(c) {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}
	vc, err := loadClubData(param.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",", param.ClubID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if vc.Level == 10 {
		return rsp
	}

	err = db.UpdateClubLevel(param.ClubID, 10)
	if err != nil {
		glog.Warning(err.Error(), ",", param.ClubID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	delAllClubData(false)
	return rsp
}
