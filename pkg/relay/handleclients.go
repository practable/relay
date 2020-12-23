package cmd

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

func HandleClients(closed <-chan struct{}, wg *sync.WaitGroup, topics *topicDirectory, clientActionsChan chan clientAction) {

	defer func() {

		wg.Done()

		log.WithFields(log.Fields{
			"func": "HandleClients",
			"verb": "closed",
		}).Trace("HandleClients closed")

	}()

	for {
		select {
		case <-closed:
			return
		case request := <-clientActionsChan:
			if request.action == clientAdd {

				addClientToTopic(topics, request.client)

			} else if request.action == clientDelete {
				deleteClientFromTopic(topics, request.client)

			}
		}
	}
}

func addClientToTopic(topics *topicDirectory, client clientDetails) {

	_, exists := topics.directory[client.topic]

	if !exists {
		topics.Lock()
		topics.directory[client.topic] = []clientDetails{client}
		topics.Unlock()

		log.WithFields(log.Fields{
			"topic":  client.topic,
			"client": client,
			"action": clientAdd,
			"verb":   "add",
			"count":  1,
		}).Info("Added first client to new topic")

	} else {
		topics.Lock()
		topics.directory[client.topic] = append(topics.directory[client.topic], client)
		count := len(topics.directory[client.topic])
		topics.Unlock()

		log.WithFields(log.Fields{
			"topic":  client.topic,
			"client": client,
			"action": clientAdd,
			"verb":   "add",
			"count":  count,
		}).Info("Added client to existing topic")
	}
}

func deleteClientFromTopic(topics *topicDirectory, client clientDetails) {

	_, exists := topics.directory[client.topic]
	if exists {
		topics.Lock()
		existingClients := topics.directory[client.topic]
		topics.directory[client.topic] = filterClients(existingClients, client)
		count := len(topics.directory[client.topic])
		topics.Unlock()

		log.WithFields(log.Fields{
			"topic":  client.topic,
			"client": client,
			"action": clientDelete,
			"verb":   "delete",
			"count":  count,
		}).Info("Deleting client from existing topic")

	} else {

		log.WithFields(log.Fields{
			"topic":  client.topic,
			"client": client,
			"action": clientDelete,
			"verb":   "delete",
			"count":  0,
		}).Info("Ignoring: can't delete client from non-existent topic")
	}
}
