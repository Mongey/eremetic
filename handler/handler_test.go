package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alde/eremetic/types"
	"github.com/gorilla/mux"
	mesos "github.com/mesos/mesos-go/mesosproto"
	. "github.com/smartystreets/goconvey/convey"
)

type mockError struct {
	message string
}

func (m mockError) Error() string {
	return m.message
}

func TestHandling(t *testing.T) {
	Convey("createID", t, func() {
		Convey("Given a string", func() {
			Convey("It should build the appropriate ID", func() {
				So(createID("1234"), ShouldEqual, "eremetic-task.1234")
			})
		})
	})

	Convey("writeJSON", t, func() {
		Convey("Should respond with a JSON and the appropriate status code", func() {
			var wr = httptest.NewRecorder()

			writeJSON(200, "foo", wr)
			contentType := wr.HeaderMap["Content-Type"][0]
			So(contentType, ShouldEqual, "application/json; charset=UTF-8")
			So(wr.Code, ShouldEqual, http.StatusOK)
		})
	})

	Convey("HandleError", t, func() {
		wr := httptest.NewRecorder()

		Convey("It should return an error status code", func() {
			var err = mockError{
				message: "Error",
			}

			handleError(err, wr)

			So(wr.Code, ShouldEqual, 422)
			So(strings.TrimSpace(wr.Body.String()), ShouldEqual, "{}")
		})
	})

	Convey("createRequest", t, func() {
		wr := httptest.NewRecorder()
		Convey("Given a valid Request", func() {
			request := types.Request{
				TaskCPUs:    0.5,
				TaskMem:     22.0,
				DockerImage: "busybox",
				Command:     "echo hello",
			}

			Convey("It should put the request on the channel", func() {
				createRequest(request, wr)

				location := wr.HeaderMap["Location"][0]
				So(location, ShouldStartWith, "/task/eremetic-task.")
				So(wr.Code, ShouldEqual, http.StatusAccepted)

				select {
				case c := <-requests:
					So(c.TaskCPUs, ShouldEqual, request.TaskCPUs)
					So(c.TaskMem, ShouldEqual, request.TaskMem)
					So(c.Command, ShouldEqual, request.Command)
					So(c.DockerImage, ShouldEqual, request.DockerImage)
					So(c.TaskID, ShouldStartWith, "eremetic-task.")
				}
			})
		})
	})

	Convey("GetTaskInfo", t, func() {
		wr := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/task/eremetic-task.1234", nil)
		m := mux.NewRouter()
		m.HandleFunc("/task/{taskId}", GetTaskInfo)
		runningTasks = make(map[string]eremeticTask)

		Convey("Not Found", func() {
			id := "eremetic-task.5678"
			runningTasks[id] = eremeticTask{
				TaskCPUs:  0.2,
				TaskMem:   0.5,
				Command:   &mesos.CommandInfo{},
				Container: &mesos.ContainerInfo{},
				Status:    "TASK_RUNNING",
				ID:        id,
				deleteAt:  time.Now(),
			}
			m.ServeHTTP(wr, r)

			So(wr.Code, ShouldEqual, http.StatusNotFound)
		})

		Convey("Found", func() {
			id := "eremetic-task.1234"
			runningTasks[id] = eremeticTask{
				TaskCPUs:  0.2,
				TaskMem:   0.5,
				Command:   &mesos.CommandInfo{},
				Container: &mesos.ContainerInfo{},
				Status:    "TASK_RUNNING",
				ID:        id,
				deleteAt:  time.Now(),
			}
			m.ServeHTTP(wr, r)

			So(wr.Code, ShouldEqual, http.StatusOK)
		})
	})

	Convey("AddTask", t, func() {
		wr := httptest.NewRecorder()

		Convey("It should respond with a location header", func() {
			data := []byte(`{"task_mem":22.0, "docker_image": "busybox", "command": "echo hello", "task_cpus":0.5, "tasks_to_launch": 1}`)
			r, _ := http.NewRequest("POST", "/task", bytes.NewBuffer(data))

			AddTask(wr, r)

			location := wr.HeaderMap["Location"][0]
			So(location, ShouldStartWith, "/task/eremetic-task.")
			So(wr.Code, ShouldEqual, http.StatusAccepted)
		})
	})
}
