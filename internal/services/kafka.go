package services

import (
	"fmt"

	"github.com/ArtemVoronov/indefinite-studies-utils/pkg/log"
)

func SendMessageToKafkaQueue(queueTopic string, message string) {
	err := Instance().KafkaProducer().CreateMessage(queueTopic, message)
	if err != nil {
		// TODO: create some daemon that catch unpublished posts
		log.Error(fmt.Sprintf("Unable to put message '%v' into queue '%v'", message, queueTopic), err.Error())
	}
}
