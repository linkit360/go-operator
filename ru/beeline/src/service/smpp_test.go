package service

import (
	//"fmt"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	inmem_client "github.com/linkit360/go-inmem/rpcclient"
	//"github.com/linkit360/go-operator/ru/beeline/src/config"
	rec "github.com/linkit360/go-utils/rec"
)

func init() {
	log.SetLevel(log.DebugLevel)
	if err := inmem_client.Init(inmem_client.ClientConfig{DSN: ":50307", Timeout: 10}); err != nil {
		log.WithField("error", err.Error()).Fatal("cannot init inmem client")
	}
}

func TestResolveRec(t *testing.T) {
	r := &rec.Record{
		Tid: "testtid",
	}

	dstAddr := "8580#3"
	s, err := resolveRec(dstAddr, r)
	assert.NoError(t, err, "resolveRec")
	assert.Equal(t, int64(777), s.Id, "service id")
	assert.Equal(t, int64(290), r.CampaignId, "campaign id")

	dstAddr = "8580"
	s, err = resolveRec(dstAddr, r)
	assert.NoError(t, err, "resolveRec")
	assert.Equal(t, int64(777), s.Id, "service id")
	assert.Equal(t, int64(290), r.CampaignId, "campaign id")

	dstAddr = "858001"
	s, err = resolveRec(dstAddr, r)
	assert.NoError(t, err, "resolveRec")
	assert.Equal(t, int64(777), s.Id, "service id")
	assert.Equal(t, int64(290), r.CampaignId, "campaign id")

	dstAddr = "85801\x00"
	s, err = resolveRec(dstAddr, r)
	assert.NoError(t, err, "resolveRec")
	assert.Equal(t, int64(777), s.Id, "service id")
	assert.Equal(t, int64(290), r.CampaignId, "campaign id")
}
