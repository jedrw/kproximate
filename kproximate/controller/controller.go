package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jedrw/kproximate/config"
	"github.com/jedrw/kproximate/logger"
	"github.com/jedrw/kproximate/metrics"
	"github.com/jedrw/kproximate/rabbitmq"
	"github.com/jedrw/kproximate/scaler"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	kpConfig, err := config.GetKpConfig()
	if err != nil {
		logger.FatalLog("Failed to get config", err)
	}

	logger.ConfigureLogger("controller", kpConfig.Debug)
	ctx, cancel := context.WithCancel(context.Background())
	kpScaler, err := scaler.NewProxmoxScaler(ctx, kpConfig)
	if err != nil {
		logger.FatalLog("Failed to initialise scaler", err)
	}

	rabbitConfig, err := config.GetRabbitConfig()
	if err != nil {
		logger.FatalLog("Failed to get rabbit config", err)
	}

	conn, mgmtClient := rabbitmq.NewRabbitmqConnection(rabbitConfig)
	defer conn.Close()

	channel := rabbitmq.NewChannel(conn)
	defer channel.Close()
	for _, queueName := range []string{scaler.ScaleUpQueueName, scaler.ScaleDownQueueName, scaler.ReplaceQueueName} {
		err = rabbitmq.DeclareQueue(channel, queueName)
		if err != nil {
			logger.FatalLog(fmt.Sprintf("Failed to declare %s queue", queueName), err)
		}
	}

	sigChan := make(chan os.Signal, 1)
	go func() {
		<-sigChan
		cancel()
	}()

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go metrics.Serve(ctx, kpScaler, kpConfig)
	logger.InfoLog("Started")
	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(time.Second * time.Duration(kpConfig.PollInterval))

			assessScaleUp(ctx, kpScaler, kpConfig.MaxKpNodes, rabbitConfig, channel, mgmtClient)
			assessScaleDown(ctx, kpScaler, rabbitConfig, channel, mgmtClient)
			assessNodeDrift(ctx, kpScaler, rabbitConfig, channel, mgmtClient)
		}
	}

}

func assessScaleUp(
	ctx context.Context,
	kpScaler scaler.Scaler,
	maxKpNodes int,
	rabbitConfig config.RabbitConfig,
	channel *amqp.Channel,
	mgmtClient *http.Client,
) {

	logger.DebugLog("Assessing for scale up")
	allScaleEvents, err := countScalingEvents(
		[]string{scaler.ScaleUpQueueName},
		channel,
		mgmtClient,
		rabbitConfig,
	)
	if err != nil {
		logger.FatalLog("Failed to count scaling events", err)
	}

	numKpNodes, err := kpScaler.NumReadyNodes()
	if err != nil {
		logger.FatalLog("Failed to get kproximate nodes", err)
	}

	if numKpNodes+allScaleEvents < maxKpNodes {
		logger.DebugLog("Calculating required scale events")
		scaleUpEvents, err := kpScaler.RequiredScaleUpEvents(allScaleEvents)
		if err != nil {
			logger.FatalLog("Failed to calculate required scale events", err)
		}

		if len(scaleUpEvents) > 0 {
			maxScaleEvents := maxKpNodes - (numKpNodes + allScaleEvents)
			numScaleEvents := min(maxScaleEvents, len(scaleUpEvents))
			scaleUpEvents = scaleUpEvents[0:numScaleEvents]
			logger.DebugLog("Selecting target hosts")
			err = kpScaler.SelectTargetHosts(ctx, scaleUpEvents)
			if err != nil {
				logger.FatalLog("Failed to select target host", err)
			}
		} else {
			logger.DebugLog("No scale up events required")
		}

		for _, scaleUpEvent := range scaleUpEvents {
			logger.DebugLog("Generated scale event", "scaleEvent", fmt.Sprintf("%+v", scaleUpEvent))
			err = queueScaleEvent(ctx, scaleUpEvent, channel, scaler.ScaleUpQueueName)
			if err != nil {
				logger.ErrorLog("Failed to queue scale up event", err)
			}

			logger.InfoLog(fmt.Sprintf("Requested scale up event: %s", scaleUpEvent.NodeName))

			time.Sleep(time.Second * 1)
		}
	} else {
		logger.DebugLog("Cannot scale up, reached maxKpNodes")
	}

}

func assessScaleDown(
	ctx context.Context,
	kpScaler scaler.Scaler,
	rabbitConfig config.RabbitConfig,
	channel *amqp.Channel,
	mgmtClient *http.Client,
) {
	logger.DebugLog("Assessing for scale down")
	allScaleEvents, err := countScalingEvents(
		[]string{
			scaler.ScaleUpQueueName,
			scaler.ScaleDownQueueName,
			scaler.ReplaceQueueName,
		},
		channel,
		mgmtClient,
		rabbitConfig,
	)
	if err != nil {
		logger.FatalLog("Failed to count scale events", err)
	}

	numKpNodes, err := kpScaler.NumReadyNodes()
	if err != nil {
		logger.FatalLog("Failed to get kproximate nodes", err)
	}

	if allScaleEvents == 0 && numKpNodes > 0 {
		logger.DebugLog("Calculating required scale events")
		scaleDownEvent, err := kpScaler.AssessScaleDown()
		if err != nil {
			logger.ErrorLog(fmt.Sprintf("Failed to assess scale down: %s", err))
		}
		if scaleDownEvent != nil {
			err = queueScaleEvent(ctx, scaleDownEvent, channel, scaler.ScaleDownQueueName)
			if err != nil {
				logger.ErrorLog(fmt.Sprintf("Failed to queue scale down event: %s", err))
			}

			logger.InfoLog(fmt.Sprintf("Requested scale down event: %s", scaleDownEvent.NodeName))
		} else {
			logger.DebugLog("No scale down events required")
		}
	} else {
		logger.DebugLog("Cannot scale down, scale event in progress or 0 kpNodes in cluster")
	}
}

func assessNodeDrift(
	ctx context.Context,
	kpScaler scaler.Scaler,
	rabbitConfig config.RabbitConfig,
	channel *amqp.Channel,
	mgmtClient *http.Client,
) {
	logger.DebugLog("Assessing for node drift")
	allScaleEvents, err := countScalingEvents(
		[]string{
			scaler.ScaleUpQueueName,
			scaler.ScaleDownQueueName,
			scaler.ReplaceQueueName,
		},
		channel,
		mgmtClient,
		rabbitConfig,
	)
	if err != nil {
		logger.FatalLog("Failed to count scale events", err)
	}

	if allScaleEvents == 0 {
		logger.DebugLog("Calculating required replacement events")
		nodeDriftEvent, err := kpScaler.AssessNodeDrift()
		if err != nil {
			logger.ErrorLog(fmt.Sprintf("Failed to assess node drift: %s", err))
		}
		if nodeDriftEvent != nil {
			err = queueScaleEvent(ctx, nodeDriftEvent, channel, scaler.ReplaceQueueName)
			if err != nil {
				logger.ErrorLog(fmt.Sprintf("Failed to queue replace event: %s", err))
			}

			logger.InfoLog(fmt.Sprintf("Requested replace event: %s", nodeDriftEvent.NodeName))
		} else {
			logger.DebugLog("No replace events required")
		}
	} else {
		logger.DebugLog("Cannot assess for node drift, scale event in progress")
	}
}

func countScalingEvents(
	queueNames []string,
	channel *amqp.Channel,
	mgmtClient *http.Client,
	rabbitConfig config.RabbitConfig,
) (int, error) {
	numScalingEvents := 0

	for _, queueName := range queueNames {
		pendingScaleEvents, err := rabbitmq.GetPendingScaleEvents(channel, queueName)
		if err != nil {
			return numScalingEvents, err
		}

		numScalingEvents += pendingScaleEvents

		runningScaleEvents, err := rabbitmq.GetRunningScaleEvents(mgmtClient, rabbitConfig, queueName)
		if err != nil {
			return numScalingEvents, err
		}

		numScalingEvents += runningScaleEvents
	}

	return numScalingEvents, nil
}

func queueScaleEvent(ctx context.Context, scaleEvent *scaler.ScaleEvent, channel *amqp.Channel, queueName string) error {
	msg, err := json.Marshal(scaleEvent)
	if err != nil {
		return err
	}

	queueCtx, queueCancel := context.WithTimeout(ctx, 5*time.Second)
	defer queueCancel()
	return channel.PublishWithContext(
		queueCtx,
		"",
		queueName,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         []byte(msg),
		})
}
