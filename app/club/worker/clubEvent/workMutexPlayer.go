package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func onPutMutexPlayer(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_PutMutexPlayer{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Error("onNewMutexPlayer() err:=", err.Error())
		return rsp
	}

	var (
		clubData, tempClubData *collClub.DBClubData
	)
	rsp.Status, clubData = checkClubPower(req.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	if clubData.DirectSupervisor.ClubID != 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	if len(clubData.PlayerMutex) >= 10 && req.ID.IsZero() == true {
		rsp.Status = errorCodeDef.ErrMaxMutexGroup
		return rsp
	}

	// 是否是盟的成员
	{
		tempPlayerMap := make(map[int64]int64)
		clubIDArr := clubData.Subordinates
		clubIDArr = append(clubIDArr, clubData.ClubID)
		for _, v := range req.PlayerID {
			if _, ok := tempPlayerMap[v]; ok == false {
				tempPlayerMap[v] = v
			} else {
				rsp.Status = errorCodeDef.ErrClubMutexPlayerRepeat
				return rsp
			}

			isFindPlayer := false
			for _, vClubID := range clubIDArr {
				tempClubData, err = loadClubData(vClubID)
				if err != nil {
					rsp.Status = errorCodeDef.ErrClubNotExist
					return rsp
				}
				if _, ok := tempClubData.MemberMap[v]; ok == true {
					isFindPlayer = true
					break
				}
			}
			if isFindPlayer == false {
				rsp.Status = errorCodeDef.ErrClubNotMember
				return rsp
			}
		}
	}

	var preMutexPlayerArr []int64
	if req.ID.IsZero() == false {
		for _, v := range clubData.PlayerMutex {
			if req.ID != v.ID {
				continue
			}
			preMutexPlayerArr = v.Player
			err = db.RemoveMutexGroupFromRedis(v.Player)
			if err != nil {
				glog.Warning("db.RemoveMutexGroupFromRedis() err. err:=", err.Error(), ",playerArr:=", v.Player)
				rsp.Status = errorCodeDef.Err_Failed
				return rsp
			}
			break
		}
	}

	rsp.Status, err = db.PutPlayerToMutexGroup(req.ClubID, req.ID, req.PlayerID)
	if err != nil {
		glog.Warning("onNewMutexPlayer() RemoveMutexGroup(). clubID:=", req.ClubID, " ,playerArr:=", preMutexPlayerArr)
		glog.Warning("onNewMutexPlayer() err. err:=", err.Error(), ",param:=", req)
		return rsp
	}
	if rsp.Status != 0 {
		glog.Warning("onNewMutexPlayer() RemoveMutexGroupFromRedis(). clubID:=", req.ClubID, " ,playerArr:=", preMutexPlayerArr)
		return rsp
	}
	err = db.PutMutexGroupToRedis(req.PlayerID)
	if err != nil {
		glog.Warning("PutMutexGroupToRedis() . clubID:=", req.ClubID, " ,playerArr:=", req.PlayerID)
	}

	delLocalClubData(req.ClubID)

	return rsp
}

func onDeleteMutexPlayerGroup(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_DeleteMutexPlayerGroup{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Error("onDeleteMutexPlayer() err:=", err.Error())
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	if clubData.DirectSupervisor.ClubID != 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	rsp.Status, err = delMutexPlayer(clubData, req.ID)

	delLocalClubData(req.ClubID)

	return rsp
}

func delMutexPlayer(clubData *collClub.DBClubData, id primitive.ObjectID) (status int32, err error) {

	status, err = db.DeletePlayerToMutexGroup(clubData.ClubID, id)
	if err != nil {
		glog.Warning("onDeleteMutexPlayer() err. err:=", err.Error(), ",param:=", clubData.ClubID, ",", id.Hex())
		return status, err
	}
	if status != 0 {
		return status, err
	}

	for _, v := range clubData.PlayerMutex {
		if id != v.ID {
			continue
		}

		err = db.RemoveMutexGroupFromRedis(v.Player)
		if err != nil {
			glog.Warning("db.RemoveMutexGroupFromRedis() err. err:=", err.Error(),
				" ,clubID:=", clubData.ClubID,
				" ,playerArr:=", v.Player)
		}
		break
	}
	return status, err
}

func onGetMutexGroup(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	rsp := &mateProto.JsonResponse{}

	req := clubProto.CS_GetMutexPlayer{}
	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Error("onGetMutexGroup() err:=", err.Error())
		return rsp
	}

	var clubData *collClub.DBClubData
	rsp.Status, clubData = checkClubPower(req.ClubID, msg.SenderID)
	if rsp.Status != 0 {
		return rsp
	}
	if clubData.DirectSupervisor.ClubID != 0 {
		rsp.Status = errorCodeDef.ErrFindSuperiorExist
		return rsp
	}

	rsp.Data = clubData.PlayerMutex
	return rsp
}
