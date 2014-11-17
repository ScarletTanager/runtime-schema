package models_test

import (
	"encoding/json"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LRPStartAuction", func() {
	var startAuction LRPStartAuction

	startAuctionPayload := `{
    "desired_lrp": {
      "process_guid": "some-guid",
			"domain": "tests",
      "instances": 1,
      "stack": "some-stack",
      "root_fs": "docker:///docker.com/docker",
      "action": {
        "action": "download",
        "args": {
          "from": "http://example.com",
          "to": "/tmp/internet",
          "cache_key": ""
        }
      },
      "disk_mb": 512,
      "memory_mb": 1024,
      "cpu_weight": 42,
      "ports": [
        5678
      ],
      "routes": [
        "route-1",
        "route-2"
      ],
      "log_guid": "log-guid",
      "log_source": "the cloud"
    },
    "instance_guid": "some-instance-guid",
    "index": 2,
    "state": 1,
    "updated_at": 1138
  }`

	BeforeEach(func() {
		startAuction = LRPStartAuction{
			Index:        2,
			InstanceGuid: "some-instance-guid",

			DesiredLRP: DesiredLRP{
				Domain:      "tests",
				ProcessGuid: "some-guid",

				RootFSPath: "docker:///docker.com/docker",
				Instances:  1,
				Stack:      "some-stack",
				MemoryMB:   1024,
				DiskMB:     512,
				CPUWeight:  42,
				Routes:     []string{"route-1", "route-2"},
				Ports: []uint32{
					5678,
				},
				LogGuid:   "log-guid",
				LogSource: "the cloud",
				Action: ExecutorAction{
					Action: DownloadAction{
						From: "http://example.com",
						To:   "/tmp/internet",
					},
				},
			},

			State:     LRPStartAuctionStatePending,
			UpdatedAt: 1138,
		}
	})

	Describe("ToJSON", func() {

		It("should JSONify", func() {
			json := startAuction.ToJSON()
			Ω(string(json)).Should(MatchJSON(startAuctionPayload))
		})
	})

	Describe("JSON", func() {
		It("should not error with a blank auction", func() {
			blankAuction := LRPStartAuction{}
			jsonBytes, err := json.Marshal(blankAuction)
			Ω(err).ShouldNot(HaveOccurred())

			err = json.Unmarshal(jsonBytes, &blankAuction)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(blankAuction).Should(BeZero())
		})
	})

	Describe("NewLRPStartAuctionFromJSON", func() {
		It("returns a LRP with correct fields", func() {
			decodedStartAuction, err := NewLRPStartAuctionFromJSON([]byte(startAuctionPayload))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedStartAuction).Should(Equal(startAuction))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedStartAuction, err := NewLRPStartAuctionFromJSON([]byte("aliens lol"))
				Ω(err).Should(HaveOccurred())

				Ω(decodedStartAuction).Should(BeZero())
			})
		})

		for field, payload := range map[string]string{
			"instance_guid": `{"process_guid": "process-guid", "index": 0}`,
		} {
			json := payload
			missingField := field

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedStartAuction, err := NewLRPStartAuctionFromJSON([]byte(json))
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(Equal("JSON has missing/invalid field: " + missingField))

					Ω(decodedStartAuction).Should(BeZero())
				})
			})
		}
	})
})
