package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jedrw/kproximate/config"
	"github.com/jedrw/kproximate/logger"
	"github.com/jedrw/kproximate/rabbitmq"
	"github.com/jedrw/kproximate/scaler"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	kpConfig, err := config.GetKpConfig()
	if err != nil {
		logger.ErrorLog("Failed to get config", "error", err)
	}

	logger.ConfigureLogger("worker", kpConfig.Debug)
	ctx, cancel := context.WithCancel(context.Background())
	kpScaler, err := scaler.NewProxmoxScaler(ctx, kpConfig)
	if err != nil {
		logger.ErrorLog("Failed to initialise scaler", "error", err)
	}

	rabbitConfig, err := config.GetRabbitConfig()
	if err != nil {
		logger.ErrorLog("Failed to get rabbit config", "error", err)
	}

	conn, _ := rabbitmq.NewRabbitmqConnection(rabbitConfig)
	defer conn.Close()

	channel := rabbitmq.NewChannel(conn)
	defer channel.Close()
	err = channel.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		logger.FatalLog("Failed to set QoS on channel", err)
	}

	consumers := map[string]<-chan amqp.Delivery{}
	for _, queueName := range []string{
		scaler.ScaleUpQueueName,
		scaler.ScaleDownQueueName,
		scaler.ReplaceQueueName,
	} {
		err = rabbitmq.DeclareQueue(channel, queueName)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("Failed to declare %s queue", queueName), err)
		}

		consumers[queueName], err = channel.Consume(
			queueName,
			"",
			false,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("Failed to register %s consumer", queueName), err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	go func() {
		<-sigChan
		cancel()
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	logger.InfoLog("Listening for scale events")

	for {
		select {
		case scaleUpMsg := <-consumers[scaler.ScaleUpQueueName]:
			handleScaleUpMsg(ctx, kpScaler, scaleUpMsg)

		case scaleDownMsg := <-consumers[scaler.ScaleDownQueueName]:
			handleScaleDownMsg(ctx, kpScaler, scaleDownMsg)

		case replaceMsg := <-consumers[scaler.ReplaceQueueName]:
			handleReplaceMsg(ctx, kpScaler, replaceMsg)

		case <-ctx.Done():
			return
		}
	}
}

func handleScaleUpMsg(ctx context.Context, kpScaler scaler.Scaler, scaleUpMsg amqp.Delivery) {
	var scaleUpEvent *scaler.ScaleEvent
	json.Unmarshal(scaleUpMsg.Body, &scaleUpEvent)

	if scaleUpMsg.Redelivered {
		kpScaler.DeleteNode(ctx, scaleUpEvent.NodeName)
		logger.InfoLog(fmt.Sprintf("Retrying scale up event: %s", scaleUpEvent.NodeName))
	} else {
		logger.InfoLog(fmt.Sprintf("Triggered scale up event: %s", scaleUpEvent.NodeName))
	}

	err := kpScaler.ScaleUp(ctx, scaleUpEvent)
	if err != nil {
		logger.WarnLog("Scale up event failed", "error", err.Error())
		kpScaler.DeleteNode(ctx, scaleUpEvent.NodeName)
		scaleUpMsg.Reject(true)
		return
	}

	scaleUpMsg.Ack(false)
}

func handleScaleDownMsg(ctx context.Context, kpScaler scaler.Scaler, scaleDownMsg amqp.Delivery) {
	var scaleDownEvent *scaler.ScaleEvent
	json.Unmarshal(scaleDownMsg.Body, &scaleDownEvent)

	if scaleDownMsg.Redelivered {
		logger.InfoLog(fmt.Sprintf("Retrying scale down event: %s", scaleDownEvent.NodeName))
	} else {
		logger.InfoLog(fmt.Sprintf("Triggered scale down event: %s", scaleDownEvent.NodeName))
	}

	scaleCtx, scaleCancel := context.WithDeadline(ctx, time.Now().Add(time.Second*300))
	defer scaleCancel()

	err := kpScaler.ScaleDown(scaleCtx, scaleDownEvent)
	if err != nil {
		logger.WarnLog(fmt.Sprintf("Scale down event failed: %s", err.Error()))
		scaleDownMsg.Reject(true)
		return
	}

	logger.InfoLog(fmt.Sprintf("Deleted %s", scaleDownEvent.NodeName))
	scaleDownMsg.Ack(false)
}

func handleReplaceMsg(ctx context.Context, kpScaler scaler.Scaler, replaceMsg amqp.Delivery) {
	var replaceEvent *scaler.ScaleEvent
	json.Unmarshal(replaceMsg.Body, &replaceEvent)

	if replaceMsg.Redelivered {
		logger.InfoLog(fmt.Sprintf("Retrying replace event: %s", replaceEvent.NodeName))
	} else {
		logger.InfoLog(fmt.Sprintf("Triggered replace event: %s", replaceEvent.NodeName))
	}

	newNodeName, err := kpScaler.ReplaceNode(ctx, replaceEvent)
	if err != nil {
		logger.WarnLog("Replace event failed", "error", err.Error())
		kpScaler.DeleteNode(ctx, newNodeName)
		replaceMsg.Reject(true)
		return
	}

	replaceMsg.Ack(false)
}
