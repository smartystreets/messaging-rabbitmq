package rabbitmq

import (
	"log"
	"sync"

	"github.com/smartystreets/messaging/v2"
)

type TransactionWriter struct {
	mutex      *sync.Mutex
	controller Controller
	channel    Channel
	closed     bool
}

func transactionWriter(controller Controller) *TransactionWriter {
	return &TransactionWriter{
		mutex:      &sync.Mutex{},
		controller: controller,
	}
}

func (this *TransactionWriter) Write(message messaging.Dispatch) error {
	if !this.ensureChannel() {
		return messaging.ErrWriterClosed
	}

	// FUTURE: if error on publish, don't publish anything else
	// until we reset the channel during commit
	// opening a new channel is what marks it as able to continue
	dispatch := toAMQPDispatch(message, utcNow())
	return this.channel.PublishMessage(message.Destination, message.Partition, dispatch)
}

func (this *TransactionWriter) Commit() error {
	if this.channel == nil {
		return nil
	}

	err := this.channel.CommitTransaction()
	if err == nil {
		return nil
	}

	log.Println("[WARN] Transaction failed, closing channel: [", err, "]")
	_ = this.channel.Close()
	this.channel = nil
	return err
}

func (this *TransactionWriter) Close() {
	this.mutex.Lock()
	this.closed = true
	this.mutex.Unlock()
}

func (this *TransactionWriter) ensureChannel() bool {
	if this.channel != nil {
		return true
	}

	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.channel = this.controller.openChannel(this.isActive)
	if this.channel == nil {
		return false
	}

	_ = this.channel.ConfigureChannelAsTransactional()
	return true
}

func (this *TransactionWriter) isActive() bool {
	return !this.closed // must be called from within the safety of a mutex
}
