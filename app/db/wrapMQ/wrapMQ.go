package wrapMQ

import (
	"encoding/json"
	"fmt"
	"github.com/streadway/amqp"
	"time"
	"vvService/appDB/localConfig"
	"vvService/commonPackge/mateProto"
)

//type MQConnectSuccess func()

var (
	publisherChan *amqp.Channel
	mqConn        *amqp.Connection
	consumer      *amqp.Channel
)

// 连接MQ
func ListenMQData(cfg *localConfig.RabbitMQJson) error {

	if consumer != nil {
		consumer.Close()
		consumer = nil
	}

	if publisherChan != nil {
		publisherChan.Close()
		publisherChan = nil
	}

	if mqConn != nil {
		mqConn.Close()
		mqConn = nil
	}

	var err error

	mqAddress := fmt.Sprintf("amqp://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.IP, cfg.Port)
	mqConn, err = amqp.DialConfig(mqAddress, amqp.Config{Heartbeat: time.Second * 180})
	if err != nil {
		return err
	}

	publisherChan, err = mqConn.Channel()
	return err
}

// MQ队列
func CreateGameQueue(queueName string) (<-chan amqp.Delivery, error) {

	var (
		err error
	)

	consumer, err = mqConn.Channel()
	if err != nil {
		return nil, err
	}

	_, err = consumer.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	// 创建消费者
	return consumer.Consume(
		queueName, // queue
		"11111",   // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
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
