package clubEvent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"math"
	"strconv"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

// 得到小数点的数量
func checkDecimalPlaces(text string) int32 {
	isFindPoint := false
	pointCount := int32(0)
	for i, _ := range text {
		if text[i] == '.' {
			if isFindPoint == true {
				return -1
			}
			isFindPoint = true
			continue
		}

		if text[i] > '9' || text[i] < '0' {
			return -2
		}

		if isFindPoint == true {
			pointCount += 1
		}
	}
	return pointCount
}

// 检查俱乐部玩法参数
func checkClubPlayParam(clubRule *collClub.DBClubRule) (int32, string) {
	var (
		err        error
		pointCount int32
	)

	funcGetFloat64 := func(text string) (float64, error) {
		temp, err1 := strconv.ParseFloat(text, 64)
		if err1 != nil || math.IsNaN(temp) == true {
			return 0, err1
		}
		pointCount = checkDecimalPlaces(clubRule.MinEntryScoreText)
		if pointCount < 0 || pointCount > 3 {
			return 0, fmt.Errorf("entryScore error. %s", clubRule.MinEntryScoreText)
		}
		return temp, nil
	}

	clubRule.MinEntryScoreInt, err = funcGetFloat64(clubRule.MinEntryScoreText)
	if err != nil {
		return errorCodeDef.Err_Param, fmt.Sprintf("entryScore error. %s", clubRule.MinEntryScoreText)
	}
	if clubRule.GameOverCon < 0 || clubRule.GameOverCon > 3 {
		return errorCodeDef.Err_Param, fmt.Sprintf("gameOverCon error. %d", clubRule.GameOverCon)
	}

	if clubRule.GameOverCon == 2 {
		clubRule.OverScoreInt, err = funcGetFloat64(clubRule.OverScoreText)
		if err != nil {
			return errorCodeDef.Err_Param, fmt.Sprintf("overScore error. %s", clubRule.OverScoreText)
		}
	}

	if clubRule.PayWay < collClub.PaywayAllWin || clubRule.PayWay > collClub.PaywayAllPlayer {
		return errorCodeDef.Err_Param, fmt.Sprintf("PayWay error. %d", clubRule.PayWay)
	}

	if clubRule.BaoMingWay < collClub.BrokerageWayFixed || clubRule.BaoMingWay > collClub.BrokerageWayPercentage {
		return errorCodeDef.Err_Param, fmt.Sprintf("brokerage error. %d", clubRule.BaoMingWay)
	}
	//if clubRule.GongXianWay != 0 && clubRule.GongXianWay != 1 {
	//	return errorCodeDef.Err_Param, fmt.Sprintf("gxWay error. %d", clubRule.GongXianWay)
	//}

	lastMax_, _temp := float64(0), float64(0)
	curMin, curMax := float64(0), float64(0)
	baoDi_ := float64(0)
	for i, _ := range clubRule.LevelArr {
		curMin, err = funcGetFloat64(clubRule.LevelArr[i].SText)
		if err != nil {
			return errorCodeDef.Err_Param, fmt.Sprintf("level error")
		}
		curMax, err = funcGetFloat64(clubRule.LevelArr[i].TText)
		if err != nil {
			return errorCodeDef.Err_Param, fmt.Sprintf("level error")
		}
		baoDi_, err = funcGetFloat64(clubRule.LevelArr[i].BaoDiText)
		if err != nil {
			return errorCodeDef.Err_Param, fmt.Sprintf("level error")
		}
		if baoDi_ < 0 {
			return errorCodeDef.Err_Param, fmt.Sprintf("level error")
		}

		if curMin < 0 || curMin >= curMax || curMin < lastMax_ {
			return errorCodeDef.Err_Param, fmt.Sprintf("level error")
		}

		_temp, err = funcGetFloat64(clubRule.LevelArr[i].VText)
		if err != nil {
			return errorCodeDef.Err_Param, fmt.Sprintf("level error")
		}

		if clubRule.BaoMingWay == collClub.BrokerageWayPercentage {
			clubRule.LevelArr[i].Percent = _temp
			if clubRule.LevelArr[i].Percent < 0 || clubRule.LevelArr[i].Percent > 100 {
				return errorCodeDef.Err_Param, fmt.Sprintf("level error")
			}
			//if baoDi_ > curMin*_temp {
			//	return errorCodeDef.Err_Param, fmt.Sprintf("level error")
			//}
		} else {
			//if baoDi_ > curMin {
			//	return errorCodeDef.Err_Param, fmt.Sprintf("level error")
			//}
			//if _temp > curMin {
			//	return errorCodeDef.Err_Param, fmt.Sprintf("level error")
			//}
		}

		lastMax_ = curMax
	}

	return 0, ""
}

func onPutClubPlay(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_PutClubPlay{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if clubData.DirectSupervisor.ClubID != 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	clubRule := collClub.DBClubRule{}
	err = json.Unmarshal([]byte(param.ClubCfgText), &clubRule)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	param.ClubCfg = &clubRule

	gameIDMap_ := make(map[int32]struct{})
	gameIDMap_[param.GameID] = struct{}{}
	for _, v := range clubData.PlayIDMap {
		if v.IsDelete == false && v.IsHide == false {
			gameIDMap_[v.GameID] = struct{}{}
		}
	}
	if len(gameIDMap_) > 5 {
		rsp.Status = errorCodeDef.ErrGameIDMore
		return rsp
	}

	if len(param.ClubPlayName) > 256 {
		rsp.Status = errorCodeDef.Err_Param
		rsp.Detail = fmt.Sprintf("field name err. %d", len(param.ClubPlayName))
	}

	serviceID := findGameServiceID(param.GameID)
	if len(serviceID) < 1 {
		rsp.Status = errorCodeDef.Err_Param
		rsp.Detail = fmt.Sprintf("field play_id err. %d", param.GameID)
		return rsp
	}

	// 参数检查
	rsp.Status, rsp.Detail = checkClubPlayParam(param.ClubCfg)
	if rsp.Status != 0 {
		return rsp
	}

	// 转发给游戏服 验证 玩法规则
	data, _ := json.Marshal(msg)
	err = wrapMQ.ForwardTo(serviceID, &data)
	if err != nil {
		rsp.Status = errorCodeDef.Err_System
		glog.Warning("onNewClubPlay() err:=", err.Error())
		return rsp
	}
	return nil
}

// 游戏服的 应答
func onPRCPutClubPlay(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	msg.MessageID = clubProto.ID_PutClubPlay

	param := mateProto.JsonResponse{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if param.Status != 0 {
		rsp.Status = param.Status
		rsp.Detail = param.Detail
		return rsp
	}

	var decodeArr []byte
	decodeArr, err = base64.StdEncoding.DecodeString(param.Data.(string))
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onPRCPutClubPlay() err. err:=", err.Error(), ",text:=", param.Data.(string))
		return rsp
	}
	msgPutCluPlay := clubProto.CS_PutClubPlay{}
	err = json.Unmarshal(decodeArr, &msgPutCluPlay)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	clubRule := collClub.DBClubRule{}
	err = json.Unmarshal([]byte(msgPutCluPlay.ClubCfgText), &clubRule)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	msgPutCluPlay.ClubCfg = &clubRule

	rsp.Status, rsp.Detail = checkClubPlayParam(msgPutCluPlay.ClubCfg)
	if rsp.Status != 0 {
		return rsp
	}

	//consumables := gjson.Get(msgPutCluPlay.PlayCfg, "consumables").Int()

	dbClubPlay := collClub.DBClubPlay{
		ID:       msgPutCluPlay.ClubPlayID,
		Name:     msgPutCluPlay.ClubPlayName,
		IsDelete: false,
		IsHide:   msgPutCluPlay.IsHide,
		GameID:   msgPutCluPlay.GameID,
		PlayCfg:  msgPutCluPlay.PlayCfg,
		TableCfg: msgPutCluPlay.TableCfg,
		ClubCfg:  msgPutCluPlay.ClubCfgText,
		//Consumables: int32(consumables),
	}

	if msgPutCluPlay.ClubPlayID == 0 {
		var clubData *collClub.DBClubData
		clubData, err = loadClubData(msgPutCluPlay.ClubID)
		if err != nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}
		playsCount := 0
		for _, v := range clubData.PlayIDMap {
			if v.IsDelete == false {
				playsCount += 1
			}
		}
		if playsCount >= 60 {
			rsp.Status = errorCodeDef.ErrClubPlaysMore
			return rsp
		}

		rsp.Status, err = db.NewClubPlay(msgPutCluPlay.ClubID, &dbClubPlay)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rsp.Status = errorCodeDef.Err_Not_Find_Item
				return rsp
			}
			rsp.Status = errorCodeDef.Err_Failed
			glog.Warning("onPRCNewClubPlay() err := ", err.Error())
			return rsp
		}
		db.PutClubOperationLog(msgPutCluPlay.ClubID, 5,
			msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
			&collClub.DBPlayUpdate{PlayName: dbClubPlay.Name})
	} else {
		err = db.UpdateClubPlay(msgPutCluPlay.ClubID, msgPutCluPlay.ClubPlayID, &dbClubPlay)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rsp.Status, rsp.Detail = errorCodeDef.Err_Not_Find_Item, ""
				return rsp
			}
		}
		db.PutClubOperationLog(msgPutCluPlay.ClubID, 6,
			msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
			&collClub.DBPlayUpdate{PlayName: dbClubPlay.Name})
	}

	// 更新后 待重新获取
	delLocalClubData(msgPutCluPlay.ClubID)
	noticeDBServiceClubChanged(msgPutCluPlay.ClubID)

	rsp.Data = &dbClubPlay

	if msgPutCluPlay.IsHide == true {
		moniMsg := mateProto.MessageMaTe{MessageID: clubProto.ID_DeleteClubPlay}
		moni := clubProto.CS_DeleteClubPlay{
			ClubID:     msgPutCluPlay.ClubID,
			ClubPlayID: msgPutCluPlay.ClubPlayID,
			GameID:     msgPutCluPlay.GameID}

		moniMsg.MsgBody = &moni
		tableSnapshotEvent.PostMaTeEvent(&moniMsg)

		db.DeleteVirtualTableConfigItem(msgPutCluPlay.ClubID, msgPutCluPlay.ClubPlayID)
		virtualTableEvent.PostMaTeEvent(nil)
	}

	return rsp
}

// 删除俱乐部玩法
func onDeleteClubPlay(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_DeleteClubPlay{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	clubPlayData, ok := clubData.PlayIDMap[param.ClubPlayID]
	if ok == false {
		rsp.Status = errorCodeDef.ErrClubNotFindPlayID
		return rsp
	} else if clubPlayData.IsDelete == true {
		rsp.Status = errorCodeDef.ErrClubNotFindPlayID
		return rsp
	}

	err = db.DeleteClubPlay(param.ClubID, param.ClubPlayID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		glog.Warning("db.DeleteClubPlay().", err.Error(), clubData.ClubID, param.ClubPlayID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	err = db.DelClubPlayPercent(clubData.ClubID, param.ClubPlayID)
	if err != nil {
		glog.Warning("db.DelClubPlayPercent().", err.Error(), clubData.ClubID, param.ClubPlayID)
	}

	db.DeleteVirtualTableConfigItem(clubData.ClubID, param.ClubPlayID)
	virtualTableEvent.PostMaTeEvent(nil)

	db.DeleteRobotCfg(clubData.ClubID, param.ClubPlayID)
	noticeRobotDeleteClubPlay(clubData.ClubID, param.ClubPlayID)

	param.GameID = clubPlayData.GameID
	msg.MsgBody = &param
	tableSnapshotEvent.PostMaTeEvent(msg)

	rsp.Data = &param

	db.PutClubOperationLog(param.ClubID, 7,
		msg.SenderID, LoadPlayerNick_Name(msg.SenderID).Nick,
		&collClub.DBPlayUpdate{PlayName: clubPlayData.Name})

	// 删除后 待重新获取
	delLocalClubData(param.ClubID)

	return rsp
}

// 获取俱乐部玩法列表
func onGetClubPlay(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetClubPlayList{}
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

	if _, ok := clubData.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	if clubData.MengZhuClubID > 0 {
		clubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}
	}

	rspBody := clubProto.SC_GetClubPlayList{VersionNum: clubData.PlayVersionNum}
	rsp.Data = &rspBody

	if clubData.PlayVersionNum == param.VersionNum {
		rspBody.ClubPlayArr = nil
		rspBody.GameIDArr = nil
	} else {
		//clubPlay := make([]*collClub.DBClubPlay, 0, 10)
		//for _, v := range clubData.PlayIDMap {
		//	if v.IsDelete == false {
		//		clubPlay = append(clubPlay, v)
		//	}
		//}
		rspBody.ClubPlayArr = clubData.PlayArr
		rspBody.GameIDArr = clubData.GameIDArr
	}

	return rsp
}

func onGetClubPlayInfo(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetClubPlay{}
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

	if _, ok := clubData.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	if clubData.MengZhuClubID > 0 {
		clubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}
	}

	rspBody := clubProto.SC_GetClubPlay{
		VersionNum: clubData.PlayVersionNum,
		GameIDArr:  clubData.GameIDArr}
	rsp.Data = &rspBody

	// 获取精简版
	if param.ClubPlayID == 0 {
		rspBody.ClubPlayArr = clubData.SimplePlayIDArr
		return rsp
	}

	if v, ok := clubData.PlayIDMap[param.ClubPlayID]; ok == false {
		rspBody.ClubPlayArr = []*collClub.DBClubPlay{v}
	}

	return rsp
}

// 获取俱乐部 游戏id列表
func onGetClubGameIDList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetClubPlayList{}
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

	if _, ok := clubData.MemberMap[msg.SenderID]; ok == false {
		rsp.Status = errorCodeDef.ErrClubNotMember
		return rsp
	}

	if clubData.MengZhuClubID > 0 {
		clubData, err = loadClubData(clubData.MengZhuClubID)
		if err != nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}
	}

	rspBody := clubProto.SC_GetClubPlayList{VersionNum: clubData.PlayVersionNum}
	rsp.Data = &rspBody

	if clubData.PlayVersionNum == param.VersionNum {
		rspBody.ClubPlayArr = nil
		rspBody.GameIDArr = nil
	} else {
		//clubPlay := make([]*collClub.DBClubPlay, 0, 10)
		//for _, v := range clubData.PlayIDMap {
		//	if v.IsDelete == false {
		//		clubPlay = append(clubPlay, v)
		//	}
		//}
		//rspBody.ClubPlayArr = clubPlay
		rspBody.GameIDArr = clubData.GameIDArr
	}

	return rsp
}

func onGetClubPlayPercent(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetClubPlayPercent{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.OperClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	isFind := false
	for _, v := range clubData.DirectSubordinate {
		if v.ClubID == param.TargetClubID {
			isFind = true
			break
		}
	}
	if isFind == false && param.TargetClubID != clubData.ClubID {
		rsp.Status = errorCodeDef.ErrIsNotDirectSupervisor
		return rsp
	}

	mzClubID := clubData.ClubID
	if clubData.MengZhuClubID > 0 {
		mzClubID = clubData.MengZhuClubID
	}

	targetPercentArr := make([]collClub.DBClubPlayPercentage, 0, 10)
	operationPercentArr := make([]collClub.DBClubPlayPercentage, 0, 10)

	err = db.GetClubAllPlayPercent(mzClubID, param.TargetClubID, &targetPercentArr)
	if err != nil && err != mongo.ErrNoDocuments {
		glog.Warning("GetClubPlayPercent()", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	err = db.GetClubAllPlayPercent(mzClubID, param.OperClubID, &operationPercentArr)
	if err != nil && err != mongo.ErrNoDocuments {
		glog.Warning("GetClubPlayPercent()", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	rsp.Data = struct {
		BLWay         int                             `json:"blWay"`
		TargetClub    []collClub.DBClubPlayPercentage `json:"targetClub"`
		OperationClub []collClub.DBClubPlayPercentage `json:"operationClub"`
		TargetClubID  int32                           `json:"targetClubID"`
	}{clubData.BiLiShowWay, targetPercentArr, operationPercentArr, param.TargetClubID}
	return rsp
}

func onGetClubMemberRemark(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ID_GetClubMemberRemark{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(param.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	memberRemarkArr := make([]clubProto.ClubMemberRemark, len(clubData.MemberMap)+len(clubData.DirectSubordinate))

	for i, v := range clubData.MemberArr {
		memberRemarkArr[i].Name = v.Remark
		memberRemarkArr[i].UID = v.ID
	}

	index := len(clubData.MemberArr)
	for _, v := range clubData.DirectSubordinate {
		memberRemarkArr[index].Name = v.Remark
		memberRemarkArr[index].UID = v.PlayerID
		index++
	}

	rsp.Data = memberRemarkArr
	return rsp
}
