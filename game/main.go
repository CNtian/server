package main

import (
	"flag"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
	"github.com/streadway/amqp"
	"io/ioutil"
	"math/rand"
	_ "net/http/pprof"
	"os"
	commonDef "qpGame/commonDefine"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/db"
	"qpGame/game/tableFactory"
	"qpGame/localConfig"
	"qpGame/management"
	"qpGame/wrapMQ"
	"strconv"
	"time"
)

func main() {
	{
		serviceStatus := ""
		flag.StringVar(&serviceStatus, "s", "0", "服务器启动状态")
		flag.Parse()

		glog.Warning("service default status := ", serviceStatus)

		defStatus, _ := strconv.Atoi(serviceStatus)
		management.SetServiceStatus(int32(defStatus))

		pid := os.Getpid()
		ioutil.WriteFile("./cur.pid", []byte(strconv.Itoa(pid)), 0666)
	}

	// 数字越小等级越高
	glog.Info("Log 数字越小等级越高. 1-warning 2-info")
	commonDef.Warning = glog.V(1)
	commonDef.Info = glog.V(2)

	rand.Seed(time.Now().UnixNano())

	flag.Parse()
	defer glog.Flush()

	var (
		err         error
		cfg         *localConfig.LocalConfig
		redisClient *redis.Client
		reboot      = true
	)

	cfg, err = localConfig.LoadConfig("./config.json")
	if err != nil {
		glog.Fatal(err.Error())
		return
	}

	for _, v := range cfg.SupportPlaying {
		if tableFactory.IsSupport(v.PlayingID) == false {
			glog.Fatalf("this module not support it. playID.%d. support these:=%v",
				v.PlayingID, tableFactory.GetSupport())
			return
		}
	}

	err = management.Init(&cfg.TableNumRange)
	if err != nil {
		glog.Fatal(err.Error())
	}

	// redis
	{
		redisClient, err = db.ConnectRedis(cfg.Redis.IP, cfg.Redis.Password, cfg.Redis.Port, cfg.Redis.LoginIndex)
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
		err := db.ConnectMongoDB(cfg.MongodbInfo.Address, cfg.MongodbInfo.DBName)
		if err != nil {
			glog.Fatal(err.Error())
		}
	}
	/*
		// 避免 重复 游戏的queue 名称
		{
			var (
				resp   *http.Response
				body   []byte
				basic  string
				url    = fmt.Sprintf("http://%s:%d/api/queues", cfg.RabbitMQ.IP, 15672)
				client = &http.Client{}
			)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				glog.Fatal(err.Error())
			}
			req.Header.Add("content-type", `application/json`)

			//Basic
			basic = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.RabbitMQ.User, cfg.RabbitMQ.Password)))
			req.Header.Add("Authorization", "Basic "+basic)

			resp, err = client.Do(req)
			if err != nil {
				glog.Fatal(err.Error())
			}

			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				glog.Fatal(err.Error())
			}
			resp.Body.Close()

			queueMap := make([]map[string]interface{}, 0)
			err = json.Unmarshal(body, &queueMap)
			if err != nil {
				glog.Fatal(err.Error())
			}
			for _, v := range queueMap {
				if keyValue, ok := v["name"]; ok == true {
					if keyValue.(string) == cfg.ID {
						glog.Fatal("repeat queueID. queueID:=" + cfg.ID)
					}
				}
			}
		}
	*/
	//{
	//	go func() {
	//		glog.Fatal(http.ListenAndServe(cfg.PprofPort, nil))
	//	}()
	//	time.Sleep(time.Second)
	//}

	for {
		var (
			localCh <-chan amqp.Delivery
			ok      = true
			chanMsg amqp.Delivery
		)

		err = wrapMQ.ConnectMQ(&cfg.RabbitMQ, cfg.ID)
		if err != nil {
			glog.Error(err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		localCh, err = wrapMQ.CreateConsumer()
		if err != nil {
			glog.Error(err.Error())
			time.Sleep(time.Second * 3)
			continue
		}

		management.LoginToHallService("hall")
		management.LoginToClubService(&mateProto.MessageMaTe{Source:"club"})

		if reboot == true {
			msg := mateProto.MessageMaTe{Source: localConfig.GetConfig().ID,
				To:        "club",
				MessageID: protoGameBasic.ID_DeleteServiceIDTable}

			err = wrapMQ.SendMsgTo(&msg,
				&protoGameBasic.SS_DeleteServiceIDTable{ServiceID: localConfig.GetConfig().ID})
			if err != nil {
				glog.Error("LoginToOtherService() queueName：=", "club", ", err:=", err.Error())
			}
			reboot = false
		}

		timerChan := time.NewTimer(time.Minute)

		for ok {
			select {
			case chanMsg, ok = <-localCh:
				if ok == false {
					break
				}
				management.HandleMQData(chanMsg.Body)
			case <-timerChan.C:
				management.BroadGameServiceStatus()
				timerChan = time.NewTimer(time.Minute)
			}
		}
		glog.Error("mq exit.")
		time.Sleep(time.Second * 3)
	}
}

/*
	{
		b := qpTable.BroadcastPlayerStatus{UID: 111, SeatNum: 9, Status: 99}

		a := qpTable.QPProtocol{}

		any, err := anypb.New(&b)
		if err != nil {
			fmt.Println(err.Error())
		}
		a.MsgBody = any

		var dataArr []byte
		dataArr, err = proto.Marshal(&a)
		if err != nil {
			fmt.Println(err.Error())
		}

		c := qpTable.QPProtocol{}
		err = proto.Unmarshal(dataArr, &c)
		if err != nil {
			fmt.Println(err.Error())
		}

		d := qpTable.BroadcastPlayerStatus{}
		err = c.MsgBody.UnmarshalTo(&d)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Println(a.String())
		return
	}
*/
