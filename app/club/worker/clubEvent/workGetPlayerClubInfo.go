package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	commonDB "vvService/commonPackge/db"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func onExitClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ExitClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var (
		clubData   *collClub.DBClubData
		clubScore  int64
		isMZMember bool
	)
	clubData, err = loadClubData(param.ClubID)
	if err != nil {
		glog.Warning("loadClubData() err. err:=", err.Error(), ",clubID:=", param.ClubID)
		rsp.Status = errorCodeDef.ErrClubNotExist
		return rsp
	}

	if _, ok := clubData.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}
	if clubData.DirectSupervisor.ClubID < 1 {
		isMZMember = true
	}

	// 是否在游戏中
	playerGameInfo, err := commonDB.GetPlayerGameInfo(db.PersonalRedisClient, msg.SenderID)
	if err != nil {
		glog.Warning("commonDB.GetPlayerGameInfo() err. uid:=", msg.SenderID, " ,err:=%s", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if playerGameInfo != nil && playerGameInfo.Table != 0 {
		rsp.Status = errorCodeDef.Err_In_Table_Yet
		return rsp
	}

	clubScore, err = db.GetPlayerClubScore(param.ClubID, msg.SenderID)
	if err != nil {
		glog.Warning("GetPlayerClubScore() err. err:=", err.Error(), param.ClubID, ",", msg.SenderID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if clubScore < -10 || clubScore > 10 {
		rsp.Status = errorCodeDef.ErrClubScoreNE0
		return rsp
	}

	if msg.SenderID == commonDef.SpecialUID {
		msg.SenderID = clubData.CreatorID
	}

	// 圈主离开
	if clubData.CreatorID == msg.SenderID {
		if clubData.DirectSupervisor.ClubID > 0 {
			rsp.Status = errorCodeDef.ErrFindSuperiorExist
			return rsp
		}

		if len(clubData.Subordinates) > 0 {
			rsp.Status = errorCodeDef.ErrSubordinateExist
			return rsp
		}

		memberArr := make([]int64, len(clubData.MemberMap))
		for i, v := range clubData.MemberArr {
			memberArr[i] = v.ID
		}

		err = db.DissolveClub(param.ClubID, memberArr, clubData.ProxyUp)
		if err != nil {
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning("onExitClub() err. err:=", err.Error(), ",clubID:=", param.ClubID, ",uid:=", clubData.CreatorID)
			return rsp
		}

		// 删除后 待重新获取
		delLocalClubData(param.ClubID)
		deleteAllMember(clubData)
		return rsp
	}

	// 成员离开
	if clubData.IsFreeExit == false {
		rsp.Status, err = db.ApplyExitClub(clubData.ClubID, msg.SenderID)
		if err != nil {
			glog.Warning(err.Error(), ",", clubData.ClubID, msg.SenderID)
		} else {
			rsp.Status = errorCodeDef.ErrWaitCheck
			noticeClubNewMail(clubData.ClubID)
		}
		return rsp
	}

	rsp.Status, err = memberExitClub(isMZMember, clubData.ClubID, msg.SenderID)
	if err != nil {
		glog.Warning(err.Error(), ",", clubData.ClubID, msg.SenderID)
	} else {
		// 黑名单
		delete(_blackListMap, msg.SenderID)
		// 互斥组
		for _, v := range clubData.PlayerMutex {
			for _, vP := range v.Player {
				if vP == msg.SenderID {
					delMutexPlayer(clubData, v.ID)
				}
			}
		}
		deleteAllMember(clubData)
	}

	return rsp
}

func memberExitClub(isMZMember bool, clubID int32, playerID int64) (int32, error) {

	playerGameIntro, err := commonDB.GetPlayerGameInfo(db.PersonalRedisClient, playerID)
	if err != nil {
		return errorCodeDef.Err_Failed, err
	}

	if playerGameIntro != nil && playerGameIntro.Table != 0 {
		tableSource, _, _, _, err := commonDB.GetTableInfo(db.GameRedisClient, playerGameIntro.Table)
		if err != nil {
			return errorCodeDef.Err_Failed, err
		}
		if len(tableSource) > 0 {
			return errorCodeDef.Err_In_Table_Yet, nil
		}
	}

	clubScore, err := db.GetPlayerClubScore(clubID, playerID)
	if err != nil {
		return errorCodeDef.ErrClubNotMember, err
	}
	if clubScore < 0 || clubScore >= commonDef.SR {
		return errorCodeDef.ErrClubPaiWeiScoreNot0, nil
	}

	rspCode := int32(0)
	rspCode, err = db.ClubMemberExit(clubID, playerID, false)
	if err != nil {
		return rspCode, err
	}

	// 转发给机器人模块
	if isMZMember == true && clubID > 0 {
		f_ := mateProto.MessageMaTe{MessageID: mateProto.ID_ChangeRole}
		f_.Data, _ = json.Marshal(&mateProto.SS_ChangedRole{OperationClubID: clubID, PlayerID: playerID, Action: false})
		f_data, _ := json.Marshal(&f_)
		wrapMQ.ForwardTo("robot", &f_data)
	}

	// 删除后 待重新获取
	delLocalClubData(clubID)

	playerNH := LoadPlayerNick_Name(playerID)
	db.PutClubOperationLog(clubID, 2,
		playerID, playerNH.Nick,
		&collClub.DBPlayerJoinExitClub{PlayerID: playerID, PlayerNick: playerNH.Nick})
	return 0, nil
}
