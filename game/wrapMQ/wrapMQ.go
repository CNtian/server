package wrapMQ

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/streadway/amqp"
	"qpGame/commonDefine/mateProto"
	"qpGame/commonDefine/mateProto/protoGameBasic"
	"qpGame/commonDefine/mateProto/protoInnerServer"
	"qpGame/localConfig"
	"time"
)

var (
	publisherChan *amqp.Channel
	mqConn        *amqp.Connection
	queueID       string
)

// 连接MQ
func ConnectMQ(cfg *localConfig.RabbitMQJson, queueName string) error {
	var err error

	if mqConn != nil {
		mqConn.Close()
		mqConn = nil
	}

	mqAddress := fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.IP, cfg.Port)
	mqConn, err = amqp.DialConfig(mqAddress, amqp.Config{Heartbeat: time.Second * 180})
	if err != nil {
		return err
	}

	publisherChan, err = mqConn.Channel()
	if err != nil {
		mqConn.Close()
		return err
	}

	queueID = queueName

	return err
}

// 创建游戏 MQ队列
func createGameConsume() (<-chan amqp.Delivery, error) {
	exchangeName := "game"
	queueName := queueID
	var (
		err        error
		newChannel *amqp.Channel
		queue      amqp.Queue
	)

	newChannel, err = mqConn.Channel()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			newChannel.Close()
		}
	}()

	err = newChannel.ExchangeDeclare(exchangeName, "fanout", false, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	queue, err = newChannel.QueueDeclare(queueName, false, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	err = newChannel.QueueBind(queueName, "#", exchangeName, false, nil)
	if err != nil {
		return nil, err
	}

	return newChannel.Consume(
		queue.Name, // queue
		queue.Name, // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
}

// ():本地消费者
func CreateConsumer() (<-chan amqp.Delivery, error) {

	return createGameConsume()
}

// 没有加锁,留意 msg 在多个 goroutine 中修改
func ReplyToSource(msg *mateProto.MessageMaTe, msgBody interface{}) error {
	msg.To = msg.Source
	msg.Source = localConfig.GetConfig().ID

	if msgBody != nil {
		msg.Data, _ = json.Marshal(msgBody)
	}
	data, err := json.Marshal(&msg)
	if err != nil {
		glog.Warning("json.Marshal() error. to:=", msg.Source, ",msgID:=", msg.MessageID, ",msg:=", string(data))
		return err
	}

	//commonDef.LOG_Info("to uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",data:=", string(msg.Data))

	//if msg.To == msg.Source {
	//	glog.Warning("msg.To == msg.Source  to:=", msg.To, ",source:=", msg.Source, ",messageID:=", msg.MessageID)
	//	return nil
	//}
	if msg.To == localConfig.GetConfig().ID {
		switch msg.MessageID {
		case protoGameBasic.ID_ClubJoinTable:
		default:
			glog.Warning("msg.To == msg.Source  to:=", msg.To, ",source:=", msg.Source, ",messageID:=", msg.MessageID)
		}
		return nil
	}

	err = publisherChan.Publish(
		"",
		msg.To,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	if err != nil {
		glog.Warning("send mq error. queueName:=", msg.Source)
	}
	return err
}

func SendMsg(msg *mateProto.MessageMaTe) error {

	msg.Source = localConfig.GetConfig().ID

	if msg.MsgBody != nil {
		msg.Data, _ = json.Marshal(msg.MsgBody)
	}
	data, err := json.Marshal(&msg)
	if err != nil {
		glog.Warning("json.Marshal() error. to:=", msg.Source, ",msgID:=", msg.MessageID, ",msg:=", string(data))
		return err
	}

	if msg.MessageID != protoInnerServer.ID_BroadGameServiceStatus {
		//commonDef.LOG_Info("to uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",data:=", string(msg.Data))
	}

	if msg.To == localConfig.GetConfig().ID {
		glog.Warning("msg.To == msg.Source  to:=", msg.To, ",source:=", msg.Source, ",messageID:=", msg.MessageID)
		return nil
	}

	err = publisherChan.Publish(
		"",
		msg.To,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	if err != nil {
		glog.Warning("send mq error. queueName:=", msg.Source)
	}
	return err
}

func SendMsgTo(msg *mateProto.MessageMaTe, msgBody interface{}) error {

	msg.Source = localConfig.GetConfig().ID

	if msgBody != nil {
		msg.Data, _ = json.Marshal(msgBody)
	}
	data, err := json.Marshal(&msg)
	if err != nil {
		glog.Warning("json.Marshal() error. to:=", msg.Source, ",msgID:=", msg.MessageID, ",msg:=", string(data))
		return err
	}

	if msg.MessageID != protoInnerServer.ID_BroadGameServiceStatus {
		//commonDef.LOG_Info("target :=", msg.Source, "to uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",data:=", string(msg.Data))
	}

	if msg.To == localConfig.GetConfig().ID {
		glog.Warning("msg.To == msg.Source  to:=", msg.To, ",source:=", msg.Source, ",messageID:=", msg.MessageID)
		return nil
	}

	err = publisherChan.Publish(
		"",
		msg.To,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	if err != nil {
		glog.Warning("send mq error. queueName:=", msg.Source)
	}
	return err
}

func SendMsgToClub(msg *mateProto.MessageMaTe, msgBody interface{}) error {
	to := msg.Source
	if len(msg.MZID) > 0 {
		to = msg.MZID
	}
	if msgBody != nil {
		msg.Data, _ = json.Marshal(msgBody)
	}
	data, err := json.Marshal(&msg)
	if err != nil {
		glog.Warning("json.Marshal() error. to:=", msg.Source, ",msgID:=", msg.MessageID, ",msg:=", string(data))
		return err
	}

	if msg.MessageID != protoInnerServer.ID_BroadGameServiceStatus {
		//commonDef.LOG_Info("to uid:=", msg.SenderID, ",msgID:=", msg.MessageID, ",data:=", string(msg.Data))
	}
	if msg.To == localConfig.GetConfig().ID {
		glog.Warning("msg.To == msg.Source  to:=", msg.To, ",source:=", msg.Source, ",messageID:=", msg.MessageID)
		return nil
	}

	err = publisherChan.Publish(
		"",
		to,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	if err != nil {
		glog.Warning("send mq error. queueName:=", msg.Source)
	}
	return err
}
