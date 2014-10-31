package models_test

import (
	"fmt"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DesiredLRP", func() {
	var lrp DesiredLRP

	lrpPayload := `{
	  "process_guid": "some-guid",
	  "domain": "some-domain",
	  "root_fs": "docker:///docker.com/docker",
	  "instances": 1,
	  "stack": "some-stack",
	  "env":[
	    {
	      "name": "ENV_VAR_NAME",
	      "value": "some environment variable value"
	    }
	  ],
	  "actions": [
	    {
	      "action": "download",
	      "args": {
	        "from": "http://example.com",
	        "to": "/tmp/internet",
	        "cache_key": ""
	      }
	    }
	  ],
	  "disk_mb": 512,
	  "memory_mb": 1024,
	  "cpu_weight": 42,
	  "ports": [
	    {
	      "container_port": 5678,
	      "host_port": 1234
	    }
	  ],
	  "routes": [
	    "route-1",
	    "route-2"
	  ],
	  "log": {
	    "guid": "log-guid",
	    "source_name": "the cloud"
	  }
	}`

	BeforeEach(func() {
		lrp = DesiredLRP{
			Domain:      "some-domain",
			ProcessGuid: "some-guid",

			Instances:  1,
			Stack:      "some-stack",
			RootFSPath: "docker:///docker.com/docker",
			MemoryMB:   1024,
			DiskMB:     512,
			CPUWeight:  42,
			Routes:     []string{"route-1", "route-2"},
			Ports: []PortMapping{
				{HostPort: 1234, ContainerPort: 5678},
			},
			Log: LogConfig{
				Guid:       "log-guid",
				SourceName: "the cloud",
			},
			EnvironmentVariables: []EnvironmentVariable{
				{
					Name:  "ENV_VAR_NAME",
					Value: "some environment variable value",
				},
			},
			Actions: []ExecutorAction{
				{
					Action: DownloadAction{
						From: "http://example.com",
						To:   "/tmp/internet",
					},
				},
			},
		}
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json := lrp.ToJSON()
			Ω(string(json)).Should(MatchJSON(lrpPayload))
		})
	})

	Describe("Validate", func() {
		var assertDesiredLRPValidationFailsWithMessage = func(lrp DesiredLRP, substring string) {
			validationErr := lrp.Validate()
			Ω(validationErr).Should(HaveOccurred())
			Ω(validationErr.Error()).Should(ContainSubstring(substring))
		}

		Context("process_guid only contains `A-Z`, `a-z`, `0-9`, `-`, and `_`", func() {
			validGuids := []string{"a", "A", "0", "-", "_", "-aaaa", "_-aaa", "09a87aaa-_aASKDn"}
			for _, validGuid := range validGuids {
				func(validGuid string) {
					It(fmt.Sprintf("'%s' is a valid process_guid", validGuid), func() {
						lrp.ProcessGuid = validGuid
						err := lrp.Validate()
						Ω(err).ShouldNot(HaveOccurred())
					})
				}(validGuid)
			}

			invalidGuids := []string{"", "bang!", "!!!", "\\slash", "star*", "params()", "invalid/key", "with.dots"}
			for _, invalidGuid := range invalidGuids {
				func(invalidGuid string) {
					It(fmt.Sprintf("'%s' is an invalid process_guid", invalidGuid), func() {
						lrp.ProcessGuid = invalidGuid
						assertDesiredLRPValidationFailsWithMessage(lrp, "process_guid")
					})
				}(invalidGuid)
			}
		})

		It("requires a positive number of instances", func() {
			lrp.Instances = 0
			assertDesiredLRPValidationFailsWithMessage(lrp, "instances")
		})

		It("requires a domain", func() {
			lrp.Domain = ""
			assertDesiredLRPValidationFailsWithMessage(lrp, "domain")
		})

		It("requires a stack", func() {
			lrp.Stack = ""
			assertDesiredLRPValidationFailsWithMessage(lrp, "stack")
		})

		It("requires actions", func() {
			lrp.Actions = nil
			assertDesiredLRPValidationFailsWithMessage(lrp, "actions")
		})
	})

	Describe("NewDesiredLRPFromJSON", func() {
		It("returns a LRP with correct fields", func() {
			decodedStartAuction, err := NewDesiredLRPFromJSON([]byte(lrpPayload))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedStartAuction).Should(Equal(lrp))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedStartAuction, err := NewDesiredLRPFromJSON([]byte("aliens lol"))
				Ω(err).Should(HaveOccurred())

				Ω(decodedStartAuction).Should(BeZero())
			})
		})

		for field, payload := range map[string]string{
			"process_guid": `{
				"domain": "some-domain",
				"actions": [
					{"action":"download","args":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
				],
				"stack": "some-stack"
			}`,
			"actions": `{
				"domain": "some-domain",
				"process_guid": "process_guid",
				"stack": "some-stack"
			}`,
			"stack": `{
				"domain": "some-domain",
				"process_guid": "process_guid",
				"actions": [
					{"action":"download","args":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
				]
			}`,
			"domain": `{
				"stack": "some-stack",
				"process_guid": "process_guid",
				"actions": [
					{"action":"download","args":{"from":"http://example.com","to":"/tmp/internet","cache_key":""}}
				]
			}`,
		} {
			json := payload
			missingField := field

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedStartAuction, err := NewDesiredLRPFromJSON([]byte(json))
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(Equal("JSON has missing/invalid field: " + missingField))

					Ω(decodedStartAuction).Should(BeZero())
				})
			})
		}
	})
})
