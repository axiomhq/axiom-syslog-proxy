package syslog

import (
	"math/rand"
	"testing"
	"time"

	"axicode.axiom.co/watchmakers/watchly/pkg/common/util"

	"github.com/stretchr/testify/suite"
)

type QueueTestSuite struct {
	suite.Suite
	queue   *Queue
	orgDocs []map[string]interface{}
}

func (suite *QueueTestSuite) SetupTest() {
	numDocs := 105
	queue := NewQueue(100)
	for i := 0; i < numDocs; i++ {
		ev := map[string]interface{}{
			"systemTimestamp": time.Now().UnixNano(),
			"serverity":       int64(rand.Int31()),
			"application":     util.RandString(rand.Int() % 255),
			"hostname":        util.RandString(rand.Int() % 255),
			"text":            util.RandString(rand.Int() % 1000),
		}

		for i := 0; i < rand.Int()%255; i++ {
			ev["metdata."+util.RandString(rand.Int()%255)] = util.RandString(rand.Int() % 255)
		}
		suite.orgDocs = append(suite.orgDocs, ev)
		queue.Push([]map[string]interface{}{ev})
	}
	suite.queue = queue
}

func (suite *QueueTestSuite) TearDownTest() {
	suite.orgDocs = nil
}

func TestQueueTestSuite(t *testing.T) {
	suite.Run(t, new(QueueTestSuite))
}

func (suite *QueueTestSuite) TestPushGet() {
	docs := suite.queue.Get()
	suite.Len(docs, 100)
	suite.Equal(docs, suite.orgDocs[:100])

	docs = suite.queue.Get()
	suite.Len(docs, 5)
	suite.Equal(docs, suite.orgDocs[100:])

	docs = suite.queue.Get()
	suite.Len(docs, 0)
}

func (suite *QueueTestSuite) TestMaxItems() {
	sizedQueue := NewQueueWithMax(25, 50)
	suite.Require().NotNil(sizedQueue)

	events := suite.queue.GetN(80)
	suite.Require().Len(events, 80)

	newSize, dropped := sizedQueue.Push(events)
	suite.EqualValues(50, newSize)
	suite.EqualValues(30, dropped)

	newSize, dropped = sizedQueue.Push(events)
	suite.EqualValues(50, newSize)
	suite.EqualValues(80, dropped)

	events = sizedQueue.GetN(25)
	suite.Len(events, 25)
	suite.EqualValues(25, sizedQueue.size())

	suite.Equal(suite.orgDocs[:25], events)
}
