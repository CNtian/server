package clubEvent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"math"
	"strconv"
	"time"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func onClubOperation(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ClubOperation{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	rsp.Data = msg.Data

	switch param.Action {
	case clubProto.ClubOperator_FROZEN:
		rsp.Status, err = setClubFrozen(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_Open:
		rsp.Status, err = setClubOpen(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_BaoDi:
		rsp.Status, err = setClubBaoDi(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_Percentage:
		//rsp.Status, err = setClubPercentage(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_ManageFee:
		//rsp.Status, err = setClubManagementFee(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_DiscardCombine:
		//rsp.Status, err = discardCombine(msg.SenderID, &param, &param.ActionData)
		rsp.Status, err = onKickoutLeague(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetNotice:
		rsp.Status, err = setClubNotice(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetClubName:
		rsp.Status, err = setClubName(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetMemberExit:
		rsp.Status, err = setClubMemberFreeExit(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_kickOutMember:
		rsp.Status, err = setClubKickOutMember(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_kickOutLeague:
		rsp.Status, err = setClubKickOutLeague(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetClubPlayPercent:
		rsp.Status, err = setClubPlayPercent(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetClubBaoDiPercent:
		//rsp.Status, err = setClubBaoDiPercent(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_PutActivity:
		rsp.Status, err = setClubActivity(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_Stocktaking:
		rsp.Status, err = stocktaking(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_UpdateVirtualTableCfg:
		rsp.Status, err = onUpdateVirtualTableConfig(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetShowScoreWater:
		rsp.Status, err = setShowScoreWater(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetShowRankingList:
		rsp.Status, err = setShowRankList(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetMZNotice:
		rsp.Status, err = setMZNotice(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetShowBaoMingFee:
		rsp.Status, err = setShowBaoMingFee(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetBiLiShowWay:
		rsp.Status, err = setBiLiShowWay(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_SetMaxTZCount:
		rsp.Status, err = setMaxTZCount(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_HideClubPlay:
		rsp.Status, err = setHideClubPlay(msg.SenderID, &param, &param.ActionData)
	case clubProto.ClubOperator_PlayerGongXianWay:
		rsp.Status, err = setPlayerGongXianWay(msg.SenderID, &param, &param.ActionData)
	default:

	}
	if err != nil {
		glog.Warning("onClubOperation() action:",
			param.Action, ",data:=", string(param.ActionData), ",err:=", err.Error())
		if rsp.Status == 0 {
			rsp.Status = errorCodeDef.Err_Failed
		}
	}
	return rsp
}

func setClubFrozen(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationFrozen struct {
		ClubID   int32 `json:"clubID"` // 目标俱乐部ID
		IsFrozen bool  `json:"isFrozen"`
	}

	actionData := ClubOperationFrozen{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}
	if clubData.IsSeal == true {
		return errorCodeDef.ErrIsSeal, nil
	}

	if param.OperationClubID != actionData.ClubID {
		_, ok := clubData.SubordinatesMap[actionData.ClubID]
		if ok == false {
			return errorCodeDef.ErrClubRelation, nil
		}
	}

	rspCode, err = db.SetClubState(actionData.ClubID, 2, actionData.IsFrozen)
	if err == nil && rspCode == 0 {
		curStatus := int32(0)
		if actionData.IsFrozen == true {
			curStatus = 1
		}
		db.PutClubOperationLog(actionData.ClubID, 8,
			senderID, LoadPlayerNick_Name(senderID).Nick,
			&collClub.DBStatusUpdate{Type: 2, CurStatus: curStatus})

		delLocalClubData(actionData.ClubID)
	}

	return rspCode, err
}

func setClubOpen(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationOpen struct {
		ClubID int32 `json:"clubID"`
		IsOpen bool  `json:"isOpen"`
	}

	actionData := ClubOperationOpen{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.IsSeal == true {
		return errorCodeDef.ErrIsSeal, nil
	}

	if param.OperationClubID != actionData.ClubID {
		_, ok := clubData.SubordinatesMap[actionData.ClubID]
		if ok == false {
			return errorCodeDef.ErrClubRelation, nil
		}
	}

	rspCode, err = db.SetClubState(actionData.ClubID, 1, actionData.IsOpen)
	if err == nil && rspCode == 0 {
		curStatus := int32(0)
		if actionData.IsOpen == true {
			curStatus = 1
		}
		db.PutClubOperationLog(actionData.ClubID, 8,
			senderID, LoadPlayerNick_Name(senderID).Nick,
			&collClub.DBStatusUpdate{Type: 1, CurStatus: curStatus})

		delLocalClubData(actionData.ClubID)
	}

	return rspCode, err
}

func setClubBaoDi(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationBaoDi struct {
		ClubID int32  `json:"clubID"`
		BaoDi  string `json:"baoDi"`
	}

	actionData := ClubOperationBaoDi{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if param.OperationClubID == actionData.ClubID {
		return errorCodeDef.Err_Param, nil
	} else {
		isFind := false
		for _, v := range clubData.DirectSubordinate {
			if v.ClubID == actionData.ClubID {
				isFind = true
				break
			}
		}
		if isFind == false {
			return errorCodeDef.ErrIsNotDirectSupervisor, nil
		}
	}

	tempFloat64, err1 := strconv.ParseFloat(actionData.BaoDi, 64)
	if err1 != nil || math.IsNaN(tempFloat64) == true {
		return errorCodeDef.Err_Param, nil
	}
	rspCode = checkDecimalPlaces(actionData.BaoDi)
	if rspCode < 0 || rspCode > 2 {
		return errorCodeDef.Err_Param, nil
	}

	tempBaoDi := commonDef.ScoreToService(tempFloat64)

	rspCode, err = db.SetClubBaoDi(actionData.ClubID, tempBaoDi)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)
	}
	return rspCode, err
}

func setClubPercentage(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {

	type ClubOperationPercentage struct {
		ClubID      int32 `json:"clubID"`
		ShowPercent int32 `json:"percent"`
	}

	req := ClubOperationPercentage{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, parentClubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if param.OperationClubID == req.ClubID {
		return errorCodeDef.ErrClubRelation, nil
	}
	isDirectClub := false
	for _, v := range parentClubData.DirectSubordinate {
		if v.ClubID == req.ClubID {
			isDirectClub = true
			break
		}
	}
	if isDirectClub == false {
		return errorCodeDef.ErrIsNotDirectSupervisor, nil
	}

	var targetClubData *collClub.DBClubData
	// 是否超过上级
	targetClubData, err = loadClubData(req.ClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}

	temp_ := (req.ShowPercent * parentClubData.DirectSupervisor.RealPercentage) / 100
	if req.ShowPercent < 0 || temp_ > parentClubData.DirectSupervisor.RealPercentage {
		return errorCodeDef.Err_Param, nil
	}

	rspCode = updateClubLinePercent(targetClubData, req.ShowPercent, parentClubData.DirectSupervisor.RealPercentage)

	return rspCode, err
}

func updateClubLinePercent(updateClub *collClub.DBClubData, showPercent, parentRealPercent int32) int32 {

	err := db.SetClubPercent(updateClub.ClubID, showPercent, parentRealPercent)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", updateClub.ClubID, " ", showPercent, " ", parentRealPercent)
		return errorCodeDef.Err_Exception
	}

	updateClub.DirectSupervisor.ShowPercentage = showPercent
	updateClub.DirectSupervisor.RealPercentage = (showPercent * parentRealPercent) / 100
	noticeDBServiceClubChanged(updateClub.ClubID)

	for i, _ := range updateClub.DirectSubordinate {
		clubDataTemp_, err := loadClubData(updateClub.DirectSubordinate[i].ClubID)
		if err != nil {
			glog.Warning(err.Error(), ",data:=", clubDataTemp_.ClubID)
			continue
		}
		updateClubLinePercent(clubDataTemp_, clubDataTemp_.DirectSupervisor.ShowPercentage, parentRealPercent)
	}
	return 0
}

func setClubManagementFee(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationManageFee struct {
		ClubID int32  `json:"clubID"`
		Fee    string `json:"fee"`
	}

	actionData := ClubOperationManageFee{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if param.OperationClubID == actionData.ClubID {
		return errorCodeDef.ErrClubRelation, nil
	} else {
		isFind := false
		for _, v := range clubData.DirectSubordinate {
			if v.ClubID == actionData.ClubID {
				isFind = true
				break
			}
		}
		if isFind == false {
			return errorCodeDef.ErrIsNotDirectSupervisor, nil
		}
	}

	tempFloat64, err1 := strconv.ParseFloat(actionData.Fee, 64)
	if err1 != nil || math.IsNaN(tempFloat64) == true {
		return errorCodeDef.Err_Param, nil
	}
	rspCode = checkDecimalPlaces(actionData.Fee)
	if rspCode < 0 || rspCode > 2 {
		return errorCodeDef.Err_Param, nil
	}

	tempFee := commonDef.ScoreToService(tempFloat64)

	rspCode, err = db.SetClubManageFee(actionData.ClubID, tempFee)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)

		noticeDBServiceClubChanged(actionData.ClubID)
	}
	return rspCode, err
}

func setClubNotice(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationSetNotice struct {
		ClubID int32  `json:"clubID"`
		Notice string `json:"notice"`
	}

	actionData := ClubOperationSetNotice{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, _ := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if param.OperationClubID != actionData.ClubID {
		return errorCodeDef.ErrClubRelation, nil
	}

	if len(actionData.Notice) > 256 {
		return errorCodeDef.Err_Param, nil
	}

	rspCode, err = db.SetClubNotice(actionData.ClubID, actionData.Notice)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)
	}
	return rspCode, err
}

func setClubName(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationSetClubName struct {
		ClubID int32  `json:"clubID"`
		Name   string `json:"name"`
	}

	actionData := ClubOperationSetClubName{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if len(actionData.Name) < 1 || len(actionData.Name) > 64 {
		return errorCodeDef.Err_Param, nil
	}

	rspCode, clubData := IsClubCreator(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}
	if clubData.DirectSupervisor.ClubID > 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	subAllClubID := clubData.Subordinates
	subAllClubID = append(subAllClubID, clubData.ClubID)

	rspCode, err = db.SetClubName(subAllClubID, actionData.Name)
	if err == nil && rspCode == 0 {
		for _, v := range subAllClubID {
			tmp_, _ := loadClubData(v)
			if tmp_ != nil {
				tmp_.Name = actionData.Name
				tmp_.ClubVerNum += 1
			}
			noticeClubMGRClubChanged(v)
		}

		db.PutClubOperationLog(actionData.ClubID, 11,
			senderID, LoadPlayerNick_Name(senderID).Nick, nil)
	}
	return rspCode, err
}

func setClubMemberFreeExit(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationSetClubMemberExit struct {
		ClubID int32 `json:"clubID"`
		IsFree bool  `json:"isFree"`
	}

	actionData := ClubOperationSetClubMemberExit{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, _ := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if param.OperationClubID != actionData.ClubID {
		return errorCodeDef.ErrClubRelation, nil
	}

	rspCode, err = db.SetClubMemberFreeExit(actionData.ClubID, actionData.IsFree)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)
	}
	return rspCode, err
}

func setClubKickOutMember(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationClubKickOutMember struct {
		ClubID  int32 `json:"clubID"`
		IsAllow bool  `json:"isAllow"`
	}

	actionData := ClubOperationClubKickOutMember{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	rspCode, err = db.SetClubKickOutMember(actionData.ClubID, actionData.IsAllow)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)
	}
	return rspCode, err
}

func setClubKickOutLeague(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationClubKickOutLeague struct {
		ClubID  int32 `json:"clubID"`
		IsAllow bool  `json:"isAllow"`
	}

	actionData := ClubOperationClubKickOutLeague{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	rspCode, err = db.SetClubKickOutLeague(actionData.ClubID, actionData.IsAllow)
	if err == nil && rspCode == 0 {
		delLocalClubData(actionData.ClubID)
	}
	return rspCode, err
}

func applyKickOutLeague(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationDiscardCombine struct {
		ClubID int32 `json:"clubID"`
	}

	targetData := ClubOperationDiscardCombine{}
	err := json.Unmarshal(*data, &targetData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if param.OperationClubID == targetData.ClubID {
		return errorCodeDef.ErrClubRelation, nil
	}

	isFind := false
	for _, v := range clubData.DirectSubordinate {
		if targetData.ClubID == v.ClubID {
			isFind = true
			break
		}
	}
	if isFind == false {
		return errorCodeDef.ErrClubRelation, nil
	}

	rspCode, err = exitLeagueNow(senderID, param.OperationClubID, param.TargetClubID)
	if rspCode != 0 {
		return rspCode, nil
	}
	if err != nil {
		glog.Warning("onApplyExitLeague() err. err:=", err.Error())
		rspCode = errorCodeDef.Err_Failed
		return rspCode, err
	}

	return rspCode, err
}

func onCheckExitLeague(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_CheckExitLeague{}
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

	var mailData *collClub.DBClubMail
	rsp.Status, mailData, err = db.CheckExitLeague(param.ApplyID, param.Pass, msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick)
	if rsp.Status != 0 {
		return rsp
	}
	if err != nil {
		glog.Warning("onCheckExitLeague() err. err:=", err.Error(), ",data:=", param)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if mailData == nil {
		glog.Warning("onCheckExitLeague() mailData is empty. data:=", param)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	if param.Pass == false {
		if x, _ := db.CheckNewMail(clubData.ClubID); x > 0 {
			clubData.IsHadNewMail = true
		} else {
			clubData.IsHadNewMail = false
		}
		return rsp
	}

	var directSupervisorClubID, directSubordinateClubID int32
	if mailData.Category == collClub.MailKickOutLeague {
		directSupervisorClubID = mailData.Data.(*collClub.DBKickOutLeague).InitiatorClubID
		directSubordinateClubID = mailData.Data.(*collClub.DBKickOutLeague).TargetClubID
	} else if mailData.Category == collClub.MailKickOutLeague {
		directSupervisorClubID = mailData.Data.(*collClub.DBApplyExitLeague).TargetClubID
		directSubordinateClubID = mailData.Data.(*collClub.DBApplyExitLeague).InitiatorClubID
	}
	rsp.Status, err = discardCombine(directSupervisorClubID, directSubordinateClubID)
	if rsp.Status != 0 || err != nil {
		db.MailOperationFailed(param.ApplyID)
	}
	if rsp.Status != 0 {
		return rsp
	}
	if err != nil {
		glog.Warning("onCheckExitLeague() mailData is empty. data:=", param)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	var club *collClub.DBClubData
	club, err = loadClubData(directSubordinateClubID)
	db.PutClubOperationLog(directSupervisorClubID, 4, msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
		&collClub.DBMergeClub{ClubID: club.ClubID, ClubName: club.Name})

	club, err = loadClubData(directSupervisorClubID)
	db.PutClubOperationLog(directSubordinateClubID, 4, msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
		&collClub.DBMergeClub{ClubID: club.ClubID, ClubName: club.Name})

	delAllClubData(true)

	if x, _ := db.CheckNewMail(clubData.ClubID); x > 0 {
		clubData.IsHadNewMail = true
	} else {
		clubData.IsHadNewMail = false
	}

	return rsp
}

func checkExitLeague(directSubordinateClubID int32) (int32, error) {
	subordinateClub, err := loadClubData(directSubordinateClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}
	//if subordinateClub.IsOpen == true {
	//	return errorCodeDef.ErrNotClose, nil
	//}
	//if time.Now().Sub(subordinateClub.CloseTimestamp).Hours() < 2.0 {
	//	return errorCodeDef.ErrCloseTimeLess, nil
	//}

	var (
		clubScoreCount, unuseClubScoreCount int64
	)
	clubScoreCount, err = db.GetClubCountScore(subordinateClub.ClubID)
	if err != nil {
		glog.Warning("discardCombine() err.", err.Error(), ", value:=", subordinateClub.ClubID)
		return errorCodeDef.Err_Failed, nil
	}
	if clubScoreCount != 0 {
		return errorCodeDef.ErrClubTotalScoreNot0, nil
	}

	clubIDArr := subordinateClub.Subordinates
	clubIDArr = append(clubIDArr, subordinateClub.ClubID)
	unuseClubScoreCount, err = db.TotalClubUnusable(clubIDArr)
	if err != nil {
		glog.Warning("discardCombine() err.", err.Error(), ", value:=", subordinateClub.ClubID)
		return errorCodeDef.Err_Failed, nil
	}
	if unuseClubScoreCount != 0 {
		return errorCodeDef.ErrClubUnuseScoreNot0, nil
	}
	return 0, nil
}

// 移除合并(直属上级ID,直属下级ID)
func discardCombine(directSupervisorClubID, directSubordinateClubID int32) (int32, error) {

	if r, e := checkExitLeague(directSubordinateClubID); r != 0 || e != nil {
		return r, e
	}
	//var (
	//	clubScoreCount, unuseClubScoreCount int64
	//)
	//clubScoreCount, err = db.GetClubCountScore(subordinateClub.ClubID)
	//if err != nil {
	//	glog.Warning("discardCombine() err.", err.Error(), ", value:=", subordinateClub.ClubID)
	//	return errorCodeDef.Err_Failed, nil
	//}
	//if clubScoreCount != 0 {
	//	return errorCodeDef.ErrClubTotalScoreNot0, nil
	//}

	//clubIDArr := subordinateClub.Subordinates
	//clubIDArr = append(clubIDArr, subordinateClub.ClubID)
	//unuseClubScoreCount, err = db.TotalClubUnusable(clubIDArr)
	//if err != nil {
	//	glog.Warning("discardCombine() err.", err.Error(), ", value:=", subordinateClub.ClubID)
	//	return errorCodeDef.Err_Failed, nil
	//}
	//if unuseClubScoreCount != 0 {
	//	return errorCodeDef.ErrClubUnuseScoreNot0, nil
	//}

	var (
		tempClubData *collClub.DBClubData
		rspCode      int32
	)

	subordinateClub, err := loadClubData(directSubordinateClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}

	tempClubID, superiorClubIDArr := directSupervisorClubID, make([]int32, 0, 10)
	for tempClubID != 0 {
		tempClubData, err = loadClubData(tempClubID)
		if err != nil {
			glog.Warning("discardCombine() err.", err.Error(), ", value:=", tempClubID)
			return errorCodeDef.ErrClubNotExist, nil
		}
		superiorClubIDArr = append(superiorClubIDArr, tempClubID)
		tempClubID = tempClubData.DirectSupervisor.ClubID
	}

	rspCode, err = db.DiscardCombination(superiorClubIDArr, directSupervisorClubID, directSubordinateClubID, subordinateClub.Subordinates)

	glog.Warning("exit league of club. s:=", directSupervisorClubID, ",x:=", directSubordinateClubID)

	noticeDBServiceClubDeleteTotal(superiorClubIDArr, directSubordinateClubID, subordinateClub.Subordinates)

	return rspCode, err
}

// 直接退出联盟
func exitLeagueNow(operID int64, directSupervisorClubID, directSubordinateClubID int32) (int32, error) {

	var (
		rspCode int32
		err     error
	)
	rspCode, err = checkExitLeague(directSubordinateClubID)
	if rspCode != 0 {
		return rspCode, err
	}
	if err != nil {
		glog.Warning("exitLeagueNow() mailData is empty. data:=", directSupervisorClubID, ",", directSubordinateClubID)
		rspCode = errorCodeDef.Err_Failed
		return rspCode, err
	}

	rspCode, err = discardCombine(directSupervisorClubID, directSubordinateClubID)
	if rspCode != 0 {
		return rspCode, err
	}
	if err != nil {
		glog.Warning("exitLeagueNow() mailData is empty. data:=", directSupervisorClubID, ",", directSubordinateClubID)
		rspCode = errorCodeDef.Err_Failed
		return rspCode, err
	}

	var club *collClub.DBClubData
	club, err = loadClubData(directSubordinateClubID)
	db.PutClubOperationLog(directSupervisorClubID, 4, operID, LoadPlayerNick_Name(operID).Nick,
		&collClub.DBMergeClub{ClubID: club.ClubID, ClubName: club.Name})

	club, err = loadClubData(directSupervisorClubID)
	db.PutClubOperationLog(directSubordinateClubID, 4, operID, LoadPlayerNick_Name(operID).Nick,
		&collClub.DBMergeClub{ClubID: club.ClubID, ClubName: club.Name})

	delAllClubData(true)

	return rspCode, err
}

func onKickoutLeague(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubOperationDiscardCombine struct {
		ClubID int32 `json:"clubID"`
	}

	targetData := ClubOperationDiscardCombine{}
	err := json.Unmarshal(*data, &targetData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.ClubID == param.TargetClubID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}
	if _, ok := clubData.SubordinatesMap[targetData.ClubID]; ok == false {
		return errorCodeDef.ErrClubRelation, nil
	}

	if clubData.DirectSupervisor.ClubID > 0 && clubData.IsKickOutLeague == false {
		return errorCodeDef.ErrPowerNotEnough, nil
	}

	direClubID := param.OperationClubID
	// 不是直属上级 操作  ,  盟主 操作
	if clubData.MengZhuClubID < 1 {
		tempClub, err := loadClubData(targetData.ClubID)
		if err != nil {
			glog.Warning("discardCombine() err.", err.Error(), ", value:=", targetData.ClubID)
			return errorCodeDef.ErrClubNotExist, err
		}
		direClubID = tempClub.DirectSupervisor.ClubID
	}

	rspCode, err = checkExitLeague(targetData.ClubID)
	if err != nil {
		return rspCode, err
	}
	if rspCode != 0 {
		return rspCode, err
	}

	var tempClubData *collClub.DBClubData
	subordinateClub, err := loadClubData(targetData.ClubID)
	if err != nil {
		return errorCodeDef.ErrClubNotExist, nil
	}

	tempClubID, superiorClubIDArr := direClubID, make([]int32, 0, 10)
	for tempClubID != 0 {
		tempClubData, err = loadClubData(tempClubID)
		if err != nil {
			glog.Warning("discardCombine() err.", err.Error(), ", value:=", tempClubID)
			return errorCodeDef.ErrClubNotExist, err
		}
		superiorClubIDArr = append(superiorClubIDArr, tempClubID)
		tempClubID = tempClubData.DirectSupervisor.ClubID
	}

	rspCode, err = db.DiscardCombination_(superiorClubIDArr, direClubID, targetData.ClubID, subordinateClub.Subordinates)
	if err != nil {
		glog.Warning("discardCombine() err.", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else if rspCode == 0 {
		delAllClubData(true)

		db.KickOutDelPercent(targetData.ClubID)

		deleteAllMember(clubData)
	}

	return rspCode, nil
}

func updateLinePercent(updateClubMap map[int32]*collClub.DBClubPlayPercentage,
	mzClubID int32, clubPlayID int64, parentRealPercentage float64,
	setClubID int32, dirSubClubID []collClub.DBClubMerge) int32 {

	perfect := int32(0)

	realPercentage, showPercentage := float64(0), float64(0)
	v, ok := updateClubMap[setClubID]
	if ok == true {
		realPercentage = (v.ShowPercentage * parentRealPercentage) / 100
		showPercentage = v.ShowPercentage
	}

	err := db.SetClubPlayPercent(mzClubID, setClubID, clubPlayID, parentRealPercentage, showPercentage, false)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", mzClubID, " ", setClubID, " ", clubPlayID, " ", showPercentage)
		return errorCodeDef.Err_Exception
	}

	msg := mateProto.MessageMaTe{MessageID: mateProto.ID_UpdateClubPlayPercent}
	msgBody := mateProto.SS_UpdateClubPlayPercent{MZClubID: mzClubID, ClubPlayID: clubPlayID}
	msgBody.ClubID, msgBody.Percent = setClubID, realPercentage
	msg.Data, _ = json.Marshal(&msgBody)

	// 通知 数据
	err = wrapMQ.PublishProto("db", &msg)
	if err != nil {
		glog.Warning("wrapMQ.PublishProto()", err.Error(), ",data:=", msgBody)
	}

	// 处理 下一级
	for _, vClubID := range dirSubClubID {
		clubDataTemp_, err := loadClubData(vClubID.ClubID)
		if err != nil {
			glog.Warning(err.Error(), ",data:=", mzClubID, " ", setClubID, " ", clubPlayID, " ", showPercentage)
			perfect = errorCodeDef.Err_Exception
			continue
		}
		updateLinePercent(updateClubMap, mzClubID, clubPlayID, realPercentage, clubDataTemp_.ClubID, clubDataTemp_.DirectSubordinate)
	}
	return perfect
}

// 设置俱乐部玩法百分比
func setClubPlayPercent(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubPlayPercent struct {
		ClubID           int32   `json:"clubID"`
		PlayID           int64   `json:"playID"`  // 玩法ID
		ParamShowPercent int32   `json:"percent"` // 百分比
		ShowPercent      float64 `json:"-"`       // 百分比
	}

	req := SetClubPlayPercent{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if req.ParamShowPercent > 100 || req.ParamShowPercent < 0 {
		return errorCodeDef.Err_Param, nil
	}
	req.ShowPercent = float64(req.ParamShowPercent)

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	isDirectClub := false
	for _, v := range clubData.DirectSubordinate {
		if v.ClubID == req.ClubID {
			isDirectClub = true
			break
		}
	}
	if isDirectClub == false {
		return errorCodeDef.ErrIsNotDirectSupervisor, nil
	}

	var (
		mzClubData *collClub.DBClubData
	)
	if clubData.MengZhuClubID != 0 {
		mzClubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			glog.Warning(err.Error(), ",clubID:=", clubData.MengZhuClubID)
			return errorCodeDef.Err_Failed, err
		}
	} else {
		mzClubData = clubData
	}

	if _, ok := mzClubData.PlayIDMap[req.PlayID]; ok == false {
		return errorCodeDef.ErrNotFindClubPlay, err
	}

	parentRealPercent := float64(0)
	parentClubPlayPercent := collClub.DBClubPlayPercentage{}

	err = db.GetClubPlayPercent(mzClubData.ClubID, clubData.ClubID, req.PlayID, &parentClubPlayPercent)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", mzClubData.ClubID, req.PlayID)
		return errorCodeDef.Err_Failed, nil
	}
	parentRealPercent = parentClubPlayPercent.RealPercentage

	if clubData.BiLiShowWay == 1 { //点中点
		// 是否超过 自己
		if (req.ShowPercent*parentRealPercent)/100 > parentRealPercent {
			return errorCodeDef.ErrClubOperationInvalid, nil
		}
	} else {
		// 是否超过 自己
		if req.ShowPercent > parentRealPercent {
			return errorCodeDef.ErrClubOperationInvalid, nil
		}
		// 真点位 -> 点中点 显示点位
		req.ShowPercent = (req.ShowPercent * 100) / parentRealPercent
	}

	subClubData, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", mzClubData.ClubID, req.ClubID, req.PlayID, req.ShowPercent)
		return errorCodeDef.Err_Failed, nil
	}

	// 获取所有俱乐部 百分比
	updateClubMap := make(map[int32]*collClub.DBClubPlayPercentage)
	tempSubClubArr := subClubData.Subordinates
	tempSubClubArr = append(tempSubClubArr, req.ClubID)
	err = db.GetSubClubPlayPercent(mzClubData.ClubID, tempSubClubArr, req.PlayID, &updateClubMap)
	if err != nil {
		glog.Warning(err.Error(), ", data:=", mzClubData.ClubID, req.ClubID, req.PlayID, req.ShowPercent)
		return errorCodeDef.Err_Failed, nil
	}
	if _, ok := updateClubMap[req.ClubID]; ok == true {
		updateClubMap[req.ClubID].ShowPercentage = req.ShowPercent
	} else {
		updateClubMap[req.ClubID] = &collClub.DBClubPlayPercentage{ShowPercentage: req.ShowPercent}
	}

	rspCode =
		updateLinePercent(updateClubMap, mzClubData.ClubID, req.PlayID, parentRealPercent, req.ClubID, subClubData.DirectSubordinate)

	return rspCode, nil
}

func updateLineBaoDiPercent(updateClubMap map[int32]*collClub.DBClubPlayPercentage,
	mzClubID int32, clubPlayID int64, parentRealPercentage int32,
	setClubID int32, dirSubClubID []collClub.DBClubMerge) int32 {

	perfect := int32(0)

	realPercentage, showPercentage := int32(0), int32(0)
	v, ok := updateClubMap[setClubID]
	if ok == true {
		realPercentage = (v.ShowBaoDiPer * parentRealPercentage) / 100
		showPercentage = v.ShowBaoDiPer
	}

	err := db.SetClubPlayPercent(mzClubID, setClubID, clubPlayID, float64(parentRealPercentage), float64(showPercentage), true)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", mzClubID, " ", setClubID, " ", clubPlayID, " ", showPercentage)
		return errorCodeDef.Err_Exception
	}

	msg := mateProto.MessageMaTe{MessageID: mateProto.ID_UpdateClubBaoDiPercent}
	msgBody := mateProto.SS_UpdateClubBaoDiPercent{MZClubID: mzClubID, ClubPlayID: clubPlayID}
	msgBody.ClubID, msgBody.Percent = setClubID, realPercentage
	msg.Data, _ = json.Marshal(&msgBody)

	// 通知 数据
	err = wrapMQ.PublishProto("db", &msg)
	if err != nil {
		glog.Warning("wrapMQ.PublishProto()", err.Error(), ",data:=", msgBody)
	}

	// 处理 下一级
	for _, vClubID := range dirSubClubID {
		clubDataTemp_, err := loadClubData(vClubID.ClubID)
		if err != nil {
			glog.Warning(err.Error(), ",data:=", mzClubID, " ", setClubID, " ", clubPlayID, " ", showPercentage)
			perfect = errorCodeDef.Err_Exception
			continue
		}
		updateLineBaoDiPercent(updateClubMap, mzClubID, clubPlayID, realPercentage, clubDataTemp_.ClubID, clubDataTemp_.DirectSubordinate)
	}
	return perfect
}

// 设置俱乐部保底百分比
func setClubBaoDiPercent(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubPlayPercent struct {
		ClubID      int32 `json:"clubID"`
		PlayID      int64 `json:"playID"`  // 玩法ID
		ShowPercent int32 `json:"percent"` // 百分比
	}

	req := SetClubPlayPercent{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if req.ShowPercent > 100 || req.ShowPercent < 0 {
		return errorCodeDef.Err_Param, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	isDirectClub := false
	for _, v := range clubData.DirectSubordinate {
		if v.ClubID == req.ClubID {
			isDirectClub = true
			break
		}
	}
	if isDirectClub == false {
		return errorCodeDef.ErrIsNotDirectSupervisor, nil
	}

	var (
		mzClubData *collClub.DBClubData
	)
	if clubData.MengZhuClubID != 0 {
		mzClubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			glog.Warning(err.Error(), ",clubID:=", clubData.MengZhuClubID)
			return errorCodeDef.Err_Failed, err
		}
	} else {
		mzClubData = clubData
	}

	if _, ok := mzClubData.PlayIDMap[req.PlayID]; ok == false {
		return errorCodeDef.ErrNotFindClubPlay, err
	}

	parentRealPercent := int32(0)
	parentClubPlayPercent := collClub.DBClubPlayPercentage{}

	err = db.GetClubPlayPercent(mzClubData.ClubID, clubData.ClubID, req.PlayID, &parentClubPlayPercent)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", mzClubData.ClubID, req.PlayID)
		return errorCodeDef.Err_Failed, nil
	}
	parentRealPercent = parentClubPlayPercent.RealBaoDiPer

	// 是否超过 自己
	if (req.ShowPercent*parentRealPercent)/100 > parentRealPercent {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	subClubData, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",data:=", mzClubData.ClubID, req.ClubID, req.PlayID, req.ShowPercent)
		return errorCodeDef.Err_Failed, nil
	}

	// 获取所有俱乐部 百分比
	updateClubMap := make(map[int32]*collClub.DBClubPlayPercentage)
	tempSubClubArr := subClubData.Subordinates
	tempSubClubArr = append(tempSubClubArr, req.ClubID)
	err = db.GetSubClubPlayPercent(mzClubData.ClubID, tempSubClubArr, req.PlayID, &updateClubMap)
	if err != nil {
		glog.Warning(err.Error(), ", data:=", mzClubData.ClubID, req.ClubID, req.PlayID, req.ShowPercent)
		return errorCodeDef.Err_Failed, nil
	}
	if _, ok := updateClubMap[req.ClubID]; ok == true {
		updateClubMap[req.ClubID].ShowBaoDiPer = req.ShowPercent
	} else {
		updateClubMap[req.ClubID] = &collClub.DBClubPlayPercentage{ShowBaoDiPer: req.ShowPercent}
	}
	rspCode =
		updateLineBaoDiPercent(updateClubMap, mzClubData.ClubID, req.PlayID, parentRealPercent, req.ClubID, subClubData.DirectSubordinate)

	return rspCode, nil
}

func setShowScoreWater(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowName struct {
		ClubID int32 `json:"clubID"`
		IsShow bool  `json:"isShow"`
	}

	req := SetClubShowName{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	sub := clubData.Subordinates
	sub = append(sub, clubData.ClubID)
	rspCode, err = db.SetShowScoreWater(sub, req.IsShow)
	if err != nil {
		glog.Warningln("setShowName()  ", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else {
		for _, v := range sub {
			tempClub, _ := loadClubData(v)
			if tempClub == nil {
				glog.Warningln("not find club data. id:=", v)
				continue
			}
			tempClub.IsShowScoreWater = req.IsShow
			tempClub.ClubVerNum += 1
		}
	}
	return rspCode, err
}

func setShowRankList(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		ClubID int32 `json:"clubID"`
		IsShow bool  `json:"isShow"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	sub := clubData.Subordinates
	sub = append(sub, clubData.ClubID)
	rspCode, err = db.UpdateClubShowRankList(sub, req.IsShow)
	if err != nil {
		glog.Warningln("setShowName()  ", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else {
		for _, v := range sub {
			tempClub, _ := loadClubData(v)
			if tempClub == nil {
				glog.Warningln("not find club data. id:=", v)
				continue
			}
			tempClub.IsShowRankList = req.IsShow
			tempClub.ClubVerNum += 1
		}
	}
	return rspCode, err
}

func setShowBaoMingFee(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		ClubID int32 `json:"clubID"`
		IsShow bool  `json:"isShow"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	sub := clubData.Subordinates
	sub = append(sub, clubData.ClubID)
	rspCode, err = db.UpdateClubShowBaoMingFee(sub, req.IsShow)
	if err != nil {
		glog.Warningln("setShowName()  ", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else {
		for _, v := range sub {
			tempClub, _ := loadClubData(v)
			if tempClub == nil {
				glog.Warningln("not find club data. id:=", v)
				continue
			}
			tempClub.IsShowBaoMingFee = req.IsShow
		}
	}
	return rspCode, err
}

func setBiLiShowWay(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		ClubID int32 `json:"clubID"`
		Value  int   `json:"value"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if req.Value != 0 && req.Value != 1 {
		return errorCodeDef.Err_Param, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}
	if req.Value == clubData.BiLiShowWay {
		return 0, nil
	}

	sub := clubData.Subordinates
	sub = append(sub, clubData.ClubID)
	rspCode, err = db.UpdateClubBiLiWay(sub, req.Value)
	if err != nil {
		glog.Warningln("setShowName()  ", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else {
		for _, v := range sub {
			tempClub, _ := loadClubData(v)
			if tempClub == nil {
				glog.Warningln("not find club data. id:=", v)
				continue
			}
			tempClub.BiLiShowWay = req.Value
			tempClub.ClubVerNum += 1
		}
	}
	return rspCode, err
}

func setMaxTZCount(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		ClubID int32 `json:"clubID"`
		Value  int32 `json:"value"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	switch req.Value {
	case 0, 10, 20, 30, 50:
	default:
		return errorCodeDef.Err_Param, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	sub := clubData.Subordinates
	sub = append(sub, clubData.ClubID)
	rspCode, err = db.UpdateClubMaxTongZhuoCount(sub, req.Value)
	if err != nil {
		glog.Warningln("setShowName()  ", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else {
		for _, v := range sub {
			tempClub, _ := loadClubData(v)
			if tempClub == nil {
				glog.Warningln("not find club data. id:=", v)
				continue
			}
			tempClub.MaxTZCount = req.Value
		}
	}
	return rspCode, err
}

func setHideClubPlay(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		ClubID     int32 `json:"clubID"`
		ClubPlayID int64 `json:"clubPlay"`
		Value      int   `json:"value"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	clubPlayV, ok := clubData.PlayIDMap[req.ClubPlayID]
	if ok == false {
		return errorCodeDef.ErrClubNotFindPlayID, nil
	}

	msg := mateProto.MessageMaTe{MessageID: clubProto.ID_PutClubPlay_RPC}

	proClubPlay := clubProto.CS_PutClubPlay{ClubID: clubData.ClubID,
		ClubPlayID: req.ClubPlayID, ClubPlayName: clubPlayV.Name, GameID: clubPlayV.GameID,
		PlayCfg: clubPlayV.PlayCfg, TableCfg: clubPlayV.TableCfg, ClubCfgText: clubPlayV.ClubCfg}

	proClubPlay.IsHide = req.Value == 1

	data_, _ := json.Marshal(&proClubPlay)

	jsonResp := mateProto.JsonResponse{}
	jsonResp.Data = base64.StdEncoding.EncodeToString(data_)

	msg.Data, _ = json.Marshal(&jsonResp)

	rsp := onPRCPutClubPlay(&msg)

	return rsp.Status, err
}

func setPlayerGongXianWay(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		Value int `json:"value"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	if req.Value != 1 && req.Value != 0 {
		return errorCodeDef.Err_Param, nil
	}

	allClubArr := clubData.Subordinates
	allClubArr = append(allClubArr, clubData.ClubID)
	rspCode, err = db.UpdateClubPlayerGongXianWay(allClubArr, req.Value)
	if err != nil {
		glog.Warning("SetGongXianWay() .", err.Error(), ",", clubData.ClubID)
		return errorCodeDef.Err_Failed, nil
	}
	for _, v := range allClubArr {
		tmp_, _ := loadClubData(v)
		if tmp_ != nil {
			tmp_.GongXianWay = int32(req.Value)
			tmp_.ClubVerNum += 1
		}
	}

	noticeDBServiceClubChanged(clubData.ClubID)

	return 0, err
}

func setMZNotice(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type SetClubShowRankList struct {
		ClubID int32  `json:"clubID"`
		Notice string `json:"notice"`
	}

	req := SetClubShowRankList{}
	err := json.Unmarshal(*data, &req)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	if len(req.Notice) > 256 {
		return errorCodeDef.Err_Param, nil
	}

	rspCode, clubData := IsClubCreator(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		return errorCodeDef.ErrFindSuperiorExist, nil
	}

	subAllClubID := clubData.Subordinates
	subAllClubID = append(subAllClubID, clubData.ClubID)

	rspCode, err = db.SetMZNotice(subAllClubID, req.Notice)
	if err != nil {
		glog.Warningln("setMZNotice()  ", err.Error())
		rspCode = errorCodeDef.Err_Failed
	} else {
		for _, v := range subAllClubID {
			tmp_, _ := loadClubData(v)
			if tmp_ != nil {
				tmp_.MZNotice = req.Notice
				tmp_.ClubVerNum += 1
			}
		}
		clubData.MZNotice = req.Notice
	}
	return rspCode, err
}

func onSetClubScore0(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_SetClubScore0{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	rspCode, clubData := IsClubCreator(param.OperationClubID, msg.SenderID)
	if rspCode != 0 {
		rsp.Status = rspCode
		return rsp
	}

	updateFunc := func(v *ClubMemberUpdateScore) {
		s, e := db.GetPlayerClubScore(v.BClubID, v.BUID)
		if e != nil {
			glog.Warning("db.GetPlayerClubScore()", v.BClubID, v.BUID, e.Error())
			return
		}
		if s == 0 {
			return
		}

		v.Value = commonDef.ScoreToClient(s * -1)

		jsonData, _ := json.Marshal(v)
		_, e = updateMemberScore(msg.SenderID, &jsonData)
		if e != nil {
			glog.Warning("updateMemberScore()", msg.SenderID, *v, e.Error())
			return
		}
	}

	if param.Opt == 1 {
		changed := ClubMemberUpdateScore{
			OperClubID: clubData.ClubID,
			BClubID:    clubData.ClubID}
		for _, v := range clubData.MemberMap {
			if v.ID == msg.SenderID {
				continue
			}
			changed.BUID = v.ID

			updateFunc(&changed)
		}
	} else if param.Opt == 2 {
		for _, v := range clubData.DirectSubordinate {

			changed := ClubMemberUpdateScore{
				OperClubID: clubData.ClubID,
				BClubID:    v.ClubID,
				BUID:       v.PlayerID}

			updateFunc(&changed)
		}
	}

	return rsp
}

// 总代  封印
func onUpdateClubStatus(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_UpdateClubStatus{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	rsp.Status, err = db.SetClubSeal(param.ClubID, param.IsOpen)

	delLocalClubData(param.ClubID)

	return rsp
}

// 设置活动
func setClubActivity(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	type ClubActivity struct {
		//ActivityID primitive.ObjectID `json:"aID"`
		MatchS   int64  `json:"match_s"`
		Continue int64  `json:"continue"`
		Notice   string `json:"notice"`

		RoundAward []uint `json:"rAward"`
		ScoreAward []uint `json:"sAward"`
	}

	actionData := ClubActivity{}
	err := json.Unmarshal(*data, &actionData)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}
	if clubData.MengZhuClubID > 0 {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	now := time.Now()

	// 活动期间不能修改
	if clubData.Activity != nil {
		_t := now.Unix()
		if _t >= clubData.Activity.Rule.MatchS.Unix() &&
			_t <= clubData.Activity.Rule.MatchE.Unix() {
			return errorCodeDef.ErrClubOperationInvalid, nil
		}
	}

	// 活动最少提前半小时 准备
	const minAdvanceSeconds = 60 * 30 //60 * 30
	if actionData.MatchS-now.Unix() < minAdvanceSeconds {
		return errorCodeDef.Err_Param, nil
	}

	// 测试
	{
		//actionData.Continue = 60 * 5
		//actionData.MatchS = time.Now().Unix() + 60
	}

	_matchS := time.Unix(actionData.MatchS, 0)
	_matchE := time.Unix(actionData.MatchS+actionData.Continue, 0)
	yearS, monthS, dayS := _matchS.Date()
	yearE, monthE, dayE := _matchE.Date()

	if _matchE.Sub(now) > time.Hour*24*7 {
		return errorCodeDef.Err_Param, nil
	}

	// 必须是同一天
	if yearS != yearE ||
		monthS != monthE ||
		dayS != dayE {
		return errorCodeDef.Err_Param, nil
	}

	// 活动最低持续时间 小于1小时
	const minActivityContinueSeconds = 60*59 + 59 // 60*59+59
	if _matchE.Unix()-actionData.MatchS < minActivityContinueSeconds ||
		actionData.Continue < minActivityContinueSeconds {
		return errorCodeDef.Err_Param, nil
	}

	if len(actionData.RoundAward) != 3 ||
		len(actionData.ScoreAward) != 3 {
		return errorCodeDef.Err_Param, nil
	}

	//awardV_ := actionData.RoundAward[0] + actionData.RoundAward[1] + actionData.RoundAward[2]
	//if awardV_ < 3 {
	//	return errorCodeDef.Err_Param, nil
	//}
	//
	//awardV_ = actionData.ScoreAward[0] + actionData.ScoreAward[1] + actionData.ScoreAward[2]
	//if awardV_ < 3 {
	//	return errorCodeDef.Err_Param, nil
	//}

	if len(actionData.Notice) > 1024 {
		return errorCodeDef.Err_Param, nil
	}

	rAward, sAward := make([]string, 0, 3), make([]string, 0, 3)
	for _, v := range actionData.RoundAward {
		rAward = append(rAward, fmt.Sprintf("%d", v))
	}
	for _, v := range actionData.ScoreAward {
		sAward = append(sAward, fmt.Sprintf("%d", v))
	}

	ac := collClub.DBClubActivity{ClubID: clubData.ClubID}
	ac.Rule = &collClub.Match{
		ActivityID: primitive.NewObjectID(),
		MatchS:     time.Unix(actionData.MatchS, 0),
		MatchE:     time.Unix(actionData.MatchS, 0).Add(time.Second * time.Duration(actionData.Continue)),
		RoundAward: rAward,
		ScoreAward: sAward,
		MatchNote:  actionData.Notice,
	}
	err = db.PutMengZhuActivity(&ac)
	if err != nil {
		return errorCodeDef.Err_Failed, nil
	} else {
		if clubData.Activity != nil {
			_activityWaitOpen.Delete(clubData.Activity.Rule.ActivityID)

			// 上次活动规则 保存至redis
			saveLastAcData, _ := json.Marshal(clubData.Activity)
			err = db.WriteLastClubActivityRule(clubData.ClubID, string(saveLastAcData))
			if err != nil {
				glog.Warning("WriteLastClubActivityRule  ", string(saveLastAcData), err.Error())
			}
		}

		clubData.LastActivity = clubData.Activity
		clubData.Activity = &ac
		_activityWaitOpen.Store(ac.Rule.ActivityID, &ac)
	}

	return rspCode, err
}
