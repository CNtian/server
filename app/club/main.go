package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"vvService/appClub/clubMGR"
	"vvService/appClub/db"
	"vvService/appClub/localConfig"
	"vvService/appClub/worker"
	"vvService/appClub/worker/clubEvent"
	"vvService/appClub/worker/tableSnapshot"
	"vvService/appClub/worker/virtualTable"
	"vvService/appClub/wrapMQ"
	commonDef "vvService/commonPackge"
	commonDB "vvService/commonPackge/db"
)

func main() {

	rand.Seed(time.Now().Unix())

	{
		pid := os.Getpid()
		ioutil.WriteFile("./cur.pid", []byte(strconv.Itoa(pid)), 0666)
	}

	mengZhuClubID := 0
	flag.IntVar(&mengZhuClubID, "mzID", 0, "")
	isTest := false
	flag.BoolVar(&isTest, "test", false, "")
	flag.Parse()

	defer glog.Flush()

	// 数字越小等级越高
	glog.Info("Log 数字越小等级越高. 1-warning 2-info")
	commonDef.Warning = glog.V(1)
	commonDef.Info = glog.V(2)

	var cfg *localConfig.LocalConfig

	rand.Seed(time.Now().Unix())

	{
		var err error
		cfg, err = localConfig.LoadConfig("./config.json")
		if err != nil {
			glog.Fatal(err.Error())
			return
		}
	}

	if len(localConfig.GetConfig().VirtualPlayer) == 2 {
		if localConfig.GetConfig().VirtualPlayer[0] >= localConfig.GetConfig().VirtualPlayer[1] {
			glog.Fatal("VirtualPlayer ....")
			return
		}
	} else if len(localConfig.GetConfig().VirtualPlayer) > 2 {
		glog.Fatal("VirtualPlayer ....")
		return
	}

	glog.Warning("init mzID:=", mengZhuClubID, " configID:=", cfg.ID, " ppid:=", os.Getppid())

	if isTest == true {
		clubMGR.LaunchMengZhu(1)
	}

	if mengZhuClubID > 0 {
		cfg.ID = fmt.Sprintf("%d", mengZhuClubID)
	} else if mengZhuClubID == 0 && cfg.ID != "club" {
		t, err := strconv.Atoi(cfg.ID)
		if err != nil {
			glog.Fatal(err.Error())
			return
		}
		mengZhuClubID = t
	}

	glog.Warning("final mzID:=", mengZhuClubID, " configID:=", cfg.ID)

	// Redis
	{
		err := db.ConnectGameRedis(cfg.Redis.IP, cfg.Redis.Password, cfg.Redis.Port, cfg.Redis.GameIndex)
		if err != nil {
			glog.Fatal(err.Error())
			return
		}

		err = db.ConnectPersonalRedis(cfg.Redis.IP, cfg.Redis.Password, cfg.Redis.Port, cfg.Redis.LoginIndex)
		if err != nil {
			glog.Fatal(err.Error())
			return
		}
	}

	// Mongodb
	{
		client, err := commonDB.ConnectMongo(cfg.MongodbInfo.Address)
		if err != nil {
			glog.Fatal(err.Error())
		}
		db.SetMongo(client, cfg.MongodbInfo.DBName)
	}

	{
		tableSnapshot.SetClubEvent(&clubEvent.SelfPostEvent)
		clubEvent.SetTableSnapshotEvent(&tableSnapshot.SelfPostEvent)
		clubEvent.SetVirtualTableEvent(&virtualTable.SelfPostEvent)

		// 桌子快照
		go tableSnapshot.HandleRequest()
		// 测试桌子
		go virtualTable.InitVirtualTable()
		// 俱乐部事件
		go clubEvent.HandleClubEvent()
		// 活动定时器
		go clubEvent.ActivityTimer(int32(mengZhuClubID))
	}

	{
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGUSR1)

		go func() {
			<-sigs
			commonDef.IsRun = false
		}()
	}

	// MQ
	for commonDef.IsRun {
		var err error
		if mengZhuClubID > 0 {
			err = wrapMQ.ListenMengZhu(&cfg.RabbitMQ, cfg.ID, worker.HandleMQData)
		} else {
			err = wrapMQ.ListenClubMGR(&cfg.RabbitMQ, cfg.ID, clubMGR.HandleMQData)
		}

		if err != nil {
			glog.Error("MQ err. err:=", err.Error(), ",", mengZhuClubID)
		}
		if commonDef.IsRun == false {
			break
		}
		time.Sleep(time.Second * 3)
		glog.Error("MQ exit...")
	}

	commonDef.Wait.Wait()

	glog.Error("isRun = ", commonDef.IsRun)
}

/*
// BytesToString converts byte slice to string.
func BytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// StringToBytes converts string to byte slice.
func StringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

*/
