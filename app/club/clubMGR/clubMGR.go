package clubMGR

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"vvService/appClub/localConfig"
	clubProto "vvService/appClub/protoDefine"
	"vvService/appClub/worker/clubEvent"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/mateProto"
	collClub "vvService/dbCollectionDefine/club"
)

func HandleMQData(data []byte) {

	msg := mateProto.MessageMaTe{}

	nowT1 := time.Now()
	defer func() {
		nowT2 := time.Now()
		dur := nowT2.Sub(nowT1)
		if dur.Milliseconds() >= 50 {
			glog.Warning("msg handle time too long. expend ", dur.Milliseconds(), " ms."+
				" msg:=", msg.MessageID, ",sender ID:=", msg.SenderID, ",data:=", string(data))
		}
	}()

	err := json.Unmarshal(data, &msg)
	if err != nil {
		glog.Warning("proto.Unmarshal", err.Error())
		return
	}

	switch msg.MessageID {
	case mateProto.ID_BroadGameServiceStatus:
	default:
		commonDef.Info.Info("uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",source:=", msg.Source, ",data:=", string(msg.Data))
	}

	var rsp *mateProto.JsonResponse

	switch msg.MessageID {
	//case clubProto.ID_NewProxy:
	//	var uid int64
	//	var isLaunchClub bool
	//	rsp, uid, isLaunchClub = onNewProxyClub(&msg)
	//	if rsp.Status == 0 && isLaunchClub == true {
	//		LaunchMengZhu(rsp.Data.(*collClub.DBClubData).ClubID)
	//		time.Sleep(100 * time.Millisecond)
	//		clubEvent.NoticeCreateClub(uid)
	//	}
	case clubProto.ID_InviteToProxy:
		rsp = onInviteToProxy(&msg)
	case clubProto.ID_HandleInviteToProxy:
		var isLaunchClub bool
		rsp, isLaunchClub = onHandleInviteToProxy(&msg)
		if rsp.Status == 0 && isLaunchClub == true {
			LaunchMengZhu(rsp.Data.(*collClub.DBClubData).ClubID)
			time.Sleep(100 * time.Millisecond)
			clubEvent.NoticeCreateClub(msg.SenderID, msg.Source)
		}
	case clubProto.ID_CancelProxy:
		rsp = onCancelProxy(&msg)
	case clubProto.ID_GetProxyClubID:
		rsp = onGetMyAllMengZhu(&msg)
	case clubProto.ID_UpdateClubLevel:
		rsp = onUpdateClubLevel(&msg)
	case clubProto.ID_UpdateClubStatus:
		rsp = onUpdateClubStatus(&msg)
	case clubProto.ID_GetProxyReportList:
		rsp = onGetProxyReportList(&msg)
	case clubProto.ID_GetProxyList:
		rsp = onGetProxyList(&msg)
	case clubProto.ID_GiveRoomCard:
		rsp = onGiveRoomCard(&msg)
	case clubProto.ID_GiveRoomCardList:
		rsp = onGiveRoomCardList(&msg)
	case clubProto.ID_NewClub:
	//rsp = clubEvent.OnNewClub(&msg)
	//if rsp.Status == 0 {
	//	LaunchMengZhu(rsp.Data.(*collClub.DBClubData).ClubID)
	//	time.Sleep(200 * time.Millisecond)
	//}
	case mateProto.ID_NoticDailyChanged:
		onNoticeDailyChanged(&msg)
	case clubProto.ID_GetPlayerClubInfo:
		rsp = GetPlayerClubInfo(&msg)
	case clubProto.ID_ApplyJoinClub:
		rsp = JoinClub(&msg)
	case clubProto.ID_GetClubData:
		rsp = GetClubDetail(&msg)
	case mateProto.ID_NoticeClubMGRLaunch:
		MengZhuLogin(&msg)
	case mateProto.ID_UpdateClubData:
		onUpdateClubData(&msg)
	case mateProto.ID_BroadGameServiceStatus,
		mateProto.ID_GameSignIn,
		mateProto.ID_PlayerInfoUpdate,
		mateProto.ID_HallUpdateClub,
		clubProto.ID_DeleteServiceIDTable:
		for k, _ := range mzClubIDMap {
			wrapMQ.PublishProto(k, &msg)
		}
	default:

	}

	if rsp == nil {
		return
	}

	msg.Data, _ = json.Marshal(rsp)
	err = wrapMQ.PublishProto(msg.Source, &msg)
	if err != nil {
		glog.Warning("uid:=", msg.SenderID,
			" ,wrapMQ.PublishData err. err:=", err.Error(),
			",msg ID:=", msg.MessageID, " ,data:=", string(msg.Data))
	}
}

func LaunchMengZhu(clubID int32) {
	cfgBak := *localConfig.GetConfig()

	cfgBak.ID = fmt.Sprintf("%d", clubID)

	topDir := fmt.Sprintf("%s/%d_mz", GetCurrPath(), clubID)
	_, err := os.Stat(topDir)
	if err != nil {
		if os.IsNotExist(err) == false {
			glog.Warning("os.Stat err.", err.Error(), clubID)
			return
		}
	}
	if os.IsNotExist(err) {
		err = os.Mkdir(topDir, 755)
		if err != nil {
			glog.Warning("os.Mkdir err.", err.Error(), clubID)
			return
		}
	}

	logDir := topDir + "/log"
	_, err = os.Stat(logDir)
	if err != nil {
		if os.IsNotExist(err) == false {
			glog.Warning("os.Stat err.", err.Error(), clubID)
			return
		}
	}
	if os.IsNotExist(err) {
		err = os.Mkdir(logDir, 755)
		if err != nil {
			glog.Warning("os.Mkdir log err.", err.Error(), clubID)
			return
		}
	}

	_, err = CopyFile(topDir+"/tableNumber.list", "./tableNumber.list")
	if err != nil {
		glog.Warning("CopyFile err.", err.Error(), clubID)
		return
	}

	_, err = CopyFile(topDir+"/appClub", "./appClub")
	if err != nil {
		glog.Warning("CopyFile err.", err.Error(), clubID)
		return
	}
	err = os.Chmod(topDir+"/appClub", 777)
	if err != nil {
		glog.Warning("Chmod err.", err.Error(), clubID)
		return
	}

	data, _ := json.Marshal(&cfgBak)
	err = os.WriteFile(topDir+"/config.json", data, 644)
	if err != nil {
		glog.Warning("WriteFile json err.", err.Error(), clubID)
		return
	}

	_, err = CopyFile(topDir+"/restart.sh", "./restart.sh")
	if err != nil {
		glog.Warning("CopyFile err.", err.Error(), clubID)
		return
	}
	err = os.Chmod(topDir+"/restart.sh", 777)
	if err != nil {
		glog.Warning("Chmod err.", err.Error(), clubID)
		return
	}
	//cmd := exec.Command("nohup",topDir+"/appClub","-log_dir","log","-alsologtostderr","-v","2")
	cmd := exec.Command(topDir+"/restart.sh", "appClub")
	//cmd := exec.Command("/bin/sh",topDir+"/restart.sh","appClub")
	//cmd:=exec.Command(topDir+"/restart.sh","appClub")
	//cmd.Stdin = os.Stdin // 给新进程设置文件描述符，可以重定向到文件中
	//cmd.Stdout = os.Stdout
	//cmd.Stderr = os.Stderr
	cmd.Dir = topDir
	cmd.Env = os.Environ()
	err = cmd.Start()
	//outData,err := cmd.CombinedOutput()
	if err != nil {
		glog.Warning("CombinedOutput err.", err.Error(), clubID)
		return
	}
	//glog.Warning("launch mz club out.",string(outData))
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func GetCurrPath() string {
	file, _ := exec.LookPath(os.Args[0])
	path, _ := filepath.Abs(file)
	index := strings.LastIndex(path, string(os.PathSeparator))
	ret := path[:index]
	return ret
}
