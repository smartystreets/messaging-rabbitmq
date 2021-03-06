package rabbitmq

import (
	"errors"
	"testing"

	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
	"github.com/smartystreets/messaging/v2"
)

func TestTransactionWriterFixture(t *testing.T) {
	gunit.Run(new(TransactionWriterFixture), t)
}

type TransactionWriterFixture struct {
	*gunit.Fixture

	writer     *TransactionWriter
	controller *FakeWriterController
}

func (this *TransactionWriterFixture) Setup() {
	this.controller = newFakeWriterController()
	this.writer = transactionWriter(this.controller)
}

///////////////////////////////////////////////////////////////

func (this *TransactionWriterFixture) TestDispatchIsWrittenToChannel() {
	dispatch := messaging.Dispatch{
		Destination: "destination",
		Partition:   "partition",
		Payload:     []byte{1, 2, 3, 4, 5},
	}

	err := this.writer.Write(dispatch)

	this.So(err, should.BeNil)
	this.So(this.controller.channel.exchange, should.Equal, dispatch.Destination)
	this.So(this.controller.channel.partition, should.Equal, dispatch.Partition)
	this.So(this.controller.channel.dispatch.Body, should.Resemble, dispatch.Payload)
	this.So(this.controller.channel.transactional, should.BeTrue)
}

///////////////////////////////////////////////////////////////

func (this *TransactionWriterFixture) TestChannelCannotBeObtained() {
	this.controller.channel = nil

	err := this.writer.Write(messaging.Dispatch{})

	this.So(err, should.NotBeNil)
}

///////////////////////////////////////////////////////////////

func (this *TransactionWriterFixture) TestFailedChannelNOTClosedOnFailedWrites() {
	this.controller.channel.err = errors.New("channel failed")

	err := this.writer.Write(messaging.Dispatch{})

	this.So(err, should.Equal, this.controller.channel.err)
	this.So(this.controller.channel.closed, should.Equal, 0)
	this.So(this.writer.channel, should.NotBeNil)
}

///////////////////////////////////////////////////////////////

func (this *TransactionWriterFixture) TestCloseWriter() {
	this.writer.Close()

	this.So(this.writer.closed, should.BeTrue)
	this.So(this.writer.Write(messaging.Dispatch{}), should.Equal, messaging.ErrWriterClosed)
}

///////////////////////////////////////////////////////////////

func (this *TransactionWriterFixture) TestCommitWithoutIsNoop() {
	err := this.writer.Commit()

	this.So(err, should.BeNil)
	this.So(this.controller.channel.commits, should.Equal, 0)
}

func (this *TransactionWriterFixture) TestCommitCallsUnderlyingChannel() {
	_ = this.writer.Write(messaging.Dispatch{})

	err := this.writer.Commit()
	this.So(err, should.BeNil)
	this.So(this.controller.channel.commits, should.Equal, 1)
}

func (this *TransactionWriterFixture) TestFailedCommitsReturnError() {
	_ = this.writer.Write(messaging.Dispatch{})
	this.controller.channel.err = errors.New("Commit failure")

	err := this.writer.Commit()
	this.So(err, should.Equal, this.controller.channel.err)
	this.So(this.controller.channel.commits, should.Equal, 1)
	this.So(this.controller.channel.closed, should.Equal, 1)
	this.So(this.writer.channel, should.BeNil)
}
