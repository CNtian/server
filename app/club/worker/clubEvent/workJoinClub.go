package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func checkClubPower(clubID int32, uid int64) (int32, *collClub.DBClubData) {

	clubData, err := loadClubData(clubID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errorCodeDef.Err_Not_Find_Item, nil
		}
		glog.Warning("onCheckApplyJoinClub() err. clubID:=", clubID, ",uid:=", uid, ",err:=", err.Error())
		return errorCodeDef.Err_Failed, nil
	}

	if uid == commonDef.SpecialUID {
		return 0, clubData
	}

	memberData, ok := clubData.MemberMap[uid]
	if ok == false {
		return errorCodeDef.ErrClubNotMember, nil
	}
	if clubData.CreatorID != uid && memberData.IsAdmin == false {
		return errorCodeDef.ErrPowerNotEnough, nil
	}
	if clubData.IsStocking == true {
		return errorCodeDef.ErrStocking, nil
	}

	return 0, clubData
}

func IsClubCreator(clubID int32, uid int64) (int32, *collClub.DBClubData) {

	clubData, err := loadClubData(clubID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errorCodeDef.Err_Not_Find_Item, nil
		}
		glog.Warning("onCheckApplyJoinClub() err. clubID:=", clubID, ",uid:=", uid, ",err:=", err.Error())
		return errorCodeDef.Err_Failed, nil
	}

	if clubData.IsStocking == true {
		return errorCodeDef.ErrStocking, nil
	}

	if uid == commonDef.SpecialUID {
		return 0, clubData
	}

	if clubData.CreatorID == uid {
		return 0, clubData
	}

	return errorCodeDef.ErrPowerNotEnough, clubData
}

func onApplyJoinClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ApplyJoinClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var playerClubData *collPlayer.PlayerInfo
	playerClubData, err = db.GetPlayerClubInfo(msg.SenderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		glog.Warning("onApplyJoinClub() uid:= ", msg.SenderID, " ,err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	if len(playerClubData.ClubData) > 2 {
		rsp.Status = errorCodeDef.ErrClubJoinMore
		return rsp
	}
	for _, v := range playerClubData.ClubData {
		if v.ClubID == param.ClubID {
			rsp.Status = errorCodeDef.ErrClubAlreadyInMember
			return rsp
		}
	}

	clubData, err := loadClubData(param.ClubID)
	if err != nil {
		glog.Warning("onApplyJoinClub() err.", err.Error(), ",status:=", rsp.Status)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if len(clubData.MemberArr) > 500 {
		rsp.Status = errorCodeDef.ErrMemberLimit
		return rsp
	}

	rsp.Status, err = isClubMemberOverlap(msg.SenderID, param.ClubID)
	if err != nil {
		glog.Warning("onApplyJoinClub() err.", err.Error(), ",status:=", rsp.Status)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	if rsp.Status != 0 {
		return rsp
	}

	senderInfo := LoadPlayerNick_Name(msg.SenderID)
	rsp.Status, err = db.ApplyJoinClub(param.ClubID, msg.SenderID, senderInfo.HeadURL, senderInfo.Nick)
	if rsp.Status != 0 {
		return rsp
	}
	if err != nil {
		glog.Warning("onApplyJoinClub() uid:=", msg.SenderID, ",err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	noticeClubNewMail(param.ClubID)
	return rsp
}

func onGetClubMail(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetClubMail{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	rsp.Status, _ = checkClubPower(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	rsp.Data, err = db.GetClubMail(param.ClubID, param.Status)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onGetApplyJoinList() err.", param, ",err:=", err.Error())
		return rsp
	}

	return rsp
}

func onCheckApplyJoinClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_CheckApplyJoinClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var (
		clubData  *collClub.DBClubData
		statusBak int32
	)
	rsp.Status, clubData = checkClubPower(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	if len(clubData.MemberArr) > 500 {
		rsp.Status = errorCodeDef.ErrMemberLimit
		return rsp
	}

	rsp.Status, err = isClubMemberOverlap(param.ApplyUID, param.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onCheckApplyJoinClub() err.", param, ",err:=", err.Error())
		return rsp
	}
	if rsp.Status != 0 {
		// 错误发生就认为 拒绝
		param.Pass = false
		statusBak = rsp.Status
	}

	rsp.Status, err = db.CheckApplyJoin(param.ApplyID, param.ClubID, param.ApplyUID, param.Pass,
		msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick)
	if rsp.Status != 0 || err != nil {
		db.MailOperationFailed(param.ApplyID)
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onCheckApplyJoinClub() err.", param, ",err:=", err.Error())
		return rsp
	}

	if rsp.Status == 0 && param.Pass == true {
		// 更新后 待重新获取
		delLocalClubData(param.ClubID)
		noticeDBServiceClubChanged(param.ClubID)

		db.PutClubOperationLog(param.ClubID, 1,
			msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
			&collClub.DBPlayerJoinExitClub{PlayerID: param.ApplyUID,
				PlayerNick: LoadPlayerNick_Name(param.ApplyUID).Nick})

		NoticePlayerJoinClub(param.ClubID, param.ApplyUID)
		deleteAllMember(clubData)
	}

	if x, _ := db.CheckNewMail(clubData.ClubID); x > 0 {
		clubData.IsHadNewMail = true
	} else {
		clubData.IsHadNewMail = false
	}

	if statusBak != 0 {
		rsp.Status = statusBak
	}

	return rsp
}

func onCheckApplyExitClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_CheckExitClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var (
		clubData   *collClub.DBClubData
		statusBak  int32
		isMZMember bool
	)
	rsp.Status, clubData = checkClubPower(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if param.Pass == true && clubData.IsKickOutMember == false {
		rsp.Status = errorCodeDef.ErrTiRenPower
		return rsp
	}

	if clubData.DirectSupervisor.ClubID > 0 {
		isMZMember = true
	}

	rsp.Status, err = db.CheckApplyExitClub(param.ApplyID, param.ClubID, param.ApplyerID, param.Pass,
		msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick)
	if rsp.Status != 0 || err != nil {
		db.MailOperationFailed(param.ApplyID)
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onCheckApplyExitClub() err.", param, ",err:=", err.Error())
		return rsp
	}

	if rsp.Status == 0 && param.Pass == true {
		rsp.Status, err = memberExitClub(isMZMember, clubData.ClubID, param.ApplyerID)
		if err != nil {
			glog.Warning(err.Error(), ",", clubData.ClubID, param.ApplyerID)
		} else {
			NoticePlayerExitClub(clubData.ClubID, param.ApplyerID)
			deleteAllMember(clubData)
		}
	}

	if x, _ := db.CheckNewMail(clubData.ClubID); x > 0 {
		clubData.IsHadNewMail = true
	} else {
		clubData.IsHadNewMail = false
	}

	if statusBak != 0 {
		rsp.Status = statusBak
	}

	return rsp
}

func onDragIntoClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msgDragIntoClub := clubProto.CS_DragIntoClub{}
	err := json.Unmarshal(msg.Data, &msgDragIntoClub)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(msgDragIntoClub.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	if len(clubData.MemberArr) > 500 {
		rsp.Status = errorCodeDef.ErrMemberLimit
		return rsp
	}

	rsp.Status, err = isClubMemberOverlap(msgDragIntoClub.PlayerID, msgDragIntoClub.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onDragIntoClub() err. err:=", err.Error(), ",param:=", msgDragIntoClub)
		return rsp
	}
	if rsp.Status != 0 {
		return rsp
	}

	rsp.Status, err = db.DragIntoClub(msgDragIntoClub.ClubID, msgDragIntoClub.PlayerID)
	if err != nil {
		glog.Warning("onDragIntoClub() err:=", err.Error())
		return rsp
	}

	// 更新后 待重新获取
	delLocalClubData(msgDragIntoClub.ClubID)
	noticeDBServiceClubChanged(msgDragIntoClub.ClubID)
	deleteAllMember(clubData)

	db.PutClubOperationLog(msgDragIntoClub.ClubID, 1,
		msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
		&collClub.DBPlayerJoinExitClub{PlayerID: msgDragIntoClub.PlayerID,
			PlayerNick: LoadPlayerNick_Name(msgDragIntoClub.PlayerID).Nick})

	NoticePlayerJoinClub(msgDragIntoClub.ClubID, msgDragIntoClub.PlayerID)

	return rsp
}

// 邀请加入 俱乐部
func onInviteJoinClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msgDragIntoClub := clubProto.CS_DragIntoClub{}
	err := json.Unmarshal(msg.Data, &msgDragIntoClub)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(msgDragIntoClub.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	if len(clubData.MemberArr) > 500 {
		rsp.Status = errorCodeDef.ErrMemberLimit
		return rsp
	}

	rsp.Status, err = isClubMemberOverlap(msgDragIntoClub.PlayerID, msgDragIntoClub.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("InviteJoinClub() err. err:=", err.Error(), ",param:=", msgDragIntoClub)
		return rsp
	}
	if rsp.Status != 0 {
		return rsp
	}

	mzClubID := clubData.ClubID
	if clubData.MengZhuClubID > 0 {
		mzClubID = clubData.MengZhuClubID
	}

	var insertR primitive.ObjectID
	rsp.Status, insertR, err = db.InviteJoinClub(mzClubID, clubData.ClubID, clubData.Name, msg.SenderID, msgDragIntoClub.PlayerID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("InviteJoinClub() err. err:=", err.Error(), ",param:=", msgDragIntoClub)
		return rsp
	}

	// 是否是 自动同意
	if rsp.Status == 0 {
		if ok, _ := db.IsAutoAgreeEmail(msgDragIntoClub.PlayerID); ok == true {
			postMsg := mateProto.MessageMaTe{MessageID: clubProto.ID_HandleInviteJoinClub,
				SenderID: msgDragIntoClub.PlayerID}
			postMsgBody := clubProto.CS_HandleInviteToProxy{EmailID: insertR, Action: 1}
			postMsg.Data, _ = json.Marshal(&postMsgBody)
			SelfPostEvent.PostMaTeEvent(&postMsg)
		}

		NoticePlayerNewMail(msgDragIntoClub.PlayerID)

		deleteAllMember(clubData)
	}

	return rsp
}

func onHandleInviteJoinClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_HandleInviteToProxy{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	if param.EmailID.IsZero() {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	email := collPlayer.DBPlayerEmail{}
	err = db.GetMyEmail(param.EmailID, &email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		glog.Warning(err.Error(), "  uid :=", msg.SenderID)
		return rsp
	}

	if email.Category != collPlayer.EmailInviteJoinClub {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	if param.Action == collPlayer.Operation_Delete {
		db.UpdateMyEmailStatus(email.ID, true)
		return rsp
	} else if param.Action == collPlayer.Operation_DisAgree {
		db.UpdateMyEmailStatus(email.ID, false)
		return rsp
	}

	if email.Status == collPlayer.MailStatusRead {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	emailContent := email.Data.(*collPlayer.ItemEmailInviteToClub)

	defer func() {
		if rsp.Status == 0 {
			db.UpdateMyEmailStatus(email.ID, false)
		}
	}()

	rsp.Status, err = isClubMemberOverlap(msg.SenderID, emailContent.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onDragIntoClub() err. err:=", err.Error(), ",param:=", msg.SenderID, ",", emailContent.ClubID)
		return rsp
	}
	if rsp.Status != 0 {
		return rsp
	}

	rsp.Status, err = db.DragIntoClub(emailContent.ClubID, msg.SenderID)
	if err != nil {
		glog.Warning("onDragIntoClub() err:=", err.Error())
		return rsp
	}

	// 更新后 待重新获取
	delLocalClubData(emailContent.ClubID)
	noticeDBServiceClubChanged(emailContent.ClubID)

	clubData_, _ := loadClubData(emailContent.ClubID)
	if clubData_ != nil {
		deleteAllMember(clubData_)
	}

	db.PutClubOperationLog(emailContent.ClubID, 1,
		msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
		&collClub.DBPlayerJoinExitClub{PlayerID: msg.SenderID,
			PlayerNick: LoadPlayerNick_Name(msg.SenderID).Nick})

	NoticePlayerJoinClub(emailContent.ClubID, msg.SenderID)
	return rsp
}

func isClubMemberOverlap(uid int64, clubID int32) (int32, error) {
	clubData, err := loadClubData(clubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, err
	}

	tempClubID := clubData.ClubID
	for i := 0; i < 100 && tempClubID != 0; i++ {
		var tempClubData *collClub.DBClubData
		tempClubData, err = loadClubData(tempClubID)
		if err != nil {
			return errorCodeDef.ErrClubNotExist, err
		}

		if tempClubData.DirectSupervisor.ClubID < 1 {
			if _, ok := tempClubData.MemberMap[uid]; ok == true {
				return errorCodeDef.ErrInMergeClub, nil
			}
			break
		}

		tempClubID = tempClubData.DirectSupervisor.ClubID
	}

	clubData, err = loadClubData(tempClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, err
	}

	for _, v := range clubData.Subordinates {
		var tempClubData *collClub.DBClubData
		tempClubData, err = loadClubData(v)
		if err != nil {
			return errorCodeDef.ErrClubNotExist, err
		}
		if _, ok := tempClubData.MemberMap[uid]; ok == true {
			return errorCodeDef.ErrInMergeClub, nil
		}
	}

	return 0, nil
}
