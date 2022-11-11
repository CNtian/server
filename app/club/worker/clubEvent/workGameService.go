package clubEvent

import (
	"encoding/json"
	"github.com/golang/glog"
	"time"
	"vvService/commonPackge/errorCodeDef"
	"vvService/commonPackge/mateProto"
)

type Service struct {
	Status        int32                  // 状态
	Heartbeat     int64                  // 时间戳
	ID            string                 // 服务ID
	SupportGameID []mateProto.SignInInfo // 支持的玩法
	TableTotal    int32                  // 桌子总数
}

type GroupGamePlay struct {
	ServiceArr  []*Service
	CurUseIndex int
}

var (
	gameIDeMap     map[int32]*GroupGamePlay // key:玩法ID value:游戏服务标识
	gameServiceMap map[string]*Service      // key:游戏服ID value:游戏服
)

func onGameSignIn(msg *mateProto.MessageMaTe) *mateProto.JsonResponse {

	rsp := &mateProto.JsonResponse{}
	req := mateProto.MsgGameSignIn{}

	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Warning("onGameSignIn() err:=")
		rsp.Status = errorCodeDef.Err_Illegal_JSON
		return rsp
	}

	//if req.Status != 0 {
	//	glog.Warning("game service status. serviceID:=", msg.Source, ",status:=", req.Status)
	//	delService(msg.Source)
	//	return rsp
	//}

	delService(msg.Source)

	if gameIDeMap == nil {
		gameIDeMap = make(map[int32]*GroupGamePlay)
	}
	if gameServiceMap == nil {
		gameServiceMap = make(map[string]*Service)
	}

	for _, v := range req.SupportPlayIDArr {

		// 服务 是否已经存在
		service, ok1 := gameServiceMap[msg.Source]
		if ok1 == false {
			service = &Service{ID: msg.Source, Heartbeat: time.Now().Unix()}
			gameServiceMap[msg.Source] = service
		}

		// 玩法ID 是否已存在
		groupGamePlay, ok := gameIDeMap[v.GameID]
		if ok == false {
			groupGamePlay = &GroupGamePlay{ServiceArr: make([]*Service, 0)}
			gameIDeMap[v.GameID] = groupGamePlay
		}

		// 玩法ID 是否已存在
		isExist := false
		for _, v1 := range groupGamePlay.ServiceArr {
			if v1.ID == msg.Source {
				service.Heartbeat = time.Now().Unix()
				service.Status = 0
				isExist = true
				break
			}
		}
		if isExist == false {
			groupGamePlay.ServiceArr = append(groupGamePlay.ServiceArr, service)
			gameIDeMap[v.GameID] = groupGamePlay
		}
	}

	gameServiceMap[msg.Source].SupportGameID = req.SupportPlayIDArr

	return rsp
}

func findGameServiceID(playID int32) string {
	groupService, ok := gameIDeMap[playID]
	if ok == false || len(groupService.ServiceArr) == 0 {
		return ""
	}

	const heartbeatD  = 121

	delServiceArr := make([]*Service, 0)

	curTT := time.Now().Unix()
	for i := groupService.CurUseIndex; i < len(groupService.ServiceArr); i++ {
		if groupService.ServiceArr[i].Status == 0 && (curTT-groupService.ServiceArr[i].Heartbeat) < heartbeatD {
			groupService.CurUseIndex = i + 1
			return groupService.ServiceArr[i].ID
		}
		if (curTT - groupService.ServiceArr[i].Heartbeat) >= heartbeatD {
			delServiceArr = append(delServiceArr, groupService.ServiceArr[i])
		}
	}

	for i := 0; i < groupService.CurUseIndex && i < len(groupService.ServiceArr); i++ {
		if groupService.ServiceArr[i].Status == 0 && (curTT-groupService.ServiceArr[i].Heartbeat) < heartbeatD {
			groupService.CurUseIndex = i + 1
			return groupService.ServiceArr[i].ID
		}
		if (curTT - groupService.ServiceArr[i].Heartbeat) >= heartbeatD {
			delServiceArr = append(delServiceArr, groupService.ServiceArr[i])
		}
	}

	for _, v := range delServiceArr {
		delete(gameServiceMap, v.ID)

		for _, groupService := range gameIDeMap {
			for i, v2 := range groupService.ServiceArr {
				if v2.ID == v.ID {
					groupService.ServiceArr = append(groupService.ServiceArr[:i], groupService.ServiceArr[i+1:]...)
					break
				}
			}
		}
	}

	return ""
}

func onGameServiceStatus(msg *mateProto.MessageMaTe) (rsp *mateProto.JsonResponse) {
	req := mateProto.MsgBroadGameServiceStatus{}

	err := json.Unmarshal(msg.Data, &req)
	if err != nil {
		glog.Error("onGameServiceStatus() err:=")
		return
	}

	v, ok := gameServiceMap[msg.Source]
	if ok == false {
		glog.Error("onGameServiceStatus(). not find source:=", msg.Source)
		return
	}

	v.Status = req.Status
	v.Heartbeat = time.Now().Unix()
	v.TableTotal = req.TableTotal

	return
}

func FindGameService(serviceID string) bool {
	_, ok := gameServiceMap[serviceID]
	return ok
}

func delService(id string) {

	delete(gameServiceMap, id)

	for _, groupService := range gameIDeMap {
		for i, v2 := range groupService.ServiceArr {
			if v2.ID == id {
				groupService.ServiceArr = append(groupService.ServiceArr[:i], groupService.ServiceArr[i+1:]...)
				break
			}
		}
	}
}

//func GetAllGameService() map[string]*Service {
//	return gameServiceMap
//}

//func onCleanHeartbeatTimeOutService() {
//	nowTT := time.Now().Unix()
//
//	for _, v := range gameServiceMap {
//		if nowTT-v.Heartbeat > 3600*2 {
//			delService(v.ID)
//		}
//	}
//}
