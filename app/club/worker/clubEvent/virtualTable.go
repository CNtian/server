package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"go.mongodb.org/mongo-driver/mongo"
	"vvService/appClub/db"
	clubProto "vvService/appClub/protoDefine"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func onUpdateVirtualTableConfig(senderID int64, param *clubProto.CS_ClubOperation, data *[]byte) (int32, error) {
	updateParam := struct {
		Oper int                             `json:"oper"` // 1:新增|更新   2:删除
		Item collClub.VirtualTableConfigItem `json:"item"`
	}{}

	err := json.Unmarshal(*data, &updateParam)
	if err != nil {
		return errorCodeDef.Err_Illegal_JSON, nil
	}

	rspCode, clubData := checkClubPower(param.OperationClubID, senderID)
	if rspCode != 0 {
		return rspCode, nil
	}
	if clubData.MengZhuClubID > 0 || clubData.CreatorID != senderID {
		return errorCodeDef.ErrClubOperationInvalid, nil
	}

	if updateParam.Oper == 1 {

		if updateParam.Item.Status == 0 {
			if updateParam.Item.ShowTableCount2 < updateParam.Item.ShowTableCount1 || updateParam.Item.ShowTableCount1 < 0 {
				return errorCodeDef.Err_Param, nil
			}
			if updateParam.Item.RunDuration2 < updateParam.Item.RunDuration1 || updateParam.Item.RunDuration1 < 0 {
				return errorCodeDef.Err_Param, nil
			}
		}

		updateParam.Item.Loop1, updateParam.Item.Loop2 = 1, 30

		_, _, err := db.UpdateVirtualTableConfig(clubData.ClubID, &updateParam.Item)
		if err != nil {
			glog.Warning("UpdateVirtualTableConfig()", err.Error())
			return errorCodeDef.Err_Failed, err
		}

		virtualTableEvent.PostMaTeEvent(nil)
		return 0, nil
	}

	return rspCode, nil
}

func onGetVirtualTableConfig(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {
	reqParam := struct {
		ClubID int32 `json:"clubID"`
	}{}

	rsp := &mateProto.JsonResponse{}

	err := json.Unmarshal(msg.Data, &reqParam)
	if err != nil {
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return rsp
	}

	rspCode, clubData := checkClubPower(reqParam.ClubID, msg.SenderID)
	if rspCode != 0 {
		rsp.Status = rspCode
		return rsp
	}
	if clubData.MengZhuClubID > 0 || clubData.CreatorID != msg.SenderID {
		rsp.Status = errorCodeDef.ErrClubOperationInvalid
		return rsp
	}

	arr := []collClub.VirtualTableConfigItem{}
	err = db.GetVirtualTableConfig(reqParam.ClubID, &arr)
	if err != nil {
		if err != mongo.ErrNilDocument {
			glog.Warning("GetVirtualTableConfig(). clubID:=", reqParam.ClubID, ",err:=", err.Error())
			return rsp
		}
		rsp.Status = errorCodeDef.Err_Not_Find_Item
	} else {
		rsp.Data = arr
	}

	return rsp
}
