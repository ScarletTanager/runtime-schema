package cb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cloudfoundry-incubator/auctioneer"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/tedsuo/rata"
)

//go:generate counterfeiter . AuctioneerClient
type AuctioneerClient interface {
	RequestLRPAuctions(auctioneerURL string, lrpStart []models.LRPStartRequest) error
	RequestTaskAuctions(auctioneerURL string, tasks []models.Task) error
}

type auctioneerClient struct {
	httpClient *http.Client
}

func NewAuctioneerClient() AuctioneerClient {
	return &auctioneerClient{
		httpClient: &http.Client{},
	}
}

func (c *auctioneerClient) RequestLRPAuctions(auctioneerURL string, lrpStarts []models.LRPStartRequest) error {
	reqGen := rata.NewRequestGenerator(auctioneerURL, auctioneer.Routes)

	payload, err := json.Marshal(lrpStarts)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(auctioneer.CreateLRPAuctionsRoute, rata.Params{}, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}

func (c *auctioneerClient) RequestTaskAuctions(auctioneerURL string, tasks []models.Task) error {
	reqGen := rata.NewRequestGenerator(auctioneerURL, auctioneer.Routes)

	payload, err := json.Marshal(tasks)
	if err != nil {
		return err
	}

	req, err := reqGen.CreateRequest(auctioneer.CreateTaskAuctionsRoute, rata.Params{}, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("http error: status code %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	return nil
}
