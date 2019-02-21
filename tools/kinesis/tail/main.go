package tail

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"os"
	"sync/atomic"

	"github.com/pkg/errors"

	"gitlab.com/marcoxavier/supervisor"

	logger "gitlab.com/vredens/go-logger"

	"gitlab.com/marcoxavier/go-kinesis/checkpoint/memory"

	kinesis "gitlab.com/marcoxavier/go-kinesis"

	"github.com/spf13/cobra"
)

var (
	s         = supervisor.NewSupervisor()
	iteration int32

	log = logger.Spawn(logger.WithTags("consumer"))

	stream             string
	endpoint           string
	region             string
	number             int
	logging            bool
	gzipDecode         bool
	skiReshardingOrder bool
)

// Command creates a new command.
func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tail",
		Short: "The tail utility displays the contents of kinesis stream to the standard output, starting in the latest record.",
		RunE:  Run,
	}
	cmd.Flags().StringVarP(&stream, "stream", "s", "", "stream name")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "kinesis endpoint")
	cmd.Flags().StringVarP(&region, "region", "r", "", "aws region, by default it will use AWS_REGION from aws config")
	cmd.Flags().IntVarP(&number, "number", "n", 0, "number of messages to show")
	cmd.Flags().BoolVar(&logging, "logging", false, "enables logging, mute by default")
	cmd.Flags().BoolVar(&gzipDecode, "gzip", false, "enables gzip decoder")
	cmd.Flags().BoolVar(&skiReshardingOrder, "skip-resharding-order", false, "if enabled, consumer will skip ordering when resharding")

	return cmd
}

// Run runs kinesis tail
func Run(cmd *cobra.Command, args []string) error {
	if err := os.Setenv("AWS_SDK_LOAD_CONFIG", "1"); err != nil {
		return err
	}

	s.DisableLogger()

	config := kinesis.ConsumerConfig{
		Group:  "tail",
		Stream: stream,
		AWS: kinesis.AWSConfig{
			Endpoint: endpoint,
			Region:   region,
		},
	}

	var skiReshardingOrderOption = dumbConsumerOption
	if skiReshardingOrder {
		skiReshardingOrderOption = kinesis.SkipReshardingOrder
	}

	checkpoint := memory.NewCheckpoint()
	consumer, err := kinesis.NewConsumer(config, handler(), checkpoint,
		kinesis.WithCheckpointStrategy(kinesis.AfterRecordBatch),
		kinesis.SinceLatest(),
		skiReshardingOrderOption(),
	)
	if err != nil {
		return err
	}

	if logging {
		consumer.SetLogger(Log)
		s.SetLogger(Log)
	}

	s.AddRunner("kinesis-tail", consumer.Run)

	s.Start()

	return nil
}

// Log logs kinesis consumer.
func Log(level string, data map[string]interface{}, format string, args ...interface{}) {
	switch level {
	case kinesis.Debug:
		log.WithData(data).Debugf(format, args...)
	case kinesis.Info:
		log.WithData(data).Infof(format, args...)
	case kinesis.Error:
		log.WithData(data).Errorf(format, args...)
	}
}

func handler() kinesis.MessageHandler {
	var f = bufio.NewWriter(os.Stdout)

	return func(message kinesis.Message) error {
		if number != 0 && atomic.LoadInt32(&iteration) >= int32(number) {
			s.Shutdown()

			return nil
		}

		msg := message.Data
		if gzipDecode {
			reader, err := gzip.NewReader(bytes.NewBuffer(message.Data))
			if err != nil {
				return errors.Wrap(err, "failed to decode message")
			}

			msg, err = ioutil.ReadAll(reader)
			if err != nil {
				return errors.Wrap(err, "failed to read decoded message")
			}
		}

		f.WriteString(string(msg) + "\n")
		f.Flush()

		atomic.AddInt32(&iteration, 1)

		return nil
	}
}

func dumbConsumerOption() kinesis.ConsumerOption {
	return func(c *kinesis.ConsumerOptions) {}
}
