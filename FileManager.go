package main

type FileManager struct {
	Receiver  func() (<-chan interface{}, error)
	Delayer   func(message interface{}) error
	Processor func(message interface{}, Delayer func(message interface{}) error) error
}

func (f FileManager) StartReceive() (<-chan interface{}, error) {
	result, err := f.Receiver()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (f FileManager) Delay(message interface{}) error {
	err := f.Delayer(message)
	if err != nil {
		return err
	}
	return nil
}

func (f FileManager) Process(message interface{}, Delayer func(message interface{}) error) error {
	err := f.Processor(message, Delayer)
	if err != nil {
		return err
	}
	return nil
}

func GetFileManagerDefaultInstance() FileManager {
	fm := FileManager{
		Receiver:  ReceiveMessage,
		Delayer:   DelayMessage,
		Processor: ProcessMessage,
	}
	return fm
}

func GetFileManagerOverloadInstance(receiver func() (<-chan interface{}, error), delayer func(message interface{}) error) FileManager {
	fm := FileManager{
		Receiver: receiver,
		Delayer:  delayer,
	}
	return fm
}
