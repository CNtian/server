package clubEvent

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"math"
	"strconv"
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

func onClubMemberOperation(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_MemberOperation{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		fmt.Println("err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	rsp.Data = msg.Data

	switch param.Action {
	case clubProto.ClubMemberOperator_FROZEN:
		rsp.Status, err = updatePlayerFrozen(msg.SenderID, &param.ActionData)
	case clubProto.ClubMemberOperator_JUDGE:
		rsp.Status, err = updateMemberScore(msg.SenderID, &param.ActionData)
	case clubProto.ClubMemberOperator_AUTHORITY:
		rsp.Status, err = updatePlayerAuthority(msg.SenderID, &param.ActionData)
	case clubProto.ClubMemberOperator_KICKOUT:
		rsp.Status, err = updateKickOutMember(msg.SenderID, &param.ActionData)
	case clubProto.ClubMemberOperator_STOP3:
		rsp.Status, err = stop3Player(msg.SenderID, &param.ActionData, 3)
	case clubProto.ClubMemberOperator_STOP4:
		rsp.Status, err = stop3Player(msg.SenderID, &param.ActionData, 4)
	case clubProto.ClubMemberOperator_UpgradeClub:
		rsp.Status, err = upgradeClub(msg.SenderID, &param.ActionData)
	case clubProto.ClubMemberOperator_Remark:
		rsp.Status, err = remarkMember(msg.SenderID, &param.ActionData)
	case clubProto.ClubMemberOperator_Robot:
		rsp.Status, err = changedRole(msg, &param.ActionData)
		if rsp.Status == 0 && err == nil {
			return nil
		}
	default:

	}
	if err != nil {
		glog.Warning("onClubMemberOperation() action:",
			param.Action, ",data:=", string(param.ActionData), ",err:=", err.Error())
		if rsp.Status == 0 {
			rsp.Status = errorCodeDef.Err_Failed
		}
	}

	return rsp
}

func updatePlayerFrozen(senderID int64, data *[]byte) (int32, error) {
	type ClubMemberOperatorFROZEN struct {
		OperationClubID int32 `json:"operationClubID"` // 操作人的俱乐部ID
		ClubID          int32 `json:"clubID"`          // 目标俱乐部ID
		UID             int64 `json:"uid"`             // 目标玩家ID
		IsFrozen        bool  `json:"isFrozen"`        // 是否冻结
	}

	actionData := ClubMemberOperatorFROZEN{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}
	var (
		clubData  *collClub.DBClubData
		isMengZhu bool
	)

	if actionData.UID == senderID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	// 验证操作人的 信息
	clubData, err = loadClubData(actionData.OperationClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, err
	}
	memberData, ok := clubData.MemberMap[senderID]
	if ok == false {
		return errorCodeDef.ErrClubNotMember, err
	}
	if clubData.DirectSupervisor.ClubID == 0 &&
		(memberData.IsAdmin == true || clubData.CreatorID == senderID) {
		isMengZhu = true
	}

	// 是否 与操作人 同一个圈子
	if _, ok = clubData.MemberMap[actionData.UID]; ok == false {
		// 是否 上下级 关系
		_, ok = clubData.SubordinatesMap[actionData.ClubID]
		if ok == false {
			return errorCodeDef.ErrClubRelation, err
		}
	}

	// 验证被操作人
	clubData, err = loadClubData(actionData.ClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}
	_, ok = clubData.MemberMap[actionData.UID]
	if err != nil {
		return errorCodeDef.ErrClubNotMember, nil
	}

	var rspCode int32
	rspCode, err = db.UpdatePlayerFrozen(actionData.ClubID, actionData.UID, senderID, actionData.IsFrozen, isMengZhu)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)
	}
	return rspCode, err
}

func updatePlayerAuthority(senderID int64, data *[]byte) (int32, error) {
	type ClubMemberOperatorAUTHORITY struct {
		ClubID  int32 `json:"clubID"` // 目标俱乐部ID
		UID     int64 `json:"uid"`    // 目标玩家ID
		IsAdmin bool  `json:"isAdmin"`
	}

	actionData := ClubMemberOperatorAUTHORITY{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if actionData.UID == senderID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	var clubData *collClub.DBClubData
	clubData, err = loadClubData(actionData.ClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}
	// 圈主 不用设置 管理员
	if clubData.CreatorID == actionData.UID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	// 圈主 亲自操作
	if clubData.CreatorID != senderID {
		return errorCodeDef.ErrPowerNotEnough, nil
	}

	var rspCode int32
	rspCode, err = db.UpdatePlayerAuthority(actionData.ClubID, actionData.UID, actionData.IsAdmin)
	if rspCode == 0 && err == nil {
		curState := int32(0)
		if actionData.IsAdmin == true {
			curState = 1
		}
		db.PutClubOperationLog(actionData.ClubID, 9,
			senderID, LoadPlayerNick_Name(senderID).Nick,
			&collClub.DBPowerUpdate{PlayerID: actionData.UID,
				PlayerNick: LoadPlayerNick_Name(actionData.UID).Nick,
				CurStatus:  curState})

		delLocalClubData(actionData.ClubID)
	}

	return rspCode, err
}

func updateKickOutMember(senderID int64, data *[]byte) (int32, error) {
	type ClubMemberOperatorKickOut struct {
		OperationClubID int32 `json:"operationClubID"` // 操作人的俱乐部ID
		ClubID          int32 `json:"clubID"`          // 目标俱乐部ID
		UID             int64 `json:"uid"`             // 目标玩家ID
	}

	actionData := ClubMemberOperatorKickOut{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if actionData.UID == senderID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	var (
		clubData           *collClub.DBClubData
		clubScore          int64
		rspCode            int32
		isMZ, isSelfMember bool
		mzClubID           int32
	)

	// 验证操作人的 信息
	rspCode, clubData = checkClubPower(actionData.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, err
	}
	if clubData.IsKickOutMember == false {
		rspCode = errorCodeDef.ErrTiRenPower
		return rspCode, err
	}

	if clubData.DirectSupervisor.ClubID < 1 {
		isMZ = true
		mzClubID = clubData.ClubID
	}

	// 是否 与操作人 同一个圈子
	if _, ok := clubData.MemberMap[actionData.UID]; ok == false {
		// 是否 上下级 关系
		_, ok = clubData.SubordinatesMap[actionData.ClubID]
		if ok == false {
			return errorCodeDef.ErrClubRelation, err
		}
		mzClubID = 0 // 取消转发
	} else {
		isSelfMember = true
	}

	if isMZ == false && isSelfMember == false {
		return errorCodeDef.ErrPowerNotEnough, err
	}

	// 验证被操作人
	clubData, err = loadClubData(actionData.ClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}
	if _, ok := clubData.MemberMap[actionData.UID]; ok == false {
		return errorCodeDef.ErrClubNotMember, nil
	}

	clubScore, err = db.GetPlayerClubScore(actionData.ClubID, actionData.UID)
	if err != nil {
		return errorCodeDef.ErrClubNotMember, nil
	}
	if clubScore < 0 || clubScore >= commonDef.SR {
		return errorCodeDef.ErrClubPaiWeiScoreNot0, nil
	}

	rspCode, err = db.ClubMemberExit(actionData.ClubID, actionData.UID, false)
	if err == nil && rspCode == 0 {
		playerNH := LoadPlayerNick_Name(senderID)
		db.PutClubOperationLog(actionData.ClubID, 10,
			senderID, playerNH.Nick,
			&collClub.DBPlayerJoinExitClub{PlayerID: actionData.UID,
				PlayerNick: LoadPlayerNick_Name(actionData.UID).Nick})

		delLocalClubData(actionData.ClubID)
		deleteAllMember(clubData)

		NoticePlayerExitClub(actionData.ClubID, actionData.UID)

		// 转发给机器人模块
		if mzClubID > 0 {
			f_ := mateProto.MessageMaTe{SenderID: senderID, MessageID: mateProto.ID_ChangeRole}
			f_.Data, _ = json.Marshal(&mateProto.SS_ChangedRole{OperationClubID: mzClubID, PlayerID: actionData.UID, Action: false})
			f_data, _ := json.Marshal(&f_)
			wrapMQ.ForwardTo("robot", &f_data)
		}
	}
	return rspCode, err
}

type ClubMemberUpdateScore struct {
	OperClubID int32  `json:"operClubID"` // 操作者的俱乐部ID
	BClubID    int32  `json:"clubID"`     // 被操作者的俱乐部ID
	BUID       int64  `json:"uid"`        // 被操作者的ID
	Value      string `json:"value"`      // 裁判分
}

func updateMemberScore(senderID int64, data *[]byte) (int32, error) {

	actionData := ClubMemberUpdateScore{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	var tempFloat64 float64
	tempFloat64, err = strconv.ParseFloat(actionData.Value, 64)
	if err != nil || math.IsNaN(tempFloat64) == true {
		return errorCodeDef.Err_Param, fmt.Errorf("value")
	}

	if tempFloat64 > 9999999 || tempFloat64 < -9999999 {
		return errorCodeDef.Err_Param, fmt.Errorf("value")
	}
	if tempFloat64 < 0.0001 && tempFloat64 > 0.0 {
		return errorCodeDef.Err_Param, fmt.Errorf("value")
	}
	if tempFloat64 < 0 && tempFloat64 > -0.0001 {
		return errorCodeDef.Err_Param, fmt.Errorf("value")
	}

	var (
		bClubData      *collClub.DBClubData
		operClubData   *collClub.DBClubData
		playerGameInfo *commonDB.PlayerGameIntro
		nowTT          = time.Now()
	)

	{ // 验证 被操作者
		bClubData, err = loadClubData(actionData.BClubID)
		if err != nil {
			glog.Warning("not find club.", actionData.BClubID)
			return errorCodeDef.ErrClubNotExist, fmt.Errorf("clubID:=%d", actionData.BClubID)
		}
		memberInfo, ok := bClubData.MemberMap[actionData.BUID]
		if ok == false {
			glog.Warning("not find member.", actionData.BClubID, actionData.BUID)
			return errorCodeDef.ErrClubNotMember, fmt.Errorf("clubID:=%d uid:=%d", actionData.BClubID, actionData.BUID)
		}
		if nowTT.Sub(memberInfo.JoinTableTime) <= time.Second*3 {
			glog.Warning("just join table.", actionData.BClubID, actionData.BUID)
			return errorCodeDef.ErrRetry, fmt.Errorf("clubID:=%d uid:=%d", actionData.BClubID, actionData.BUID)
		}
	}

	{ // 验证 操作者
		operClubData, err = loadClubData(actionData.OperClubID)
		if err != nil {
			glog.Warning("not find club.", actionData.OperClubID)
			return errorCodeDef.ErrClubNotExist, fmt.Errorf("clubID:=%d", actionData.OperClubID)
		}
		memberInfo, ok := operClubData.MemberMap[senderID]
		if ok == false {
			glog.Warning("not find member.", actionData.OperClubID, senderID)
			return errorCodeDef.ErrClubNotMember, fmt.Errorf("clubID:=%d uid:=%d", actionData.OperClubID, senderID)
		}

		if nowTT.Sub(memberInfo.JoinTableTime) <= time.Second*3 {
			glog.Warning("just join table.", actionData.OperClubID, senderID)
			return errorCodeDef.ErrRetry, fmt.Errorf("clubID:=%d uid:=%d", actionData.OperClubID, senderID)
		}
	}

	playerGameInfo, err = commonDB.GetPlayerGameInfo(db.PersonalRedisClient, senderID)
	if err != nil {
		glog.Warning("commonDB.GetPlayerGameInfo() err. uid:=", senderID, " ,err:=%s", err.Error())
		return errorCodeDef.Err_Failed, fmt.Errorf("commonDB.GetPlayerGameInfo() err. err:=%s", err.Error())
	}
	if playerGameInfo != nil && playerGameInfo.Table != 0 {
		return errorCodeDef.Err_In_Table_Yet, nil
	}
	playerGameInfo, err = commonDB.GetPlayerGameInfo(db.PersonalRedisClient, actionData.BUID)
	if err != nil {
		glog.Warning("commonDB.GetPlayerGameInfo() err. uid:=", actionData.BUID, " ,err:=%s", err.Error())
		return errorCodeDef.Err_Failed, fmt.Errorf("commonDB.GetPlayerGameInfo() err. err:=%s", err.Error())
	}
	if playerGameInfo != nil && playerGameInfo.Table != 0 {
		return errorCodeDef.Err_In_Table_Yet, nil
	}

	var (
		isQuanZhu   = false
		isQuanAdmin = false

		isSuperQuanZhu = false
		isSuperAdmin   = false

		isMengZhu      = false
		isMengZhuAdmin = false

		isOperClubMember = false
		mzClubID         int32
	)

	// 操作者 和 被操作者 是否 同一个圈子
	if _, ok := operClubData.MemberMap[actionData.BUID]; ok == true {
		isOperClubMember = true
	} else if _, ok := operClubData.SubordinatesMap[actionData.BClubID]; ok == false {
		// 是否 是同一个盟
		return errorCodeDef.ErrClubRelation, fmt.Errorf("operClubID:=%d BclubID:=%d", operClubData.ClubID, actionData.BClubID)
	}

	if operClubData.MengZhuClubID > 0 {
		mzClubID = operClubData.MengZhuClubID
	} else if operClubData.MengZhuClubID < 1 {
		mzClubID = operClubData.ClubID
	}

	{ // 确定操作人 权限
		// 没有上级 && 圈主
		if operClubData.DirectSupervisor.ClubID < 1 {
			if operClubData.CreatorID == senderID {
				isMengZhu = true
			} else if memberData, ok := operClubData.MemberMap[senderID]; ok == true && memberData.IsAdmin == true {
				isMengZhuAdmin = true
			}
		} else {
			if operClubData.CreatorID == senderID {
				if isOperClubMember == true {
					isQuanZhu = true
				} else {
					isSuperQuanZhu = true
				}
			} else if memberData, ok := operClubData.MemberMap[senderID]; ok == true && memberData.IsAdmin == true {
				if isOperClubMember == true {
					isQuanAdmin = true
				} else {
					isSuperAdmin = true
				}
			}
		}
	}

	// 非盟主,不能给自己裁判
	if isMengZhu == false && senderID == actionData.BUID {
		return errorCodeDef.ErrPowerNotEnough, nil
	}

	dbParam := db.UpdateMemberScoreParam{
		Value: commonDef.ScoreToService(tempFloat64),
	}

	updateClubScoreFunc := func() (int32, error) {
		var (
			tempRspCode int32
			gateWayID   string
		)
		tempRspCode, err = db.UpdateMemberScore(mzClubID, &dbParam)
		if tempRspCode == 0 && err == nil {
			gateWayID, _ = commonDB.LoadGatewayServiceID(db.PersonalRedisClient, dbParam.A_UID)
			if len(gateWayID) > 0 {
				clubScoreText := commonDef.ScoreToClient(dbParam.Retrun_A_Score)
				noticeClubScoreChanged := mateProto.MessageMaTe{To: gateWayID, SenderID: dbParam.A_UID, MessageID: clubProto.ID_CurScoreChanged}
				noticeClubScoreChanged.Data, _ = json.Marshal(&clubProto.SC_CurScoreChanged{ClubID: dbParam.A_ClubID, Score: clubScoreText, MZClubID: mzClubID})
				wrapMQ.PublishProto(gateWayID, &noticeClubScoreChanged)
			}

			if dbParam.A_UID != dbParam.B_UID {
				clubScoreText := commonDef.ScoreToClient(dbParam.Retrun_B_Score)
				noticeClubScoreChanged := mateProto.MessageMaTe{To: gateWayID, SenderID: dbParam.B_UID, MessageID: clubProto.ID_CurScoreChanged}
				noticeClubScoreChanged.Data, _ = json.Marshal(&clubProto.SC_CurScoreChanged{ClubID: dbParam.B_ClubID, Score: clubScoreText, MZClubID: mzClubID})

				gateWayID, _ = commonDB.LoadGatewayServiceID(db.PersonalRedisClient, dbParam.B_UID)
				if len(gateWayID) > 0 {
					wrapMQ.PublishProto(gateWayID, &noticeClubScoreChanged)
				}
				wrapMQ.PublishProto("robot", &noticeClubScoreChanged)
			}

			if dbParam.Before_B_Score < 0 {
				unusableScoreMsg := mateProto.MessageMaTe{To: "db", SenderID: dbParam.B_UID, MessageID: mateProto.ID_DeletePlayerUnusableScore}
				unusableScoreMsg.Data, _ = json.Marshal(&mateProto.SS_DeletePlayerUnusableScore{PlayerID: dbParam.B_UID, ClubID: dbParam.B_ClubID,
					CurScore: dbParam.Retrun_B_Score, BeforeScore: dbParam.Before_B_Score})
				rpcErr := wrapMQ.PublishProto("db", &unusableScoreMsg)
				if rpcErr != nil {
					glog.Warning(string(unusableScoreMsg.Data))
				}
			}
			if dbParam.Before_A_Score < 0 {
				unusableScoreMsg := mateProto.MessageMaTe{To: "db", SenderID: dbParam.A_UID, MessageID: mateProto.ID_DeletePlayerUnusableScore}
				unusableScoreMsg.Data, _ = json.Marshal(&mateProto.SS_DeletePlayerUnusableScore{PlayerID: dbParam.A_UID, ClubID: dbParam.A_ClubID,
					CurScore: dbParam.Retrun_A_Score, BeforeScore: dbParam.Before_A_Score})
				rpcErr := wrapMQ.PublishProto("db", &unusableScoreMsg)
				if rpcErr != nil {
					glog.Warning(string(unusableScoreMsg.Data))
				}
			}
		}
		return tempRspCode, err
	}

	{
		dbParam.A_ClubID, dbParam.A_ClubName = operClubData.ClubID, operClubData.Name
		dbParam.A_UID, dbParam.A_Nick = senderID, LoadPlayerNick_Name(senderID).Nick

		dbParam.B_ClubID, dbParam.B_ClubName = bClubData.ClubID, bClubData.Name
		dbParam.B_UID, dbParam.B_Nick = actionData.BUID, LoadPlayerNick_Name(actionData.BUID).Nick

		if isMengZhu == true {
			if senderID == actionData.BUID {
				// 盟主 -> 盟主自己
				dbParam.OperationRelation = db.OR__MengZhu_To_MengZhu
				return updateClubScoreFunc()
			} else if isOperClubMember == true {
				// 盟主 -> 本圈成员
				dbParam.OperationRelation = db.OR__MengZhu_To_ChengYun
				return updateClubScoreFunc()
			} else {
				// 盟主 -> 下级 成员
				dbParam.OperationRelation = db.OR__MengZhu_To_XiaJiChengYun
				return updateClubScoreFunc()
			}
			return 0, nil
		}

		if isMengZhuAdmin == true {
			if isOperClubMember == true {
				// 盟主管理员 -> 本圈成员
				dbParam.OperationRelation = db.OR__MengZhu_To_ChengYun
				return updateClubScoreFunc()
			} else {
				// 盟主管理员 -> 下级 成员
				dbParam.OperationRelation = db.OR__MengZhu_To_XiaJiChengYun
				return updateClubScoreFunc()
			}
		}

		if (isQuanZhu == true || isQuanAdmin == true) && isOperClubMember == true {
			// 圈主(管理员) -> 本圈成员
			dbParam.OperationRelation = db.OR__QuanZhu_To_ChengYun
			return updateClubScoreFunc()
		}

		if (isSuperQuanZhu == true || isSuperAdmin == true) && isOperClubMember == false {
			// 上级圈主(管理员) -> 下级 成员
			dbParam.OperationRelation = db.OR__ShangJi_To_XiaJi
			return updateClubScoreFunc()
		}
	}

	return errorCodeDef.ErrNotDefine, nil
}

func stop3Player(senderID int64, data *[]byte, players int) (int32, error) {
	type ClubMemberStop3Player struct {
		OperClubID   int32 `json:"operClubID"` // 操作者的俱乐部ID
		TargetClubID int32 `json:"clubID"`     // 被操作者的俱乐部ID
		TargetUID    int64 `json:"uid"`        // 被操作者的ID
		Value        bool  `json:"value"`      // 是否禁止玩
	}

	actionData := ClubMemberStop3Player{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if actionData.TargetUID == senderID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	// 验证操作人的 信息
	var (
		clubData *collClub.DBClubData
	)
	clubData, err = loadClubData(actionData.OperClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, err
	}
	_, ok := clubData.MemberMap[senderID]
	if ok == false {
		return errorCodeDef.ErrClubNotMember, err
	}

	// 是否 与操作人 同一个圈子
	if _, ok = clubData.MemberMap[actionData.TargetUID]; ok == false {
		// 是否 上下级 关系
		_, ok = clubData.SubordinatesMap[actionData.TargetClubID]
		if ok == false {
			return errorCodeDef.ErrClubRelation, err
		}
	}

	// 验证被操作人
	clubData, err = loadClubData(actionData.TargetClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}
	_, ok = clubData.MemberMap[actionData.TargetUID]
	if err != nil {
		return errorCodeDef.ErrClubNotMember, nil
	}

	var rspCode int32
	rspCode, err = db.UpdatePlayerStop3(actionData.TargetClubID, actionData.TargetUID, actionData.Value, players)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.TargetClubID)
	}
	return rspCode, err
}

func NoticePlayerExitClub(clubID int32, uid int64) {
	gateWayID, _ := commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
	if len(gateWayID) < 1 {
		return
	}

	noticeClubScoreChanged := mateProto.MessageMaTe{To: gateWayID, SenderID: uid, MessageID: clubProto.ID_NoticePlayerExitClub}
	noticeClubScoreChanged.Data, _ = json.Marshal(&clubProto.SC_NoticePlayerExitClub{ClubID: clubID})
	wrapMQ.PublishProto(gateWayID, &noticeClubScoreChanged)
}

func NoticeClubStockingFinish(clubID int32, uid int64, status int32) {
	gateWayID, _ := commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
	if len(gateWayID) < 1 {
		return
	}

	noticeMsg := mateProto.MessageMaTe{To: gateWayID, SenderID: uid, MessageID: clubProto.ID_NoticeClubStockingFinish}
	noticeMsg.Data, _ = json.Marshal(&mateProto.JsonResponse{Status: status})
	wrapMQ.PublishProto(gateWayID, &noticeMsg)
}

func NoticePlayerJoinClub(clubID int32, uid int64) {
	gateWayID, _ := commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
	if len(gateWayID) < 1 {
		return
	}

	noticeClubScoreChanged := mateProto.MessageMaTe{To: gateWayID, SenderID: uid, MessageID: clubProto.ID_NoticePlayerJoinClub}
	noticeClubScoreChanged.Data, _ = json.Marshal(&clubProto.SC_NoticePlayerJoinClub{ClubID: clubID})
	wrapMQ.PublishProto(gateWayID, &noticeClubScoreChanged)
}

func NoticeCreateClub(uid int64, gatewayID string) {
	if len(gatewayID) < 1 {
		gatewayID, _ = commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
		if len(gatewayID) < 1 {
			return
		}
	}

	noticeClubScoreChanged := mateProto.MessageMaTe{To: gatewayID, SenderID: uid, MessageID: clubProto.ID_CreateMengZhuClub}
	noticeClubScoreChanged.Data, _ = json.Marshal(&mateProto.JsonResponse{})
	wrapMQ.PublishProto(gatewayID, &noticeClubScoreChanged)
}

func NoticeReGetClub(uid int64, gatewayID string) {
	if len(gatewayID) < 1 {
		gatewayID, _ = commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
		if len(gatewayID) < 1 {
			return
		}
	}

	noticeClubScoreChanged := mateProto.MessageMaTe{To: gatewayID, SenderID: uid, MessageID: clubProto.ID_NoticeReGetClub}
	noticeClubScoreChanged.Data, _ = json.Marshal(&mateProto.JsonResponse{})
	wrapMQ.PublishProto(gatewayID, &noticeClubScoreChanged)
}

func onUpdateMemberOffline(msg *mateProto.MessageMaTe) {
	body := mateProto.SS_PlayerOnline{}
	err := json.Unmarshal(msg.Data, &body)
	if err != nil {
		return
	}

	if v, ok := mzMemberMap[msg.SenderID]; ok == true {
		updatePlayerOnline(v, msg.SenderID, false)
	}
}

//func onUpdateMemberOnline(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
//	rsp := &mateProto.JsonResponse{}
//
//	req := clubProto.CS_PlayerSceneChanged{}
//	err := json.Unmarshal(msg.Data, &req)
//	if err != nil {
//		rsp.Status = errorCodeDef.Err_Illegal_JSON
//		return rsp
//	}
//
//	v, err := loadClubData(req.ClubID)
//	if err != nil {
//		rsp.Status = errorCodeDef.Err_Failed
//		glog.Warning(err.Error(), ",", req.ClubID)
//		return rsp
//	}
//
//	mzID := v.MengZhuClubID
//	if v.MengZhuClubID < 1 {
//		mzID = v.ClubID
//	}
//
//	if req.ClubID == 0 {
//		updatePlayerOnline(mzID, msg.SenderID, false)
//	} else {
//		updatePlayerOnline(mzID, msg.SenderID, true)
//	}
//
//	return rsp
//}

func noticePlayerChangedScore(clubID int32, uid int64, score int64) {
	gateWayID, _ := commonDB.LoadGatewayServiceID(db.PersonalRedisClient, uid)
	if len(gateWayID) > 0 {
		clubScoreText := commonDef.ScoreToClient(score)
		noticeClubScoreChanged := mateProto.MessageMaTe{To: gateWayID, SenderID: uid, MessageID: clubProto.ID_CurScoreChanged}
		noticeClubScoreChanged.Data, _ = json.Marshal(&clubProto.SC_CurScoreChanged{ClubID: clubID, Score: clubScoreText})
		wrapMQ.PublishProto(gateWayID, &noticeClubScoreChanged)
	}
}
