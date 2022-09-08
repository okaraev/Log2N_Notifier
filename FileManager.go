package main

type FileManager struct {
	Receiver         func(connectionParams interface{}) (<-chan interface{}, error)
	Delayer          func(message interface{}, connectionParams interface{}) error
	Processor        func(message interface{}, Delayer func(message interface{}) error) error
	ConnectionParams interface{}
}

func (f FileManager) StartReceive() (<-chan interface{}, error) {
	result, err := f.Receiver(f.ConnectionParams)
	return result, err
}

func (f FileManager) Delay(message interface{}) error {
	err := f.Delayer(message, f.ConnectionParams)
	return err
}

func (f FileManager) Process(message interface{}, Delayer func(message interface{}) error) error {
	err := f.Processor(message, Delayer)
	return err
}

func GetFileManagerDefaultInstance(connectionParams interface{}) FileManager {
	fm := FileManager{
		Receiver:         ReceiveMessage,
		Delayer:          DelayMessage,
		Processor:        ProcessMessage,
		ConnectionParams: connectionParams,
	}
	return fm
}

func GetFileManagerOverloadInstance(
	receiver func(connectionParams interface{}) (<-chan interface{}, error),
	delayer func(message interface{}, connectionParams interface{}) error,
	connectionParams interface{},
) FileManager {
	fm := FileManager{
		Receiver:         receiver,
		Delayer:          delayer,
		ConnectionParams: connectionParams,
	}
	return fm
}
