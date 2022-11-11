package clubMGR

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/worker/clubEvent"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

var (
	_clubMap = make(map[int32]*collClub.DBClubData)

	mzClubIDMap = make(map[string]struct{}) // key:盟主的俱乐部ID

	_dailyMengZhuMap = make(map[int32]*dbCollectionDefine.DBDailyMengZHuPlayer)
)

func onNewProxyClub(msg *mateProto.MessageMaTe) (*mateProto.JsonResponse, int64, bool) {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_NewProxy{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp, 0, false
	}

	// 是否是总代
	if err = db.CheckTopProxy(msg.SenderID); err != nil {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp, 0, false
	}

	var playerClubInfo *collPlayer.PlayerInfo
	playerClubInfo, err = db.GetPlayerClubInfo(param.UID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
		} else {
			glog.Warning("onNewClub() err:=", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
		}
		return rsp, 0, false
	}

	if len(playerClubInfo.ClubData) > 2 {
		rsp.Status = errorCodeDef.ErrClubJoinMore
		return rsp, 0, false
	}
	for _, v := range playerClubInfo.ClubData {
		cv, err := _loadClubData(v.ClubID)
		if err != nil {
			glog.Warning("loadClubData() ", v.ClubID, ",", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return rsp, 0, false
		}
		// 该玩家是否有自己圈子已经存在总代
		if cv.CreatorID == param.UID {
			if cv.ProxyUp == 0 {
				err = db.AddProxy(msg.SenderID, cv.ClubID)
				if err != nil {
					glog.Warning("AddProxy() err:=", err.Error(), ",", msg.SenderID, ",", cv.ClubID)
					rsp.Status = errorCodeDef.Err_Failed
					return rsp, 0, false
				}
				_delLocalClubData(cv.ClubID)
				return rsp, 0, false
			} else if cv.ProxyUp > 0 {
				rsp.Status = errorCodeDef.ErrExistTopProxy
				return rsp, 0, false
			}
		}
	}

	var dbClubData *collClub.DBClubData
	dbClubData, err = db.NewClub(param.UID, msg.SenderID, "圈名未定义", nil)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onNewClub() uid:=", msg.SenderID, ",err := ", err.Error())
		return rsp, 0, false
	}

	rsp.Data = dbClubData
	return rsp, param.UID, true
}

func onInviteToProxy(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_InviteToProxy{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	// 是否是总代
	if err = db.CheckTopProxy(msg.SenderID); err != nil {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	var playerClubInfo *collPlayer.PlayerInfo
	playerClubInfo, err = db.GetPlayerClubInfo(param.UID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
		} else {
			glog.Warning("GetPlayerClubInfo() err:=", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
		}
		return rsp
	}

	if len(playerClubInfo.ClubData) > 2 {
		rsp.Status = errorCodeDef.ErrClubJoinMore
		return rsp
	}
	for _, v := range playerClubInfo.ClubData {
		cv, err := _loadClubData(v.ClubID)
		if err != nil {
			glog.Warning("loadClubData() ", v.ClubID, ",", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		// 该玩家是否有自己圈子已经存在总代
		if cv.CreatorID == param.UID {
			if cv.ProxyUp > 0 {
				rsp.Status = errorCodeDef.ErrExistTopProxy
				return rsp
			}
		}
	}

	var insertR primitive.ObjectID
	rsp.Status, insertR, err = db.InviteToMengZhu(msg.SenderID, param.UID)
	if err != nil {
		glog.Warning("InviteToMengZhu() err:=", err.Error(), msg.SenderID, " ", param.UID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	// 是否是 自动同意
	if rsp.Status == 0 {
		if ok, _ := db.IsAutoAgreeEmail(param.UID); ok == true {
			postMsg := mateProto.MessageMaTe{MessageID: clubProto.ID_HandleInviteToProxy,
				SenderID: param.UID}
			postMsgBody := clubProto.CS_HandleInviteToProxy{EmailID: insertR, Action: 1}
			postMsg.Data, _ = json.Marshal(&postMsgBody)
			postMsgData, _ := json.Marshal(&postMsg)

			HandleMQData(postMsgData)
		}
		clubEvent.NoticePlayerNewMail(param.UID)
	}

	return rsp
}

func onHandleInviteToProxy(msg *mateProto.MessageMaTe) (*mateProto.JsonResponse, bool) {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_HandleInviteToProxy{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp, false
	}
	if param.EmailID.IsZero() {
		rsp.Status = errorCodeDef.Err_Param
		return rsp, false
	}

	email := collPlayer.DBPlayerEmail{}
	err = db.GetMyEmail(param.EmailID, &email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp, false
		}
		glog.Warning(err.Error(), "  uid :=", msg.SenderID)
		return rsp, false
	}
	if email.Category != collPlayer.EmailInviteToMengZhu {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp, false
	}

	if param.Action == collPlayer.Operation_Delete {
		db.UpdateMyEmailStatus(email.ID, true)
		return rsp, false
	} else if param.Action == collPlayer.Operation_DisAgree {
		db.UpdateMyEmailStatus(email.ID, false)
		return rsp, false
	}

	if email.Status == collPlayer.MailStatusRead {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp, false
	}
	emailContent := email.Data.(*collPlayer.ItemEmailInviteToMengZhu)
	// 是否是总代
	if err = db.CheckTopProxy(emailContent.Uid); err != nil {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp, false
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
		return rsp, false
	}

	if len(playerClubInfo.ClubData) > 2 {
		rsp.Status = errorCodeDef.ErrClubJoinMore
		return rsp, false
	}

	defer func() {
		if rsp.Status == 0 {
			db.UpdateMyEmailStatus(email.ID, false)
		}
	}()

	var nullMeng *collClub.DBClubData
	for _, v := range playerClubInfo.ClubData {
		cv, err := _loadClubData(v.ClubID)
		if err != nil {
			glog.Warning("loadClubData() ", v.ClubID, ",", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return rsp, false
		}
		// 是否是自己创建的,  是否 不是子圈
		if cv.CreatorID != msg.SenderID || cv.MengZhuClubID > 0 {
			continue
		}
		// 是否存在代理
		if cv.ProxyUp > 0 {
			rsp.Status = errorCodeDef.ErrExistTopProxy
			return rsp, false
		}
		nullMeng = cv
	}

	// 利用之前的
	if nullMeng != nil {
		err = db.AddProxy(emailContent.Uid, nullMeng.ClubID)
		if err != nil {
			glog.Warning("AddProxy() err:=", err.Error(), ",", msg.SenderID, ",", nullMeng.ClubID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp, false
		}
		_delLocalClubData(nullMeng.ClubID)
		return rsp, false
	}

	var dbClubData *collClub.DBClubData
	dbClubData, err = db.NewClub(msg.SenderID, emailContent.Uid, "圈名未定义", nil)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("onNewClub() uid:=", msg.SenderID, ",err := ", err.Error())
		return rsp, false
	}

	rsp.Data = dbClubData
	return rsp, true
}

func GetPlayerClubInfo(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rspBody := clubProto.SC_GetPlayerClubInfo{
		ClubInfo: make([]*clubProto.PlayerClubInfo, 0, 3)}

	rsp := &mateProto.JsonResponse{Data: &rspBody}

	playerInfo, err := db.GetPlayerClubInfo(msg.SenderID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			rsp.Status = errorCodeDef.ErrClubOperationInvalid
			return rsp
		}
		glog.Warning("getPlayerClubInfo() err. err:=", err.Error())
		return rsp
	}

	for _, v := range playerInfo.ClubData {
		clubData, _ := _loadClubData(v.ClubID)
		if clubData == nil {
			continue
		}
		_p := clubEvent.LoadPlayerNick_Name(clubData.CreatorID)
		rspBody.ClubInfo = append(rspBody.ClubInfo, &clubProto.PlayerClubInfo{
			ClubID:          clubData.ClubID,
			ClubCreatorName: _p.Nick,
			URL:             _p.HeadURL,
			ClubName:        clubData.Name,
			Score:           commonDef.ScoreToClient(v.Score),
			LastTime:        v.LastPlayTime.Unix(),
		})
	}

	return rsp
}

func GetClubDetail(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rspBody := clubProto.CS_GetClubData{}

	rsp := &mateProto.JsonResponse{}
	err := json.Unmarshal(msg.Data, &rspBody)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if len(rspBody.ClubIDArr) < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	if len(mzClubIDMap) > 10000 {
		mzClubIDMap = make(map[string]struct{})
	}

	mzID, err := db.GetClubMengZhuID(rspBody.ClubIDArr[0])
	if err != nil {
		if err != redis.Nil {
			glog.Warning("GetClubMengZhuID :=", rspBody.ClubIDArr[0], ",err:=", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}

		clubData, err := _loadClubData(rspBody.ClubIDArr[0])
		if err != nil {
			glog.Warning("_loadClubData :=", rspBody.ClubIDArr[0], ",err:=", err.Error())
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		err = db.WriteClubMengZhuID(rspBody.ClubIDArr[0], clubData.MengZhuClubID)
		if err != nil {
			glog.Warning("WriteClubMengZhuID :=", rspBody.ClubIDArr[0], ",err:=", err.Error())
		}
		if clubData.MengZhuClubID == 0 {
			mzID = fmt.Sprintf("%d", rspBody.ClubIDArr[0])
		}
	}
	if mzID == "0" {
		mzID = fmt.Sprintf("%d", rspBody.ClubIDArr[0])
	}
	mzClubIDMap[mzID] = struct{}{}

	err = wrapMQ.PublishProto(mzID, msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}
	return nil
}

func JoinClub(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_ApplyJoinClub{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var club *collClub.DBClubData
	club, err = _loadClubData(param.ClubID)
	if err != nil {
		glog.Warning(param.ClubID, err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	mzID := ""
	if club.MengZhuClubID > 0 {
		mzID = fmt.Sprintf("%d", club.MengZhuClubID)
	} else {
		mzID = fmt.Sprintf("%d", club.ClubID)
	}
	err = wrapMQ.PublishProto(mzID, msg)
	if err != nil {
		glog.Warning(param.ClubID, err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	return nil
}

func MengZhuLogin(msg *mateProto.MessageMaTe) {
	mzClubIDMap[msg.MZID] = struct{}{}
}

func onGetMyAllMengZhu(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	list, err := db.GetProxy(msg.SenderID)
	if err != nil {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	arr := make([]clubProto.SC_GetProxyClubID, 0, len(list))
	for _, v := range list {
		cv, _ := _loadClubData(v)
		if cv == nil {
			continue
		}
		arr = append(arr, clubProto.SC_GetProxyClubID{ClubID: v, ClubName: cv.Name})
	}
	rsp.Data = arr
	return rsp
}

func onUpdateClubLevel(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_UpdateClubLevel{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	mzID := ""
	if param.ClubID < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	v, err := _loadClubData(param.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if v.ProxyUp != msg.SenderID {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	mzID = fmt.Sprintf("%d", param.ClubID)
	err = wrapMQ.PublishProto(mzID, msg)
	if err != nil {
		glog.Warning(param.ClubID, err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	return nil
}

func onUpdateClubStatus(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_UpdateClubStatus{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	mzID := ""
	if param.ClubID < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	v, err := _loadClubData(param.ClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if v.ProxyUp != msg.SenderID {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	mzID = fmt.Sprintf("%d", param.ClubID)
	err = wrapMQ.PublishProto(mzID, msg)
	if err != nil {
		glog.Warning(param.ClubID, err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	return nil
}

func onGetProxyList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetProxy{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if param.CurPage < 0 || param.CurPage > 100 ||
		param.PageSize < 1 || param.PageSize > 30 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	clubArr, err := db.GetProxy(msg.SenderID)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	rspBody := clubProto.SC_GetProxy{MZCount: len(clubArr)}
	rsp.Data = &rspBody
	if param.PageSize*param.CurPage >= len(clubArr) {
		return rsp
	}

	year, month, day := time.Now().Date()
	stringDate := fmt.Sprintf("%d%02d%02d", year, month, day)
	param.Date, _ = strconv.Atoi(stringDate)

	endIndex := param.CurPage * param.PageSize
	dataArr := make([]*dbCollectionDefine.DBDailyMengZHuPlayer, 0, param.PageSize)
	for ; endIndex < len(clubArr); endIndex++ {
		dv, ok := _dailyMengZhuMap[clubArr[endIndex]]
		if ok == false {
			dataArr = append(dataArr, &dbCollectionDefine.DBDailyMengZHuPlayer{ClubID: clubArr[endIndex]})
			continue
		}
		dataArr = append(dataArr, dv)
	}

	sealMap := make(map[int32]bool)
	db.GetClubSealStatus(clubArr[param.CurPage*param.PageSize:endIndex], &sealMap)

	for _, v := range dataArr {
		cv, _ := _loadClubData(v.ClubID)
		if cv == nil {
			continue
		}

		v.ClubInfo_ = &dbCollectionDefine.ClubInfo{CreatorID: cv.CreatorID, ClubName: cv.Name}

		if sv, ok := sealMap[v.ClubID]; ok == true {
			v.ClubInfo_.Seal = sv
		}
		h1, h2, n, _ := db.LoadPlayerHead(cv.CreatorID)
		v.ClubInfo_.CreatorHead += h1 + h2
		v.ClubInfo_.CreatorName = n
		v.ClubInfo_.CurRoomCard, _, _ = db.GetPlayerProperty(cv.CreatorID)
	}

	rspBody.Data = dataArr
	return rsp
}

func onGetProxyReportList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GetProxyReportList{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	clubArr, err := db.GetProxy(msg.SenderID)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	if param.ClubID == 0 {
		r := make([]dbCollectionDefine.DBDailyMengZHuPlayer, 0, 7)
		err = db.GetCurrentMengZhuDaily(clubArr, &r)
		if err != nil {
			glog.Warning(err.Error(), ",", msg.SenderID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		rsp.Data = &struct {
			ClubID int32                                     `json:"clubID"`
			Arr    []dbCollectionDefine.DBDailyMengZHuPlayer `json:"arr"`
		}{ClubID: param.ClubID, Arr: r}
		return rsp
	} else if param.ClubID == -1 {
		if len(clubArr) > 0 {
			param.ClubID = clubArr[0]
		}
	} else {
		ok := false
		for _, v := range clubArr {
			if v == param.ClubID {
				ok = true
				break
			}
		}
		if ok == false {
			rsp.Status = errorCodeDef.Err_Param
			return rsp
		}
	}

	r := make([]dbCollectionDefine.DBDailyMengZHuPlayer, 0, 7)
	err = db.GetCurrentMengZhuDaily([]int32{param.ClubID}, &r)
	if err != nil {
		glog.Warning(err.Error(), ",", msg.SenderID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	rsp.Data = &struct {
		ClubID int32                                     `json:"clubID"`
		Arr    []dbCollectionDefine.DBDailyMengZHuPlayer `json:"arr"`
	}{ClubID: param.ClubID, Arr: r}
	return rsp
}

func onGiveRoomCard(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_GiveRoomCard{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if param.Value < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	clubArr, err := db.GetProxy(msg.SenderID)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	ok := false
	for _, v := range clubArr {
		if v == param.ToClubID {
			ok = true
			break
		}
	}
	if ok == false {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	cv, err := _loadClubData(param.ToClubID)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	err = db.GiveRoomCard(msg.SenderID, cv.CreatorID, param.Value)
	if err != nil {
		glog.Warning(err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	return rsp
}

func onGiveRoomCardList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	v, err := db.GiveRoomCardList(msg.SenderID)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			glog.Warning(err.Error(), ", uid:=", msg.SenderID)
			rsp.Status = errorCodeDef.Err_Failed
		}
		return rsp
	}

	rsp.Data = v
	return rsp
}

func onCancelProxy(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	param := clubProto.CS_CancelProxy{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	if len(param.ClubID) > 10 || len(param.ClubID) < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	err = db.CancelProxy(msg.SenderID, param.ClubID)
	if err != nil {
		glog.Warning(err.Error(), " uid:=", msg.SenderID, ",clubID:=", param.ClubID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	} else {
		for _, v := range param.ClubID {
			_delLocalClubData(v)
		}
	}
	return rsp
}

func onNoticeDailyChanged(msg *mateProto.MessageMaTe) {
	param := mateProto.SS_NoticDailyChanged{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		glog.Warning(err.Error())
		return
	}
	if param.MZClubID == 0 {
		_dailyMengZhuMap = make(map[int32]*dbCollectionDefine.DBDailyMengZHuPlayer)
		return
	}

	v, ok := _dailyMengZhuMap[param.MZClubID]
	if ok == false {
		v = &dbCollectionDefine.DBDailyMengZHuPlayer{}
		v.ClubID = param.MZClubID
		v.Players, _ = db.GetMengZhuAllPlayers(param.MZClubID)
		_dailyMengZhuMap[param.MZClubID] = v
	}

	v.Consumables = param.Consumables
	v.GameRoundCount = param.GameRoundCount
	v.GameToConsumablesArr = param.GameCategoryArr
	v.DailyPlayers = param.DailyPlayers
}

func onUpdateClubData(msg *mateProto.MessageMaTe) {
	param := mateProto.SS_UpdateClubData{}
	err := json.Unmarshal(msg.Data, &param)
	if err != nil {
		glog.Warning("onUpdateClubData() err:=", err.Error(), ",data:=", string(msg.Data))
		return
	}
	if param.ClubID == 0 {
		_clubMap = make(map[int32]*collClub.DBClubData)
		return
	}

	_delLocalClubData(param.ClubID)
}
