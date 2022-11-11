package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/streadway/amqp"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"vvService/appDB/db"
	"vvService/appDB/localConfig"
	"vvService/appDB/protoDefine"
	"vvService/appDB/worker"
	"vvService/appDB/wrapMQ"
	commonDef "vvService/commonPackge"
	commonDB "vvService/commonPackge/db"
	"vvService/commonPackge/mateProto"
)

func main() {

	{
		pid := os.Getpid()
		ioutil.WriteFile("./cur.pid", []byte(strconv.Itoa(pid)), 0666)
	}

	flag.Parse()
	defer glog.Flush()

	var cfg *localConfig.LocalConfig

	{
		var err error
		cfg, err = localConfig.LoadConfig("./config.json")
		if err != nil {
			glog.Fatal(err.Error())
			return
		}
	}

	// Redis
	{
		redisClient, err := db.ConnectRedis(cfg.Redis.IP, cfg.Redis.Password, cfg.Redis.Port, cfg.Redis.LoginIndex)
		if err != nil {
			glog.Fatal(err.Error())
		}
		db.PersonalRedisClient = redisClient

		redisClient, err = db.ConnectRedis(cfg.Redis.IP, cfg.Redis.Password, cfg.Redis.Port, cfg.Redis.GameIndex)
		if err != nil {
			glog.Fatal(err.Error())
		}
		db.GameRedisClient = redisClient
	}

	// Mongodb
	{
		client, err := commonDB.ConnectMongo(cfg.MongodbInfo.Address)
		if err != nil {
			glog.Fatal(err.Error())
		}
		db.SetMongo(client, cfg.MongodbInfo.DBName)
	}
	//db.PutTongZhuItem(20220823, []int64{556677, 1234567}, primitive.NewObjectID())

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
		err := wrapMQ.ListenMQData(&cfg.RabbitMQ)
		if err != nil {
			glog.Error("ListenMQData() err. err:=", err.Error())
			time.Sleep(time.Second * 3)
			continue
		}
		var mqChan <-chan amqp.Delivery
		mqChan, err = wrapMQ.CreateGameQueue(cfg.ID)
		if err != nil {
			glog.Error("CreateGameQueue() err. err:=", err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		year, month, day := time.Now().Date()
		lastDay, _ := strconv.Atoi(fmt.Sprintf("%d%02d%02d", year, month, day))
		timer17Second := time.NewTimer(time.Second * 17)

		worker.DailyTotalDay = day

		timer1Hour := time.NewTimer(time.Hour)
		lastCleanDay := day

		for commonDef.IsRun {
			select {
			case mqMsg := <-mqChan: // 处理MQ数据
				worker.HandleMQData(&mqMsg.Body)
				err = mqMsg.Ack(false)
				if err != nil {
					glog.Warning("Ack() err. err:=", err.Error())
				}
			case <-timer17Second.C:
				now_ := time.Now()
				curYear, curMonth, curDay := now_.Date()
				curHour := now_.Hour()

				if worker.DailyTotalDay != curDay {
					msg := mateProto.MessageMaTe{MessageID: protoDefine.ID_WriteDaily}
					msgData, _ := json.Marshal(&msg)
					worker.HandleMQData(&msgData)

					worker.DailyTotalDay = curDay
				}

				if curDay != day && curHour == 6 {
					//if true {
					msg := mateProto.MessageMaTe{MessageID: protoDefine.ID_TotalMangeFee}
					msg.Data, _ = json.Marshal(protoDefine.SS_TotalMangeFee{Date: lastDay})
					msgData, _ := json.Marshal(&msg)

					worker.HandleMQData(&msgData)

					lastDay, _ = strconv.Atoi(fmt.Sprintf("%d%02d%02d", curYear, curMonth, curDay))
					day = curDay
				}

				timer17Second = time.NewTimer(time.Second * 17)

			case <-timer1Hour.C:
				nowTT := time.Now()
				_, _, curDay := nowTT.Date()
				curHour := nowTT.Hour()

				if curDay != lastCleanDay && curHour == 5 {
					msg := mateProto.MessageMaTe{MessageID: protoDefine.ID_DeleteExpiredData}
					msgData, _ := json.Marshal(&msg)
					worker.HandleMQData(&msgData)

					lastCleanDay = curDay
				}
				timer1Hour = time.NewTimer(time.Hour)
			}
		} // for

		if commonDef.IsRun == false {
			break
		} else {
			time.Sleep(time.Second * 3)
			glog.Error("MQ exit...")
		}
	}

	glog.Error("exit...", commonDef.IsRun)
}
