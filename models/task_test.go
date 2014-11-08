package models_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry-incubator/runtime-schema/models"
)

var _ = Describe("Task", func() {
	var task Task

	taskPayload := `{
		"task_guid":"some-guid",
		"domain":"some-domain",
		"root_fs": "docker:///docker.com/docker",
		"stack":"some-stack",
		"env":[
			{
				"name":"ENV_VAR_NAME",
				"value":"an environmment value"
			}
		],
		"executor_id":"executor",
		"actions":[
			{
				"action":"download",
				"args":{
					"from":"old_location",
					"to":"new_location",
					"cache_key":"the-cache-key"
				}
			}
		],
		"privileged":true,
		"result_file":"some-file.txt",
		"result": "turboencabulated",
		"failed":true,
		"failure_reason":"because i said so",
		"memory_mb":256,
		"disk_mb":1024,
		"cpu_weight": 42,
		"log": {
			"guid": "123",
			"source_name": "APP"
		},
		"created_at": 1393371971000000000,
		"updated_at": 1393371971000000010,
		"first_completed_at": 1393371971000000030,
		"state": 1,
		"annotation": "[{\"anything\": \"you want!\"}]... dude"
	}`

	BeforeEach(func() {
		task = Task{
			TaskGuid:   "some-guid",
			Domain:     "some-domain",
			RootFSPath: "docker:///docker.com/docker",
			Stack:      "some-stack",
			EnvironmentVariables: []EnvironmentVariable{
				{
					Name:  "ENV_VAR_NAME",
					Value: "an environmment value",
				},
			},
			Actions: []ExecutorAction{
				{
					Action: DownloadAction{
						From:     "old_location",
						To:       "new_location",
						CacheKey: "the-cache-key",
					},
				},
			},
			Log: LogConfig{
				Guid:       "123",
				SourceName: "APP",
			},
			Privileged:       true,
			ExecutorID:       "executor",
			ResultFile:       "some-file.txt",
			Result:           "turboencabulated",
			Failed:           true,
			FailureReason:    "because i said so",
			MemoryMB:         256,
			DiskMB:           1024,
			CPUWeight:        42,
			CreatedAt:        time.Date(2014, time.February, 25, 23, 46, 11, 00, time.UTC).UnixNano(),
			UpdatedAt:        time.Date(2014, time.February, 25, 23, 46, 11, 10, time.UTC).UnixNano(),
			FirstCompletedAt: time.Date(2014, time.February, 25, 23, 46, 11, 30, time.UTC).UnixNano(),
			State:            TaskStatePending,
			Annotation:       `[{"anything": "you want!"}]... dude`,
		}
	})

	Describe("ToJSON", func() {
		It("should JSONify", func() {
			json := task.ToJSON()
			Ω(string(json)).Should(MatchJSON(taskPayload))
		})
	})

	Describe("NewTaskFromJSON", func() {
		It("returns a Task with correct fields", func() {
			decodedTask, err := NewTaskFromJSON([]byte(taskPayload))
			Ω(err).ShouldNot(HaveOccurred())

			Ω(decodedTask).Should(Equal(task))
		})

		Context("with an invalid payload", func() {
			It("returns the error", func() {
				decodedTask, err := NewTaskFromJSON([]byte("aliens lol"))
				Ω(err).Should(HaveOccurred())

				Ω(decodedTask).Should(BeZero())
			})
		})

		for field, payload := range map[string]string{
			"task_guid": `{"domain": "some-domain", "stack": "some-stack", "actions": [{"action": "run", "args": {"path": "date"}}]}`,
			"actions":   `{"domain": "some-domain", "task_guid": "process-guid", "stack": "some-stack"}`,
			"stack":     `{"domain": "some-domain", "task_guid": "process-guid", "actions": [{"action": "run", "args": {"path": "date"}}]}`,
			"domain":    `{"stack": "some-stack", "task_guid": "process-guid", "actions": [{"action": "run", "args": {"path": "date"}}]}`,
		} {
			json := payload
			missingField := field

			Context("when the json is missing a "+missingField, func() {
				It("returns an error indicating so", func() {
					decodedStartAuction, err := NewTaskFromJSON([]byte(json))
					Ω(err).Should(HaveOccurred())
					Ω(err.Error()).Should(Equal("JSON has missing/invalid field: " + missingField))

					Ω(decodedStartAuction).Should(BeZero())
				})
			})
		}

		Context("when the task GUID is present but invalid", func() {
			json := `{"domain": "some-domain", "task_guid": "invalid/guid", "stack": "some-stack", "actions": [{"action": "run", "args": {"path": "date"}}]}`

			It("returns an error indicating so", func() {
				decodedStartAuction, err := NewTaskFromJSON([]byte(json))
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(Equal("JSON has missing/invalid field: task_guid"))

				Ω(decodedStartAuction).Should(BeZero())
			})
		})

		Context("with an invalid CPU weight", func() {
			json := `{"domain": "some-domain", "task_guid": "guid", "cpu_weight": 101, "stack": "some-stack", "actions": [{"action": "run", "args": {"path": "date"}}]}`

			It("returns an error", func() {
				decodedStartAuction, err := NewTaskFromJSON([]byte(json))
				Ω(err).Should(HaveOccurred())
				Ω(err.Error()).Should(Equal("JSON has missing/invalid field: cpu_weight"))

				Ω(decodedStartAuction).Should(BeZero())

			})
		})
	})
})
