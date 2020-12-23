package crossbar

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAddDeleteClients(t *testing.T) {
	suppressLog()
	defer displayLog()
	var topics topicDirectory
	topics.directory = make(map[string][]clientDetails)

	client1 := randomClient()
	client2 := randomClient()
	client3 := randomClientForTopic(client2.topic)
	client4 := randomClientForTopic(client2.topic)

	addClientToTopic(&topics, client1)
	addClientToTopic(&topics, client2)
	addClientToTopic(&topics, client3)
	addClientToTopic(&topics, client4)

	clientList := []clientDetails{client1, client2, client3, client4}
	clientShouldExist := []bool{true, true, true, true}

	for i := range clientList {
		if clientExists(&topics, clientList[i]) != clientShouldExist[i] {
			t.Errorf("bare/addClientToTopic: client %v has WRONG existence status, should be %v\n", i, clientShouldExist[i])
		}
	}

	deleteClientFromTopic(&topics, client1)
	deleteClientFromTopic(&topics, client2)
	deleteClientFromTopic(&topics, client4)

	clientShouldExist = []bool{false, false, true, false}

	for i := range clientList {
		if clientExists(&topics, clientList[i]) != clientShouldExist[i] {
			t.Errorf("bare/deleteClientFromTopic(): client %v has WRONG existence status, should be %v\n", i, clientShouldExist[i])
		}
	}

}

func TestHandler(t *testing.T) {
	suppressLog()
	defer displayLog()
	//This test fails if you don't give the handler a chance to action the commands, hence the time.Sleep
	var wg sync.WaitGroup
	closed := make(chan struct{})
	clientActionsChan := make(chan clientAction)

	var topics topicDirectory
	topics.directory = make(map[string][]clientDetails)

	go HandleClients(closed, &wg, &topics, clientActionsChan)

	client1 := clientDetails{"client1", "topic1", make(chan message, 2)}
	client2 := clientDetails{"client2", "topic2", make(chan message, 2)}
	client3 := clientDetails{"client3", "topic2", make(chan message, 2)}
	client4 := clientDetails{"client4", "topic2", make(chan message, 2)}

	clientActionsChan <- clientAction{clientAdd, client1}
	clientActionsChan <- clientAction{clientAdd, client2}
	clientActionsChan <- clientAction{clientAdd, client3}
	clientActionsChan <- clientAction{clientAdd, client4}

	time.Sleep(1 * time.Millisecond)

	clientList := []clientDetails{client1, client2, client3, client4}
	clientShouldExist := []bool{true, true, true, true}

	for i := range clientList {
		if clientExists(&topics, clientList[i]) != clientShouldExist[i] {
			t.Errorf("handler/addClientToTopic: client %v has WRONG existence status, should be %v\n", i, clientShouldExist[i])
		}
	}

	clientActionsChan <- clientAction{clientDelete, client1}
	clientActionsChan <- clientAction{clientDelete, client2}
	clientActionsChan <- clientAction{clientDelete, client4}

	time.Sleep(1 * time.Millisecond)

	clientShouldExist = []bool{false, false, true, false}

	for i := range clientList {
		if clientExists(&topics, clientList[i]) != clientShouldExist[i] {
			t.Errorf("handler/deleteClientFromTopic(): client %v has WRONG existence status, should be %v\n", i, clientShouldExist[i])
			t.Errorf("%v", topics.directory)
		}
	}

}

func randomClient() clientDetails {
	return clientDetails{uuid.New().String(), uuid.New().String(), make(chan message)}
}

func randomClientForTopic(topic string) clientDetails {
	return clientDetails{uuid.New().String(), topic, make(chan message)}
}
