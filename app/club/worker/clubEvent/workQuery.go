package clubEvent

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"strconv"
	"time"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	"vvService/dbCollectionDefine"
	collClub "vvService/dbCollectionDefine/club"
	collPlayer "vvService/dbCollectionDefine/playerInfo"
)

func NewGetClubMemberData(clubData *collClub.DBClubData, player *collClub.DBClubMember, date int,
	playerClubScore map[int64]collPlayer.PlayerInfo, total map[int64]dbCollectionDefine.DBClubPlayerTotal) *clubProto.GetClubMemberData {
	playerNH := LoadPlayerNick_Name(player.ID)
	item := clubProto.GetClubMemberData{
		PlayerID:   player.ID,
		PlayerHead: playerNH.HeadURL,
		PlayerNick: playerNH.Nick,
	}

	if v, ok := playerClubScore[player.ID]; ok == true {
		for _, v := range v.ClubData {
			if v.ClubID == clubData.ClubID {
				item.Score = commonDef.ScoreToClient(v.Score)
				break
			}
		}
	}

	item.ClubID = clubData.ClubID
	item.ClubName = clubData.Name
	item.ClubCreatorID = clubData.CreatorID
	item.ClubCreatorName = LoadPlayerNick_Name(clubData.CreatorID).Nick
	if v, _ := clubData.MemberMap[player.ID]; v != nil {
		item.JoinTime = v.JoinClubTime.Unix()
	}

	if player.ID == clubData.CreatorID {
		item.Role = 2
	} else if player.IsAdmin == true {
		item.Role = 1
	}
	if player.Frozen.IsFrozen == true {
		item.Status = 1
	}
	item.IsStop3 = player.IsNo3
	item.IsStop4 = player.IsNo4
	item.Remark = player.Remark

	if v, ok := total[player.ID]; ok == true {
		item.GameScoreText = commonDef.ScoreToClient(v.GameScore)
		item.GameCount = v.GameCount
		item.HaoKaText = commonDef.ScoreToClient(v.HaoKa)
		item.XiaoHaoScore = commonDef.ScoreToClient(v.XiaoHaoScore)
	}

	return &item
}

func onGetClubMember(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubMember{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.PlayerClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	now__ := time.Now()
	if req.Date == 0 {
		year, month, day := now__.Date()
		req.Date, _ = strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	}

	rspBody := clubProto.SC_GetClubMember{CurDate: now__.Unix()}
	rsp.Data = &rspBody

	if req.QueryMemberUID != 0 {
		if memberData, ok := clubData.MemberMap[req.QueryMemberUID]; ok == true {
			queryPlayerIDArr := []int64{req.QueryMemberUID}

			total := make(map[int64]dbCollectionDefine.DBClubPlayerTotal)
			err = db.GetPlayerTotal(clubData.ClubID, queryPlayerIDArr, req.Date, &total)
			if err != nil {
				glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
				return rsp
			}
			clubScoreMap := make(map[int64]collPlayer.PlayerInfo)
			err = db.GetSomePlayerClubScore(queryPlayerIDArr, &clubScoreMap)
			if err != nil {
				glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
				return rsp
			}

			var item *clubProto.GetClubMemberData
			item = NewGetClubMemberData(clubData, memberData, req.Date,
				clubScoreMap, total)

			rspBody.MemberArr = []*clubProto.GetClubMemberData{item}
			return rsp
		}

		var playerClubInfo *collPlayer.PlayerInfo
		playerClubInfo, err = db.GetPlayerClubInfo(req.QueryMemberUID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rsp.Status = errorCodeDef.Err_Not_Find_Item
			} else {
				rsp.Status = errorCodeDef.Err_Failed
			}
			return rsp
		}

		for _, v := range playerClubInfo.ClubData {
			if _, ok := clubData.SubordinatesMap[v.ClubID]; ok == false {
				continue
			}
			directSubordinateClubData, err := loadClubData(v.ClubID)
			if err != nil {
				rsp.Status = errorCodeDef.Err_Failed
				glog.Error("onGetClubMember() err:=", err.Error())
				return rsp
			}

			memberData, _ := directSubordinateClubData.MemberMap[req.QueryMemberUID]
			if memberData == nil {
				rsp.Status = errorCodeDef.Err_Not_Find_Item
				return rsp
			}

			queryPlayerIDArr := []int64{req.QueryMemberUID}

			total := make(map[int64]dbCollectionDefine.DBClubPlayerTotal)
			err = db.GetPlayerTotal(directSubordinateClubData.ClubID, queryPlayerIDArr, req.Date, &total)
			if err != nil {
				glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
				return rsp
			}
			clubScoreMap := make(map[int64]collPlayer.PlayerInfo)
			err = db.GetSomePlayerClubScore(queryPlayerIDArr, &clubScoreMap)
			if err != nil {
				glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
				return rsp
			}

			var item *clubProto.GetClubMemberData
			item = NewGetClubMemberData(directSubordinateClubData, memberData, req.Date,
				clubScoreMap, total)
			item.Remark = ""
			rspBody.MemberArr = []*clubProto.GetClubMemberData{item}
			return rsp
		}

		rsp.Status = errorCodeDef.Err_Not_Find_Item
		return rsp
	} else if req.QueryClubID == clubData.ClubID {

		rspBody.MemberArr = make([]*clubProto.GetClubMemberData, 0, len(clubData.MemberArr)+len(clubData.DirectSubordinate))
		rspBody.MemberArr = make([]*clubProto.GetClubMemberData, 0, 100)

		queryPlayerIDArr := make([]int64, len(clubData.MemberArr))
		for i, v := range clubData.MemberArr {
			queryPlayerIDArr[i] = v.ID
		}

		total := make(map[int64]dbCollectionDefine.DBClubPlayerTotal)
		err = db.GetPlayerTotal(clubData.ClubID, queryPlayerIDArr, req.Date, &total)
		if err != nil {
			glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
			return rsp
		}
		clubScoreMap := make(map[int64]collPlayer.PlayerInfo)
		err = db.GetSomePlayerClubScore(queryPlayerIDArr, &clubScoreMap)
		if err != nil {
			glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
			return rsp
		}

		for i := 0; i < len(clubData.MemberArr); i++ {
			var item *clubProto.GetClubMemberData
			item = NewGetClubMemberData(clubData, clubData.MemberArr[i], req.Date,
				clubScoreMap, total)
			rspBody.MemberArr = append(rspBody.MemberArr, item)
		}

		for i := 0; i < len(clubData.DirectSubordinate); i++ {
			directSubordinateClubData, err := loadClubData(clubData.DirectSubordinate[i].ClubID)
			if err != nil {
				rsp.Status = errorCodeDef.Err_Failed
				glog.Error("onGetClubMember() err:=", err.Error())
				return rsp
			}
			var item *clubProto.GetClubMemberData
			playerData, _ := directSubordinateClubData.MemberMap[clubData.DirectSubordinate[i].PlayerID]
			if playerData == nil {
				rsp.Status = errorCodeDef.ErrClubRelation
				return rsp
			}

			{
				queryPlayerIDArr = []int64{clubData.DirectSubordinate[i].PlayerID}
				total = make(map[int64]dbCollectionDefine.DBClubPlayerTotal)
				err = db.GetPlayerTotal(directSubordinateClubData.ClubID, queryPlayerIDArr, req.Date, &total)
				if err != nil {
					glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
					return rsp
				}
				clubScoreMap = make(map[int64]collPlayer.PlayerInfo)
				err = db.GetSomePlayerClubScore(queryPlayerIDArr, &clubScoreMap)
				if err != nil {
					glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
					return rsp
				}
			}

			item = NewGetClubMemberData(directSubordinateClubData, playerData, req.Date,
				clubScoreMap, total)
			item.Remark = clubData.DirectSubordinate[i].Remark
			rspBody.MemberArr = append(rspBody.MemberArr, item)
		}

		rspBody.MyClubMemberCount = len(clubData.MemberArr) + len(clubData.DirectSubordinate)
		rspBody.AllMemberCount = len(clubData.MemberArr)

		for _, v := range clubData.Subordinates {
			clubData, err = loadClubData(v)
			if err != nil {
				glog.Warning("not fin club. err. err:=", err.Error(), ",clubID:=", v)
				return rsp
			}
			rspBody.AllMemberCount += len(clubData.MemberArr)
		}
	} else if req.QueryClubID > 0 {
		rspBody.MemberArr = make([]*clubProto.GetClubMemberData, 0, 1)

		if _, ok := clubData.SubordinatesMap[req.QueryClubID]; ok == false {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}

		clubData, err = loadClubData(req.QueryClubID)
		if err != nil {
			glog.Warning("not fin club. err. err:=", err.Error(), ",clubID:=", req.QueryClubID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}

		queryPlayerIDArr := make([]int64, len(clubData.MemberArr))
		for i, v := range clubData.MemberArr {
			queryPlayerIDArr[i] = v.ID
		}

		total := make(map[int64]dbCollectionDefine.DBClubPlayerTotal)
		err = db.GetPlayerTotal(clubData.ClubID, queryPlayerIDArr, req.Date, &total)
		if err != nil {
			glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
			return rsp
		}
		clubScoreMap := make(map[int64]collPlayer.PlayerInfo)
		err = db.GetSomePlayerClubScore(queryPlayerIDArr, &clubScoreMap)
		if err != nil {
			glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
			return rsp
		}

		for i := 0; i < len(clubData.MemberArr); i++ {
			var item *clubProto.GetClubMemberData
			item = NewGetClubMemberData(clubData, clubData.MemberArr[i], req.Date,
				clubScoreMap, total)
			item.Remark = ""
			rspBody.MemberArr = append(rspBody.MemberArr, item)
		}

		for i := 0; i < len(clubData.DirectSubordinate); i++ {
			directSubordinateClubData, err := loadClubData(clubData.DirectSubordinate[i].ClubID)
			if err != nil {
				rsp.Status = errorCodeDef.Err_Failed
				glog.Error("onGetClubMember() err:=", err.Error())
				return rsp
			}
			var item *clubProto.GetClubMemberData
			playerData, _ := directSubordinateClubData.MemberMap[clubData.DirectSubordinate[i].PlayerID]
			if playerData == nil {
				rsp.Status = errorCodeDef.ErrClubRelation
				return rsp
			}

			{
				queryPlayerIDArr = []int64{clubData.DirectSubordinate[i].PlayerID}
				total = make(map[int64]dbCollectionDefine.DBClubPlayerTotal)
				err = db.GetPlayerTotal(directSubordinateClubData.ClubID, queryPlayerIDArr, req.Date, &total)
				if err != nil {
					glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
					return rsp
				}
				clubScoreMap = make(map[int64]collPlayer.PlayerInfo)
				err = db.GetSomePlayerClubScore(queryPlayerIDArr, &clubScoreMap)
				if err != nil {
					glog.Warning("NewGetClubMemberData() err. err:=", err.Error())
					return rsp
				}
			}

			item = NewGetClubMemberData(directSubordinateClubData, playerData, req.Date,
				clubScoreMap, total)
			item.Remark = ""
			rspBody.MemberArr = append(rspBody.MemberArr, item)
		}
	} else {
		rsp.Status = errorCodeDef.Err_Param
	}

	return rsp
}

func onGetClubList(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubList{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.PlayerClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	rspBody := clubProto.SC_GetClubList{}
	rsp.Data = &rspBody

	if req.QueryClubID == 0 && req.QuerySubordinateID == 0 {

		rspBody.ClubArr = make([]*clubProto.GetClubListData, 0, len(clubData.DirectSubordinate))

		tempDirectSubordinateID := make([]int32, 0, len(clubData.DirectSubordinate))
		tempDirectSubordinateID = append(tempDirectSubordinateID, clubData.ClubID)
		for _, v1 := range clubData.DirectSubordinate {
			tempDirectSubordinateID = append(tempDirectSubordinateID, v1.ClubID)
		}

		var clubListArr []*collClub.DBClubData
		clubListArr, err = db.GetClubList(tempDirectSubordinateID)
		if err != nil {
			glog.Warning("db.TotalClubUnusable() err. err:=", err.Error(), ",clubIDarr:=", req.QuerySubordinateID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		for _, v := range clubListArr {
			clubScore, _ := db.GetPlayerClubScore(v.ClubID, v.CreatorID)
			queryItem := clubProto.GetClubListData{
				ClubID:           v.ClubID,
				ClubName:         v.Name,
				ClubCreatorID:    v.CreatorID,
				ClubCreatorName:  LoadPlayerNick_Name(v.CreatorID).Nick,
				IsOpen:           v.IsOpen,
				IsFrozen:         v.IsFrozen,
				Percent:          v.DirectSupervisor.ShowPercentage,
				ManageFee:        commonDef.ScoreToClient(v.ManagementFee),
				ScoreCount:       commonDef.ScoreToClient(v.ClubScoreCount),
				BaoDi:            commonDef.ScoreToClient(v.BaoDiScore),
				ClubCreatorScore: commonDef.ScoreToClient(clubScore),
			}
			if v.DirectSupervisor.ClubID > 0 {
				superClub, err1 := loadClubData(v.DirectSupervisor.ClubID) //db.LoadClub()
				if err1 != nil {
					if err1 == mongo.ErrNoDocuments {
						rsp.Status = errorCodeDef.ErrClubNotExist
					} else {
						glog.Warning("db.LoadClub() err. err:=", err1.Error(), ",clubID:=", v.DirectSupervisor.ClubID)
					}
					return rsp
				}
				queryItem.SuperiorClubID = superClub.ClubID
				queryItem.SuperiorClubCreatorID = superClub.CreatorID
				queryItem.SuperiorClubName = superClub.Name
				queryItem.SuperiorClubCreatorName = LoadPlayerNick_Name(superClub.CreatorID).Nick
			}

			//tempClubIDArr := v.Subordinates
			//tempClubIDArr = append(tempClubIDArr, v.ClubID)

			unusableScore := v.UnusableScore * -1 //int64(0)
			//unusableScore, err = db.TotalClubUnusable(tempClubIDArr)
			//if err != nil {
			//	glog.Warning("db.TotalClubUnusable() err. err:=", err.Error(), ",clubIDarr:=", tempClubIDArr)
			//	rsp.Status = errorCodeDef.Err_Failed
			//	return rsp
			//}
			queryItem.Unusable = commonDef.ScoreToClient(unusableScore)

			rspBody.ClubArr = append(rspBody.ClubArr, &queryItem)
		}
		rspBody.DirectlyClubCount = len(clubData.DirectSubordinate)
		rspBody.AllSubClubCount = len(clubData.Subordinates) + 1

	} else if req.QueryClubID != 0 {
		if req.QueryClubID != clubData.ClubID {
			if _, ok := clubData.SubordinatesMap[req.QueryClubID]; ok == false {
				rsp.Status = errorCodeDef.ErrClubRelation
				return rsp
			}
		}
		clubData, err = db.LoadClub(req.QueryClubID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rsp.Status = errorCodeDef.ErrClubNotExist
			} else {
				glog.Warning("db.LoadClub() err. err:=", err.Error())
			}
			return rsp
		}

		clubScore, _ := db.GetPlayerClubScore(clubData.ClubID, clubData.CreatorID)
		queryItem := clubProto.GetClubListData{
			ClubID:           clubData.ClubID,
			ClubName:         clubData.Name,
			ClubCreatorID:    clubData.CreatorID,
			ClubCreatorName:  LoadPlayerNick_Name(clubData.CreatorID).Nick,
			IsOpen:           clubData.IsOpen,
			IsFrozen:         clubData.IsFrozen,
			Percent:          clubData.DirectSupervisor.ShowPercentage,
			ManageFee:        commonDef.ScoreToClient(clubData.ManagementFee),
			ScoreCount:       commonDef.ScoreToClient(clubData.ClubScoreCount),
			BaoDi:            commonDef.ScoreToClient(clubData.BaoDiScore),
			ClubCreatorScore: commonDef.ScoreToClient(clubScore),
		}
		if clubData.DirectSupervisor.ClubID > 0 {
			superClub, err1 := loadClubData(clubData.DirectSupervisor.ClubID) //db.LoadClub(clubData.DirectSupervisor.ClubID)
			if err1 != nil {
				if err1 == mongo.ErrNoDocuments {
					rsp.Status = errorCodeDef.ErrClubNotExist
				} else {
					glog.Warning("db.LoadClub() err. err:=", err1.Error(), ",clubID:=", clubData.DirectSupervisor.ClubID)
				}
				return rsp
			}
			queryItem.SuperiorClubID = superClub.ClubID
			queryItem.SuperiorClubCreatorID = superClub.CreatorID
			queryItem.SuperiorClubName = superClub.Name
			queryItem.SuperiorClubCreatorName = LoadPlayerNick_Name(superClub.CreatorID).Nick
		}

		//tempClubIDArr := clubData.Subordinates
		//tempClubIDArr = append(tempClubIDArr, req.QueryClubID)

		unusableScore := clubData.UnusableScore * -1 //int64(0)
		//unusableScore, err = db.TotalClubUnusable(tempClubIDArr)
		//if err != nil {
		//	glog.Warning("db.TotalClubUnusable() err. err:=", err.Error(), ",clubIDarr:=", tempClubIDArr)
		//	rsp.Status = errorCodeDef.Err_Failed
		//	return rsp
		//}
		var clubListArr []*collClub.DBClubData
		clubListArr, err = db.GetClubList([]int32{req.QueryClubID})
		if err != nil {
			glog.Warning("db.TotalClubUnusable() err. err:=", err.Error(), ",clubIDarr:=", req.QueryClubID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		if len(clubListArr) > 0 {
			queryItem.ScoreCount = commonDef.ScoreToClient(clubListArr[0].ClubScoreCount)
		}

		queryItem.Unusable = commonDef.ScoreToClient(unusableScore)
		rspBody.ClubArr = []*clubProto.GetClubListData{&queryItem}
	} else if req.QuerySubordinateID != 0 {
		if req.QuerySubordinateID != clubData.ClubID {
			if _, ok := clubData.SubordinatesMap[req.QuerySubordinateID]; ok == false {
				rsp.Status = errorCodeDef.ErrClubRelation
				return rsp
			}
		}

		clubData, err = loadClubData(req.QuerySubordinateID) //db.LoadClub(req.QuerySubordinateID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rsp.Status = errorCodeDef.ErrClubNotExist
			} else {
				glog.Warning("db.LoadClub() err. err:=", err.Error())
			}
			return rsp
		}

		rspBody.ClubArr = make([]*clubProto.GetClubListData, 0, len(clubData.DirectSubordinate))

		tempDirectSubordinateID := make([]int32, 0, len(clubData.DirectSubordinate))
		for _, v1 := range clubData.DirectSubordinate {
			tempDirectSubordinateID = append(tempDirectSubordinateID, v1.ClubID)
		}

		var clubListArr []*collClub.DBClubData
		clubListArr, err = db.GetClubList(tempDirectSubordinateID)
		if err != nil {
			glog.Warning("db.TotalClubUnusable() err. err:=", err.Error(), ",clubIDarr:=", req.QuerySubordinateID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		for _, v := range clubListArr {
			queryItem := clubProto.GetClubListData{
				ClubID:          v.ClubID,
				ClubName:        v.Name,
				ClubCreatorID:   v.CreatorID,
				ClubCreatorName: LoadPlayerNick_Name(v.CreatorID).Nick,
				IsOpen:          v.IsOpen,
				IsFrozen:        v.IsFrozen,
				Percent:         v.DirectSupervisor.ShowPercentage,
				ManageFee:       commonDef.ScoreToClient(v.ManagementFee),
				ScoreCount:      commonDef.ScoreToClient(v.ClubScoreCount),
				BaoDi:           commonDef.ScoreToClient(v.BaoDiScore),
			}
			if v.DirectSupervisor.ClubID > 0 {
				superClub, err1 := loadClubData(v.DirectSupervisor.ClubID) //db.LoadClub(v.DirectSupervisor.ClubID)
				if err1 != nil {
					if err1 == mongo.ErrNoDocuments {
						rsp.Status = errorCodeDef.ErrClubNotExist
					} else {
						glog.Warning("db.LoadClub() err. err:=", err1.Error(), ",clubID:=", v.DirectSupervisor.ClubID)
					}
					return rsp
				}
				queryItem.SuperiorClubID = superClub.ClubID
				queryItem.SuperiorClubCreatorID = superClub.CreatorID
				queryItem.SuperiorClubName = superClub.Name
				queryItem.SuperiorClubCreatorName = LoadPlayerNick_Name(superClub.CreatorID).Nick
			}

			//tempClubIDArr := v.Subordinates
			//tempClubIDArr = append(tempClubIDArr, v.ClubID)

			unusableScore := v.UnusableScore * -1 //int64(0)
			//unusableScore, err = db.TotalClubUnusable(tempClubIDArr)
			//if err != nil {
			//	glog.Warning("db.TotalClubUnusable() err. err:=", err.Error(), ",clubIDarr:=", tempClubIDArr)
			//	rsp.Status = errorCodeDef.Err_Failed
			//	return rsp
			//}
			queryItem.Unusable = commonDef.ScoreToClient(unusableScore)

			rspBody.ClubArr = append(rspBody.ClubArr, &queryItem)
		}
		rspBody.DirectlyClubCount = len(clubData.DirectSubordinate)
		rspBody.AllSubClubCount = len(clubData.Subordinates)
	} else {
		rsp.Status = errorCodeDef.Err_Param
	}

	return rsp
}

func onGetClubScoreLog(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubScoreLog{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	if req.CurPage < 0 || req.CurPage > 1000 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	if len(req.LogType) < 1 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	clubData, err := loadClubData(req.ClubID)
	if err != nil {
		glog.Warning(err.Error(), ",", req.ClubID)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}
	if clubData.IsShowScoreWater == false {
		if clubData.CreatorID == msg.SenderID {
		} else if _, ok := clubData.AdminMemberMap[msg.SenderID]; ok == true {
		} else {
			return rsp
		}
	}

	logTypeMap := map[int32]int32{}
	for _, v := range req.LogType {
		logTypeMap[v] = v
	}
	req.LogType = make([]int32, 0, 10)

	for _, v := range logTypeMap {
		req.LogType = append(req.LogType, v)

		switch v {
		case 0, 1, 6, 7, 10:
			if rsp.Status != 0 {
				return rsp
			}
		default:
		}
	}
	rsp.Status = 0
	clubScoreLog := &clubProto.SC_GetClubScoreLog{}
	rsp.Data = &clubScoreLog

	mzClubID := clubData.ClubID
	if clubData.MengZhuClubID != 0 {
		mzClubID = clubData.MengZhuClubID
	}

	clubScoreLog.LogArr, err = db.GetClubScoreLog(mzClubID, req.ClubID, req.Data, msg.SenderID, req.LogType, req.CurPage, req.PageSize)
	if err != nil {
		glog.Warning("db.GetClubScoreLog() err.err:=", err.Error(),
			",clubID:=", req.ClubID, ",logType:=", req.LogType, ",uid:=", msg.SenderID)
		rsp.Status = errorCodeDef.Err_Failed
	}

	return rsp
}

func onGetClubOperationLog(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubOperationLog{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	rsp.Status, _ = checkClubPower(req.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	rsp.Data, err = db.GetClubOperationLog(req.ClubID)
	if err != nil {
		glog.Warning("db.onGetClubOperationLog() err.err:=", err.Error(), ",clubID:=", req.ClubID)
		rsp.Status = errorCodeDef.Err_Failed
	}
	return rsp
}

func onGetClubGameRecord(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubGameRecord{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.OperClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	if req.CurPage < 0 || req.PageSize < 1 || req.PageSize > 50 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	var (
		rspBody   clubProto.SC_GetClubGameRecord
		curDate   time.Time
		clubIDArr []int32
	)

	if req.QClubID == 0 && req.PlayerID == 0 && req.QClubPlayID == 0 && req.QTableID == 0 {
		clubIDArr = clubData.Subordinates
		clubIDArr = append(clubIDArr, clubData.ClubID)
	} else if req.PlayerID != 0 {
		playerClubID := isBelongToClub(req.PlayerID, clubData)
		if playerClubID == 0 {
			rsp.Status = errorCodeDef.Err_Not_Find_Item
			return rsp
		}
		clubIDArr = []int32{playerClubID}
	} else if req.QClubID != 0 {
		if _, ok := clubData.SubordinatesMap[req.QClubID]; ok == false {
			if req.QClubID != req.OperClubID {
				rsp.Status = errorCodeDef.ErrClubRelation
				return rsp
			}
		}
		subClubData, _ := loadClubData(req.QClubID)
		if subClubData == nil {
			rsp.Status = errorCodeDef.ErrClubNotExist
			return rsp
		}

		clubIDArr = subClubData.Subordinates
		clubIDArr = append(clubIDArr, req.QClubID)
	}

	rspBody.Arr, curDate, err = db.GetClubGameRecordTotal(clubIDArr, req.PlayerID, req.Date, req.QClubPlayID, req.QTableID,
		req.PageSize, req.CurPage)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
	}

	rspBody.CurTime = curDate.Unix()

	rsp.Data = &rspBody
	return rsp
}

func onGetClubPlayerTotal(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubPlayerTotal{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	//rsp.Status, _ = checkClubPower(req.OperClubID, msg.SenderID)
	//if rsp.Status != 0 {
	//	return rsp
	//}

	var rspBody clubProto.SC_GetClubPlayerTotal

	rspBody.CurTime = time.Now().Unix()
	if req.Date == 0 {
		year, month, day := time.Now().Date()
		req.Date, _ = strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	}

	playerTotal := make(map[int64]dbCollectionDefine.DBClubPlayerTotal)

	err = db.GetPlayerTotal(req.ClubID, []int64{req.PlayerID}, req.Date, &playerTotal)
	if err != nil {
		glog.Warning("db.GetPlayerTotal() err.err:=", err.Error(),
			",clubID:=", req.ClubID, ",uid:=", msg.SenderID)
		rsp.Status = errorCodeDef.Err_Failed
	}

	for _, v := range playerTotal {
		rspBody.TotalGameScore = commonDef.ScoreToClient(v.GameScore)
		rspBody.TotalZengSong = commonDef.ScoreToClient(v.ZengSong)
		//rspBody.TotalGongXianScore = commonDef.ScoreToClient(v.GongXian)
		//rspBody.TotalJiangLiScore = commonDef.ScoreToClient(v.JiangLiScore)
		//rspBody.TotalBaodi = commonDef.ScoreToClient(v.BaoDi)
	}

	clubTotalMap := map[int32]*dbCollectionDefine.DBClubTotal{}
	clubTotalMap, err = db.GetClubTotal([]int32{req.ClubID}, req.Date)
	if err != nil {
		if err != mongo.ErrNoDocuments {
			glog.Warning(err.Error())
		}
	} else {
		v, ok := clubTotalMap[req.ClubID]
		if ok == true {
			rspBody.TotalJiangLiScore = commonDef.ScoreToClient(v.JiangLi)
			rspBody.TotalBaodi = commonDef.ScoreToClient(v.BaoDi)
		}
	}

	rsp.Data = &rspBody
	return rsp
}

func onGetClubTotal(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetClubTotal{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.OperClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	rspBody := clubProto.SC_GetClubTotal{}
	rspBody.Total = make([]*clubProto.RspClubTotalItem, 0, len(clubData.DirectSubordinate))

	year, month, day := time.Now().Date()
	rspBody.CurDate, _ = strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	if req.Date == 0 {
		req.Date = rspBody.CurDate
	}

	// 队长玩家ID -> 圈子ID
	if req.QPlayerID != 0 {
		playerClubArr, err := db.GetPlayerClub(req.QPlayerID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				rsp.Status = errorCodeDef.ErrClubRelation
				return rsp
			}
			glog.Warning(err.Error(), ", ", req.QPlayerID)
			rsp.Status = errorCodeDef.Err_Failed
			return rsp
		}
		for _, v := range playerClubArr {
			if _, ok := clubData.SubordinatesMap[v.ClubID]; ok == true {
				tempClubData, err := loadClubData(v.ClubID)
				if err != nil {
					glog.Warning(err.Error(), ", ", req.QPlayerID)
					rsp.Status = errorCodeDef.Err_Failed
					return rsp
				}
				if tempClubData.CreatorID == req.QPlayerID {
					req.QClubID = v.ClubID
					break
				}
			}
		}
		if req.QClubID == 0 {
			rsp.Status = errorCodeDef.ErrClubRelation
			return rsp
		}
	}

	remarkMap := map[int32]string{}
	queryClubIDArr := make([]int32, 0, len(clubData.DirectSubordinate))
	if req.QClubID == 0 {
		rspBody.AllSubClubCount = len(clubData.Subordinates) + 1
		rspBody.AllDirSubClubCount = len(clubData.DirectSubordinate)
		queryClubIDArr = append(queryClubIDArr, clubData.ClubID)

		for i := 0; i < len(clubData.DirectSubordinate); i++ {
			queryClubIDArr = append(queryClubIDArr, clubData.DirectSubordinate[i].ClubID)
			remarkMap[clubData.DirectSubordinate[i].ClubID] = clubData.DirectSubordinate[i].Remark
		}
	} else {
		if _, ok := clubData.SubordinatesMap[req.QClubID]; ok == false {
			rsp.Status = errorCodeDef.ErrClubRelation
			return rsp
		}

		queryClubIDArr = append(queryClubIDArr, req.QClubID)
		clubData, err = loadClubData(req.QClubID)
		if err != nil {
			glog.Warning("onGetClubTotal() err:=", err.Error())
			rsp.Status = errorCodeDef.ErrClubOperationInvalid
			return rsp
		}
		for i := 0; i < len(clubData.DirectSubordinate); i++ {
			queryClubIDArr = append(queryClubIDArr, clubData.DirectSubordinate[i].ClubID)
		}
	}

	clubScoreMap, err := db.GetSomeClubScore(queryClubIDArr)
	if err != nil {
		glog.Warning("onGetClubTotal() err. err:=", err.Error(), ",param:=", req)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	findMap, err := db.GetClubTotal(queryClubIDArr, req.Date)
	if err != nil {
		glog.Warning("onGetClubTotal() err. err:=", err.Error(), ",param:=", req)
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	var tempClubData *collClub.DBClubData
	for _, v := range queryClubIDArr {
		tempClubData, err = loadClubData(v)
		if err != nil {
			glog.Warning("loadClubData() err. err:=", err.Error(), ",clubID:=", v)
			continue
		}
		item := clubProto.RspClubTotalItem{PlayerID: tempClubData.CreatorID}
		item.JoinTime = tempClubData.CreateTime.Unix()
		playerClubInfo, _ := db.GetPlayerClubInfo(tempClubData.CreatorID)
		if playerClubInfo != nil {
			for _, v := range playerClubInfo.ClubData {
				if v.ClubID == tempClubData.ClubID {
					item.Score = commonDef.ScoreToClient(v.Score)
					break
				}
			}
		}
		playerInfo := LoadPlayerNick_Name(tempClubData.CreatorID)
		item.PlayerHead = playerInfo.HeadURL
		item.PlayerNick = playerInfo.Nick
		item.ClubID = tempClubData.ClubID
		item.ClubName = tempClubData.Name
		item.BiLi = tempClubData.DirectSupervisor.ShowPercentage
		item.IsOpen = tempClubData.IsOpen
		item.IsFrozen = tempClubData.IsFrozen
		item.IsKickOutMember = tempClubData.IsKickOutMember
		item.IsKickOutLeague = tempClubData.IsKickOutLeague
		item.JingJie = commonDef.ScoreToClient(tempClubData.BaoDiScore)
		if score, ok := clubScoreMap[tempClubData.ClubID]; ok == true {
			item.ZongFen = commonDef.ScoreToClient(score[0])
			item.FuFen = commonDef.ScoreToClient(score[1])
		}
		if vs, ok := remarkMap[v]; ok == true {
			item.Remark = vs
		}

		queryRspItem, _ := findMap[v]
		if queryRspItem != nil {
			item.GameCount = queryRspItem.GameCount
			item.GameScoreText = commonDef.ScoreToClient(queryRspItem.GameScore)
			item.HaoKaText = commonDef.ScoreToClient(queryRspItem.HaoKa)
			item.XiaoHaoScore = commonDef.ScoreToClient(queryRspItem.GongXian)
			item.JiangLi = commonDef.ScoreToClient(queryRspItem.JiangLi)
		}

		rspBody.Total = append(rspBody.Total, &item)
	}

	rsp.Data = &rspBody
	return rsp
}

func isBelongToClub(uid int64, clubData *collClub.DBClubData) int32 {

	playerClubInfo, err := db.GetPlayerClubInfo(uid)
	if err != nil {
		glog.Warning("GetPlayerClubInfo() err. err:=", err.Error(), ",uid:=", uid)
		return 0
	}

	for _, v := range playerClubInfo.ClubData {
		if _, ok := clubData.SubordinatesMap[v.ClubID]; ok == true {
			return v.ClubID
		}
	}

	if _, ok := clubData.MemberMap[uid]; ok == true {
		return clubData.ClubID
	}
	return 0
}

func onGetMemberJudgeLog(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetMemberJudgeLog{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	if req.CurPage < 0 || req.CurPage > 1000 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}

	clubScoreLog := &clubProto.SC_GetClubScoreLog{}
	rsp.Data = &clubScoreLog

	mzClubID := clubData.ClubID
	if clubData.MengZhuClubID != 0 {
		mzClubID = clubData.MengZhuClubID
	}

	clubScoreLog.LogArr, err = db.GetMemberJudgeLog(mzClubID, req.ClubID, req.Date, req.UID, req.Category, req.CurPage, req.PageSize)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Failed
		glog.Warning("GetMemberJudgeLog() err. err:=", err.Error())
		return rsp
	}

	//nowTT := time.Now()
	//year, month, day := nowTT.Date()
	//date__, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
	//playerTotal := make(map[int64]dbCollectionDefine.DBClubPlayerTotal)

	//err = db.GetPlayerTotal(req.ClubID, []int64{req.UID}, date__, &playerTotal)
	//if err != nil {
	//	glog.Warning("db.GetPlayerClubInfo() err.err:=", err.Error(),
	//		",clubID:=", req.ClubID, ",uid:=", req.UID)
	//	rsp.Status = errorCodeDef.Err_Failed
	//}
	//
	//for _, v := range playerTotal {
	//	clubScoreLog.TotalGameScore = commonDef.ScoreToClient(v.GameScore)
	//	clubScoreLog.TotalZengSong = commonDef.ScoreToClient(v.ZengSong)
	//	clubScoreLog.TotalGongXianScore = commonDef.ScoreToClient(v.GongXian)
	//	clubScoreLog.TotalJiangLiScore = commonDef.ScoreToClient(v.JiangLiScore)
	//}

	return rsp
}

func onGetPlayerInLeague(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_QueryPlayerLeague{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	var operClubData *collClub.DBClubData
	rsp.Status, operClubData = checkClubPower(req.OperationClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	for _, v := range getAllMember(operClubData) {
		if v.ID == req.PlayerID {
			req.PlayerClubID = v.OfClubID
		}
	}
	if req.PlayerClubID == 0 {
		rsp.Status = errorCodeDef.ErrClubRelation
		return rsp
	}

	var playerClubData *collClub.DBClubData
	playerClubData, err = loadClubData(req.PlayerClubID)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Not_Find_Item
		return rsp
	}
	if _, ok := playerClubData.MemberMap[req.PlayerID]; ok == false {
		rsp.Status = errorCodeDef.Err_Not_Find_Item
		return rsp
	}

	clubDataArr := make([]*collClub.DBClubData, 0, 6)
	getDirSuperiorClubData(&clubDataArr, req.OperationClubID, req.PlayerClubID)

	arr := make([]clubProto.QueryPlayerLeagueItem, len(clubDataArr))

	for i, v := range clubDataArr {
		arr[i].ClubID = v.ClubID
		arr[i].ClubName = v.Name
		arr[i].UID = v.CreatorID
		tempPlayer := LoadPlayerNick_Name(v.CreatorID)
		if tempPlayer != nil {
			arr[i].HeadURL, arr[i].Name = tempPlayer.HeadURL, tempPlayer.Nick
		}
	}
	rsp.Data = arr
	return rsp
}

func getDirSuperiorClubData(arr *[]*collClub.DBClubData, topClubID, clubID int32) {
	tempClubData, err := loadClubData(clubID)
	if err != nil {
		glog.Error("not find club.", clubID, ",err:=", err.Error())
		return
	}
	if tempClubData.DirectSupervisor.ClubID < 1 {
		*arr = append(*arr, tempClubData)
		return
	}
	if tempClubData.ClubID == topClubID {
		*arr = append(*arr, tempClubData)
		return
	}

	getDirSuperiorClubData(arr, topClubID, tempClubData.DirectSupervisor.ClubID)
	*arr = append(*arr, tempClubData)
}

func getTwoPlayerTogetherData(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetTwoPlayerTogetherData{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}

	if len(req.PlayerID) != 2 {
		rsp.Status = errorCodeDef.Err_Param
		return rsp
	}

	rsp.Data, err = db.GetTwoPlayerTogetherData(req.MZClubID, req.Date, req.PlayerID)
	if err != nil {
		glog.Warning("GetTwoPlayerTogetherData() err. err:=", err.Error())
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	return rsp
}

func onGetMzMember(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	mzClubID, ok := mzMemberMap[msg.SenderID]
	if ok == false {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	memberData, ok := mzClubPlayerOnlineMap[mzClubID]
	if ok == false {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}
	rsp.Data = memberData.uid

	return rsp
}

func onInviteMzOnlineMember(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_InviteMzOnlineMember{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Unidentifiable
		return rsp
	}
	if len(req.PlayerID) > 8 {
		rsp.Status = errorCodeDef.Err_Failed
		return rsp
	}

	noticeInviteJoinTable := mateProto.MessageMaTe{MessageID: clubProto.ID_NoticeInviteJoinTable}
	noticeInviteJoinTable.Data = msg.Data

	for _, v := range req.PlayerID {
		noticeInviteJoinTable.SenderID = v
		wrapMQ.PublishProto(msg.Source, &noticeInviteJoinTable)
	}

	return rsp
}
