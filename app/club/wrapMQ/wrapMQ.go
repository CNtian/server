package wrapMQ

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/streadway/amqp"
	"time"
	"vvService/appClub/localConfig"
	commonDef "vvService/commonPackge"
	"vvService/commonPackge/mateProto"
)

type handleMQFunc func([]byte)
type MQConnectSuccess func()

const exchangeName = "mzClub"

var (
	publisherChan *amqp.Channel
	mqConn        *amqp.Connection
)

// 连接MQ
func ListenClubMGR(cfg *localConfig.RabbitMQJson, queueID string, mqFunc handleMQFunc) (err error) {

	mqAddress := fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.IP, cfg.Port)
	mqConn, err = amqp.DialConfig(mqAddress, amqp.Config{Heartbeat: time.Second * 180})
	if err != nil {
		return err
	}
	defer func() {
		mqConn.Close()
		mqConn = nil
	}()

	publisherChan, err = mqConn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		publisherChan.Close()
		publisherChan = nil
	}()

	return createClubMGRQueue(mqConn, queueID, mqFunc)
}

// MQ队列
func createClubMGRQueue(conn *amqp.Connection, queueName string, handleMQ handleMQFunc) (err error) {

	var (
		consumer *amqp.Channel
		msgs     <-chan amqp.Delivery
	)

	consumer, err = conn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		consumer.Close()
	}()

	err = consumer.ExchangeDeclare(exchangeName, "fanout", false, true, false, false, nil)
	if err != nil {
		return err
	}

	_, err = consumer.QueueDeclare(queueName, false, true, false, false, nil)
	if err != nil {
		return err
	}

	// 创建消费者
	msgs, err = consumer.Consume(
		queueName, // queue
		queueName, // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	err = SendToExchange(&mateProto.MessageMaTe{
		Source:    localConfig.GetConfig().ID,
		MessageID: mateProto.ID_NoticeClubMGRLaunch}, "mzClub")
	if err != nil {
		glog.Warning("SendToMengZhuClubExchange() err. ", err.Error())
	}

	// 处理MQ数据
	for v := range msgs {
		handleMQ(v.Body)
		if commonDef.IsRun == false {
			break
		}
	}
	return
}

func ListenMengZhu(cfg *localConfig.RabbitMQJson, queueID string, mqFunc handleMQFunc) (err error) {
	mqAddress := fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.IP, cfg.Port)
	mqConn, err = amqp.DialConfig(mqAddress, amqp.Config{Heartbeat: time.Second * 180})
	if err != nil {
		return err
	}
	defer func() {
		mqConn.Close()
		mqConn = nil
	}()

	publisherChan, err = mqConn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		publisherChan.Close()
		publisherChan = nil
	}()

	return createMengZhuQueue(mqConn, queueID, mqFunc)
}

func createMengZhuQueue(conn *amqp.Connection, queueName string, handleMQ handleMQFunc) (err error) {
	var (
		consumer *amqp.Channel
		msgs     <-chan amqp.Delivery
	)

	consumer, err = conn.Channel()
	if err != nil {
		return err
	}
	defer func() {
		consumer.Close()
	}()

	_, err = consumer.QueueDeclare(queueName, false, true, false, false, nil)
	if err != nil {
		return err
	}

	err = consumer.QueueBind(queueName, "#", exchangeName, false, nil)
	if err != nil {
		return err
	}

	// 创建消费者
	msgs, err = consumer.Consume(
		queueName, // queue
		queueName, // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	err = SendToExchange(&mateProto.MessageMaTe{
		Source:    localConfig.GetConfig().ID,
		MessageID: mateProto.ID_ClubServiceLaunch,
		MZID:      localConfig.GetConfig().ID,
	}, "game")
	if err != nil {
		glog.Warning("BroadcastClubLaunch to game. err:=", err.Error())
	}

	err = PublishProto("club", &mateProto.MessageMaTe{
		Source:    localConfig.GetConfig().ID,
		MessageID: mateProto.ID_NoticeClubMGRLaunch,
		MZID:      localConfig.GetConfig().ID})
	if err != nil {
		glog.Warning("BroadcastClubLaunch to club. err:=", err.Error())
	}

	for v := range msgs {
		handleMQ(v.Body)

		if commonDef.IsRun == false {
			break
		}
	}

	return nil
}

// 发布数据
func PublishProto(queueName string, msg *mateProto.MessageMaTe) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = publisherChan.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	return err
}

// 发布数据
func SendToSource(msg *mateProto.MessageMaTe, msgBody interface{}) error {
	if msgBody != nil {
		msg.Data, _ = json.Marshal(msgBody)
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	err = publisherChan.Publish(
		"",
		msg.Source,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	return err
}

func ForwardTo(queueName string, data *[]byte) error {

	err := publisherChan.Publish(
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        *data,
		},
	)
	return err
}

func SendToExchange(msg *mateProto.MessageMaTe, name string) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return publisherChan.Publish(name, "#", false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        data,
	})
}
