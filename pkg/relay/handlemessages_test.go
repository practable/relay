package cmd

/*
func TestHandleMessages(t *testing.T) {

	var wg sync.WaitGroup
	closed := make(chan struct{})
	messagesChan := make(chan message, 10)
	clientActionsChan := make(chan clientAction)

	//setup client handling
	var topics topicDirectory
	topics.directory = make(map[string][]clientDetails)

	go HandleClients(closed, &wg, &topics, clientActionsChan)
	go HandleMessages(closed, &wg, &topics, messagesChan)

	//make some clients
	client1Chan := make(chan message, 2)
	client2Chan := make(chan message, 2)
	client3Chan := make(chan message, 2)
	client4Chan := make(chan message, 2)

	client1 := clientDetails{"client1", "/topic1", client1Chan}
	client2 := clientDetails{"client2", "/topic2", client2Chan}
	client3 := clientDetails{"client3", "/topic2", client3Chan}
	client4 := clientDetails{"client4", "/topic2", client4Chan}

	clientActionsChan <- clientAction{clientAdd, client1}
	clientActionsChan <- clientAction{clientAdd, client2}
	clientActionsChan <- clientAction{clientAdd, client3}
	clientActionsChan <- clientAction{clientAdd, client4}

	//let the clients be processed
	time.Sleep(1 * time.Millisecond)

	//check clients exist before proceeding
	clientList := []clientDetails{client1, client2, client3, client4}
	clientShouldExist := []bool{true, true, true, true}

	for i := range clientList {
		if clientExists(&topics, clientList[i]) != clientShouldExist[i] {
			t.Errorf("HandleMessages/addClientToTopic: client %v has WRONG existence status, should be %v\n", i, clientShouldExist[i])
		}
	}

	//make test messages
	b1 := []byte{'c', 'r', 'o', 's', 's'}
	var testMessage1 = message{sender: client1, mt: websocket.BinaryMessage, data: b1}
	b2 := []byte{'b', 'a', 'r'}
	var testMessage2 = message{sender: client3, mt: websocket.BinaryMessage, data: b2}

	// send some messages on behalf the clients
	messagesChan <- testMessage1
	messagesChan <- testMessage2

	//let the messages be processed
	time.Sleep(2 * time.Millisecond)

	//check who got what ...
	msg, err := read(client1Chan, 1*time.Millisecond)
	if err == nil {
		t.Errorf("Client 1 should have got an timeout but got %v,%v", msg, err)
	}

	msg, err = read(client2Chan, 1*time.Millisecond)
	if err != nil && !bytes.Equal(msg.data, b2) {
		t.Errorf("Client 2 should have got msg but got %v,%v", msg, err)
	}
	msg, err = read(client3Chan, 1*time.Millisecond)
	if err == nil {
		t.Errorf("Client 3 should have got an timeout but got %v,%v", msg, err)
	}
	msg, err = read(client4Chan, 1*time.Millisecond)
	if err != nil && !bytes.Equal(msg.data, b2) {
		t.Errorf("Client 4 should have got msg but got %v, %v", msg, err)
	}

	// delete a client and see what happens
	clientActionsChan <- clientAction{clientDelete, client4}

	time.Sleep(1 * time.Millisecond)

	//send a msg but this time from client 2 - should only go to client 3
	//swapping clients like this detected that sending clients were being unsubscribed
	//so keep the test like this (sending to a previous sender)
	b3 := []byte{'f', 'o', 'o'}
	var testMessage3 = message{sender: client2, mt: websocket.BinaryMessage, data: b2}

	messagesChan <- testMessage3

	time.Sleep(1 * time.Millisecond)

	msg, err = read(client2Chan, 1*time.Millisecond)
	if err == nil {
		t.Errorf("Client 2 should have got an timeout but got %v,%v", msg, err)
	}

	msg, err = read(client3Chan, 1*time.Millisecond)
	if err != nil && !bytes.Equal(msg.data, b3) {
		t.Errorf("Client 3 should have got msg but got %v,%v,%v", msg, err, topics.directory)
	}

	msg, err = read(client4Chan, 1*time.Millisecond)
	if err == nil {
		t.Errorf("Client 4 should have got an timeout but got %v,%v", msg, err)
	}
}

func read(messageChannel chan message, timeout time.Duration) (message, error) {

	select {
	case msg := <-messageChannel:
		return msg, nil
	case <-time.After(timeout):
		return message{}, errors.New("timeout reading from channel")
	}

}
*/
