package kinesis

import (
	"context"
	"testing"
	"time"

	"gitlab.com/marcoxavier/go-kinesis/mocks"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

func TestRunner_Closed(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
	}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("", errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(checkpoint.AssertExpectations(t)).To(BeTrue(), "Should try to get last sequence")
}

func TestRunner_Process_FailsToGetLastCheckpoint(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
	}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("", errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(checkpoint.AssertExpectations(t)).To(BeTrue(), "Should try to get last sequence")
}

func TestRunner_Process_FailsGettingShardIterator(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(nil, errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get shard iterator")
}

func TestRunner_Process_FailsGettingRecords(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	getShardIteratorOutput := &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("some_shard_iterator")}
	getRecordsInput := &kinesis.GetRecordsInput{
		ShardIterator: getShardIteratorOutput.ShardIterator,
	}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(getShardIteratorOutput, nil)
	kinesisAPI.On("GetRecordsWithContext", ctx, getRecordsInput).Return(nil, errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get records")
}

func TestRunner_Process_ShardClosedDoNothing(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	closed := false
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
		shutdown: func() {
			closed = true
		},
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	getShardIteratorOutput := &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("some_shard_iterator")}
	getRecordsInput := &kinesis.GetRecordsInput{
		ShardIterator: getShardIteratorOutput.ShardIterator,
	}
	getRecordsOutput := &kinesis.GetRecordsOutput{NextShardIterator: nil}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(getShardIteratorOutput, nil)
	kinesisAPI.On("GetRecordsWithContext", ctx, getRecordsInput).Return(getRecordsOutput, nil)

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(closed).To(BeTrue())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get records")
}

func TestRunner_Process_FailsHandleRecord(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
		handler:            func(Message) error { return errors.New("something failed") },
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	getShardIteratorOutput := &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("some_shard_iterator")}
	getRecordsInput := &kinesis.GetRecordsInput{
		ShardIterator: getShardIteratorOutput.ShardIterator,
	}
	record := &kinesis.Record{PartitionKey: aws.String("some_partition"), Data: []byte("some_data"), SequenceNumber: aws.String("some_sequence_number2")}
	getRecordsOutput := &kinesis.GetRecordsOutput{NextShardIterator: aws.String("some_shard_iterator"), Records: []*kinesis.Record{record}}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(getShardIteratorOutput, nil)
	kinesisAPI.On("GetRecordsWithContext", ctx, getRecordsInput).Return(getRecordsOutput, nil)
	checkpoint.On("Set", r.checkpointIdentifier(), "some_sequence_number").Return(errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get records")
}

func TestRunner_Process_PanicsHandleRecord(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
		handler: func(Message) error {
			panic("something failed")
		},
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	getShardIteratorOutput := &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("some_shard_iterator")}
	getRecordsInput := &kinesis.GetRecordsInput{
		ShardIterator: getShardIteratorOutput.ShardIterator,
	}
	record := &kinesis.Record{PartitionKey: aws.String("some_partition"), Data: []byte("some_data"), SequenceNumber: aws.String("some_sequence_number2")}
	getRecordsOutput := &kinesis.GetRecordsOutput{NextShardIterator: aws.String("some_shard_iterator"), Records: []*kinesis.Record{record}}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(getShardIteratorOutput, nil)
	kinesisAPI.On("GetRecordsWithContext", ctx, getRecordsInput).Return(getRecordsOutput, nil)
	checkpoint.On("Set", r.checkpointIdentifier(), "some_sequence_number").Return(errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get records")
}

func TestRunner_Process_HandlesWithSuccess(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecordBatch,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
		handler:            func(Message) error { return nil },
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	getShardIteratorOutput := &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("some_shard_iterator")}
	getRecordsInput := &kinesis.GetRecordsInput{
		ShardIterator: getShardIteratorOutput.ShardIterator,
	}
	record := &kinesis.Record{PartitionKey: aws.String("some_partition"), Data: []byte("some_data"), SequenceNumber: aws.String("some_sequence_number2")}
	getRecordsOutput := &kinesis.GetRecordsOutput{NextShardIterator: aws.String("some_shard_iterator"), Records: []*kinesis.Record{record}}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(getShardIteratorOutput, nil)
	kinesisAPI.On("GetRecordsWithContext", ctx, getRecordsInput).Return(getRecordsOutput, nil)
	checkpoint.On("Set", r.checkpointIdentifier(), "some_sequence_number2").Return(errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get records")
}

func TestRunner_Process_HandlesWithSuccessAfterRecordStrategy(t *testing.T) {
	RegisterTestingT(t)

	// Assign
	ctx := context.TODO()
	checkpoint := &mocks.Checkpoint{}
	kinesisAPI := &mocks.KinesisAPI{}
	r := runner{
		client:             kinesisAPI,
		checkpoint:         checkpoint,
		checkpointStrategy: AfterRecord,
		stream:             "some_stream",
		group:              "some_group",
		tick:               time.Hour,
		handler:            func(Message) error { return nil },
	}
	getShardIteratorInput := &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(r.shardID),
		StreamName:             aws.String(r.stream),
		ShardIteratorType:      aws.String(kinesis.ShardIteratorTypeAfterSequenceNumber),
		StartingSequenceNumber: aws.String("some_sequence_number"),
	}
	getShardIteratorOutput := &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("some_shard_iterator")}
	getRecordsInput := &kinesis.GetRecordsInput{
		ShardIterator: getShardIteratorOutput.ShardIterator,
	}
	record := &kinesis.Record{PartitionKey: aws.String("some_partition"), Data: []byte("some_data"), SequenceNumber: aws.String("some_sequence_number2")}
	getRecordsOutput := &kinesis.GetRecordsOutput{NextShardIterator: aws.String("some_shard_iterator"), Records: []*kinesis.Record{record}}
	checkpoint.On("Get", r.checkpointIdentifier()).Return("some_sequence_number", nil)
	kinesisAPI.On("GetShardIteratorWithContext", ctx, getShardIteratorInput).Return(getShardIteratorOutput, nil)
	kinesisAPI.On("GetRecordsWithContext", ctx, getRecordsInput).Return(getRecordsOutput, nil)
	checkpoint.On("Set", r.checkpointIdentifier(), "some_sequence_number2").Return(errors.New("something failed"))

	// Act
	err := r.process(ctx)

	// Assert
	Expect(err).ToNot(HaveOccurred())
	Expect(kinesisAPI.AssertExpectations(t)).To(BeTrue(), "Should try to get records")
}
